/**
 * Cloudflare Worker - PostHog Metrics & Reverse Proxy
 * Securely proxies PostHog API requests, events, assets, and aggregates metrics
 */

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // CORS headers
    const corsHeaders = {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
      'Access-Control-Allow-Credentials': 'true'
    };

    // Handle preflight
    if (request.method === 'OPTIONS') {
      return new Response(null, { headers: corsHeaders });
    }

    // Route: GET /metrics (returns HTML for HTMX)
    if (url.pathname === '/metrics' && request.method === 'GET') {
      try {
        const posthogKey = env.POSTHOG_API_KEY;
        const projectId = env.POSTHOG_PROJECT_ID;

        if (!posthogKey || !projectId) {
          return new Response(
            `<div class="error">⚠️ Missing PostHog configuration</div>`,
            { status: 500, headers: { ...corsHeaders, 'Content-Type': 'text/html' } }
          );
        }

        // Fetch events from PostHog API
        const eventsUrl = `https://us.posthog.com/api/projects/${projectId}/events/?limit=500`;
        const eventsResponse = await fetch(eventsUrl, {
          headers: {
            'Authorization': `Bearer ${posthogKey}`,
            'Content-Type': 'application/json'
          }
        });

        if (!eventsResponse.ok) {
          throw new Error(`PostHog API error: ${eventsResponse.status}`);
        }

        const eventsData = await eventsResponse.json();
        const events = eventsData.results || [];

        // Aggregate metrics
        const metrics = aggregateMetrics(events);

        // Render as HTML
        const html = renderMetricsHTML(metrics);

        return new Response(html, {
          status: 200,
          headers: { ...corsHeaders, 'Content-Type': 'text/html' }
        });
      } catch (error) {
        console.error('Worker error:', error);
        return new Response(
          `<div class="error">⚠️ Unable to load metrics<br><small>${error.message}</small></div>`,
          { status: 500, headers: { ...corsHeaders, 'Content-Type': 'text/html' } }
        );
      }
    }

    // Route: GET /health
    if (url.pathname === '/health' && request.method === 'GET') {
      return new Response(JSON.stringify({ status: 'ok' }), {
        headers: { ...corsHeaders, 'Content-Type': 'application/json' }
      });
    }

    // PostHog Reverse Proxy Routes
    // Handles JS SDK assets (/array/*, /static/*), event tracking (/e/*, /capture, /s/*), and flags (/flags/*, /decide)
    if (
      url.pathname.startsWith('/static/') ||
      url.pathname.startsWith('/array/') ||
      url.pathname.startsWith('/e/') ||
      url.pathname.startsWith('/s/') ||
      url.pathname.startsWith('/flags/') ||
      url.pathname.startsWith('/decide') ||
      url.pathname.startsWith('/capture') ||
      url.pathname.startsWith('/engage') ||
      url.pathname.startsWith('/i/')
    ) {
      return handlePostHogProxy(request, url);
    }

    // Fallback 404
    return new Response(
      JSON.stringify({ error: 'Not found' }),
      { status: 404, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
    );
  }
};

/**
 * Forward PostHog requests to official PostHog endpoints
 */
async function handlePostHogProxy(request, url) {
  const isAsset = url.pathname.startsWith('/static/') || url.pathname.startsWith('/array/');
  const targetHost = isAsset ? 'us-assets.i.posthog.com' : 'us.i.posthog.com';

  const proxyUrl = new URL(url.toString());
  proxyUrl.host = targetHost;
  proxyUrl.protocol = 'https:';

  const headers = new Headers(request.headers);
  headers.set('host', targetHost);
  if (request.headers.get('x-forwarded-for')) {
    headers.set('x-forwarded-for', request.headers.get('x-forwarded-for'));
  }

  const init = {
    method: request.method,
    headers: headers,
    redirect: 'follow'
  };

  if (request.method !== 'GET' && request.method !== 'HEAD') {
    init.body = await request.clone().arrayBuffer();
  }

  try {
    const response = await fetch(proxyUrl.toString(), init);
    const responseHeaders = new Headers(response.headers);
    responseHeaders.set('Access-Control-Allow-Origin', '*');
    responseHeaders.set('Access-Control-Allow-Credentials', 'true');

    if (isAsset) {
      responseHeaders.set('Cache-Control', 'public, max-age=86400');
    }

    return new Response(response.body, {
      status: response.status,
      headers: responseHeaders
    });
  } catch (error) {
    console.error('PostHog proxy error:', error);
    return new Response(JSON.stringify({ error: 'Proxy failed' }), {
      status: 502,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Content-Type': 'application/json'
      }
    });
  }
}

/**
 * Aggregate PostHog events into meaningful metrics
 */
function aggregateMetrics(events) {
  const metrics = {
    totalBoops: 0,
    totalEvents: events.length,
    featureToggles: {
      eyeTracking: 0,
      googlyEyes: 0,
      partyMode: 0,
      treats: 0
    },
    eventBreakdown: {},
    topCombo: 0,
    lastUpdated: new Date().toISOString(),
    eventSamples: {
      boops: [],
      featureToggles: [],
      consoleCmds: []
    }
  };

  events.forEach(event => {
    const eventName = event.event;
    const properties = event.properties || {};

    // Count total events by type
    metrics.eventBreakdown[eventName] = (metrics.eventBreakdown[eventName] || 0) + 1;

    // Boop tracking
    if (eventName === 'boop') {
      metrics.totalBoops++;
      const comboCount = properties.combo_count || 0;
      if (comboCount > metrics.topCombo) {
        metrics.topCombo = comboCount;
      }
      metrics.eventSamples.boops.push({
        timestamp: event.timestamp,
        combo: comboCount,
        via: properties.via || 'unknown'
      });
    }

    // Feature toggle tracking
    if (eventName === 'feature_toggle') {
      const feature = properties.feature;
      const enabled = properties.enabled;

      if (enabled) {
        if (feature === 'eye_tracking') metrics.featureToggles.eyeTracking++;
        else if (feature === 'googly_eyes') metrics.featureToggles.googlyEyes++;
        else if (feature === 'party_mode') metrics.featureToggles.partyMode++;
      }

      metrics.eventSamples.featureToggles.push({
        feature,
        enabled,
        timestamp: event.timestamp
      });
    }

    // Treat tracking
    if (eventName === 'treat_shower') {
      metrics.featureToggles.treats++;
    }

    // Console commands
    if (eventName === 'console_command') {
      metrics.eventSamples.consoleCmds.push({
        command: properties.command,
        timestamp: event.timestamp
      });
    }
  });

  // Keep only recent samples for display
  metrics.eventSamples.boops = metrics.eventSamples.boops.slice(-10);
  metrics.eventSamples.featureToggles = metrics.eventSamples.featureToggles.slice(-10);
  metrics.eventSamples.consoleCmds = metrics.eventSamples.consoleCmds.slice(-5);

  return metrics;
}

/**
 * Render metrics as HTML (for HTMX)
 */
function renderMetricsHTML(metrics) {
  const { totalBoops, totalEvents, featureToggles, topCombo, eventBreakdown, lastUpdated } = metrics;

  let html = `
    <div class="metric-row">
      <span class="metric-label">Total Boops</span>
      <span class="metric-value">${totalBoops}</span>
    </div>
    <div class="metric-row">
      <span class="metric-label">Total Events Tracked</span>
      <span class="metric-value">${totalEvents}</span>
    </div>
    <div class="metric-row">
      <span class="metric-label">Max Boop Combo</span>
      <span class="metric-value">×${topCombo}</span>
    </div>

    <h3>🎮 Feature Toggles</h3>
    <div class="feature-grid">
      <div class="feature-box">
        <div class="feature-name">Eye Tracking</div>
        <div class="feature-count">${featureToggles.eyeTracking}</div>
      </div>
      <div class="feature-box">
        <div class="feature-name">Googly Eyes</div>
        <div class="feature-count">${featureToggles.googlyEyes}</div>
      </div>
      <div class="feature-box">
        <div class="feature-name">Party Mode</div>
        <div class="feature-count">${featureToggles.partyMode}</div>
      </div>
      <div class="feature-box">
        <div class="feature-name">Treats</div>
        <div class="feature-count">${featureToggles.treats}</div>
      </div>
    </div>

    <h3>📈 Event Breakdown</h3>
  `;

  Object.entries(eventBreakdown).forEach(([event, count]) => {
    html += `
      <div class="metric-row">
        <span class="metric-label">${event}</span>
        <span class="metric-value">${count}</span>
      </div>
    `;
  });

  html += `
    <div style="margin-top: 16px; padding-top: 12px; border-top: 1px solid rgba(0, 212, 255, 0.2); font-size: 0.8rem; color: #666;">
      Last updated: ${new Date(lastUpdated).toLocaleTimeString()}
    </div>
  `;

  return html;
}