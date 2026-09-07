/**
 * Cloudflare Worker - PostHog Metrics & Reverse Proxy
 * Securely proxies PostHog API requests, events, assets, and aggregates metrics
 */

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    const origin = request.headers.get('Origin');
    const allowedOrigins = [
      'https://jbirdkerr.net',
      'https://www.jbirdkerr.net',
      'https://jbirdkerr.github.io'
    ];

    // If an origin is passed by a browser and it's not allowed, block cross-origin access
    const isAllowedOrigin = !origin || allowedOrigins.includes(origin);
    const corsOrigin = isAllowedOrigin ? (origin || 'https://jbirdkerr.net') : 'https://jbirdkerr.net';

    // CORS headers
    const corsHeaders = {
      'Access-Control-Allow-Origin': corsOrigin,
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization, HX-Request, HX-Current-URL, HX-Target, HX-Trigger, HX-Trigger-Name',
      'Access-Control-Allow-Credentials': 'true'
    };

    // Reject unauthorized cross-origin requests from foreign browser domains
    if (origin && !isAllowedOrigin && request.method !== 'OPTIONS') {
      return new Response(JSON.stringify({ error: 'Unauthorized origin' }), {
        status: 403,
        headers: { ...corsHeaders, 'Content-Type': 'application/json' }
      });
    }

    // Handle preflight
    if (request.method === 'OPTIONS') {
      return new Response(null, { headers: corsHeaders });
    }

    // Route: GET /metrics (returns HTML for HTMX with edge caching)
    if (url.pathname === '/metrics' && request.method === 'GET') {
      const cache = caches.default;
      const cacheKey = new Request(url.origin + url.pathname); // URL-only key, no request headers
      let cachedResponse = await cache.match(cacheKey);

      if (cachedResponse) {
        return cachedResponse;
      }

      try {
        const posthogKey = env.POSTHOG_API_KEY;
        const projectId = env.POSTHOG_PROJECT_ID;

        if (!posthogKey || !projectId) {
          return new Response(
            `<div class="error">⚠️ Missing PostHog configuration</div>`,
            { status: 500, headers: { ...corsHeaders, 'Content-Type': 'text/html' } }
          );
        }

        // Query PostHog ClickHouse database using HogQL for instant all-time stats
        const queryUrl = `https://us.posthog.com/api/projects/${projectId}/query/`;
        const hogqlQuery = {
          query: {
            kind: 'HogQLQuery',
            query: `
              SELECT
                countIf(event = 'boop') as total_boops,
                max(if(event = 'boop', toFloat(JSONExtractRaw(properties, 'combo_count')), 0)) as top_combo,
                count() as total_events,
                countIf(event = 'feature_toggle' and JSONExtractRaw(properties, 'feature') = '"eye_tracking"' and JSONExtractRaw(properties, 'enabled') = 'true') as eye_tracking,
                countIf(event = 'feature_toggle' and JSONExtractRaw(properties, 'feature') = '"googly_eyes"' and JSONExtractRaw(properties, 'enabled') = 'true') as googly_eyes,
                countIf(event = 'feature_toggle' and JSONExtractRaw(properties, 'feature') = '"party_mode"' and JSONExtractRaw(properties, 'enabled') = 'true') as party_mode,
                countIf(event = 'treat_shower') as treats,
                sum(if(event = 'eye_movement', toFloat(JSONExtractRaw(properties, 'distance_px')), 0)) as total_eye_dist
              FROM events
            `
          }
        };

        const breakdownQuery = {
          query: {
            kind: 'HogQLQuery',
            query: `SELECT event, count() as cnt FROM events GROUP BY event ORDER BY cnt DESC LIMIT 10`
          }
        };

        // Fetch both aggregations in parallel
        const [statsResponse, breakdownResponse] = await Promise.all([
          fetch(queryUrl, {
            method: 'POST',
            headers: {
              'Authorization': `Bearer ${posthogKey}`,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify(hogqlQuery)
          }),
          fetch(queryUrl, {
            method: 'POST',
            headers: {
              'Authorization': `Bearer ${posthogKey}`,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify(breakdownQuery)
          })
        ]);

        if (!statsResponse.ok || !breakdownResponse.ok) {
          throw new Error(`PostHog API error: ${statsResponse.status || breakdownResponse.status}`);
        }

        const statsData = await statsResponse.json();
        const breakdownData = await breakdownResponse.json();

        const row = (statsData.results && statsData.results[0]) || [0, 0, 0, 0, 0, 0, 0, 0];
        const breakdownMap = {};
        (breakdownData.results || []).forEach(([evt, cnt]) => {
          breakdownMap[evt] = cnt;
        });

        const totalEyeDistancePx = Math.round(Number(row[7])) || 0;

        const metrics = {
          totalBoops: Number(row[0]) || 0,
          topCombo: Math.round(Number(row[1])) || 0,
          totalEvents: Number(row[2]) || 0,
          eyeDistancePx: totalEyeDistancePx,
          featureToggles: {
            eyeTracking: Number(row[3]) || 0,
            googlyEyes: Number(row[4]) || 0,
            partyMode: Number(row[5]) || 0,
            treats: Number(row[6]) || 0
          },
          eventBreakdown: breakdownMap,
          lastUpdated: new Date().toISOString()
        };

        // Render as HTML
        const html = renderMetricsHTML(metrics);

        const response = new Response(html, {
          status: 200,
          headers: {
            ...corsHeaders,
            'Content-Type': 'text/html',
            'Cache-Control': 'public, max-age=10, s-maxage=10'
          }
        });

        ctx.waitUntil(cache.put(cacheKey, response.clone()));
        return response;
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
      return handlePostHogProxy(request, url, corsHeaders);
    }

    // Fallback 404
    return new Response(
      JSON.stringify({ error: 'Not found' }),
      { status: 404, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
    );
  }
};

/**
 * Forward PostHog requests to official PostHog endpoints with CORS & Cache handling
 */
async function handlePostHogProxy(request, url, corsHeaders) {
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
    
    // Set CORS origin and credentials
    Object.entries(corsHeaders).forEach(([key, val]) => {
      responseHeaders.set(key, val);
    });

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
        ...corsHeaders,
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
  const { totalBoops, totalEvents, eyeDistancePx, featureToggles, topCombo, eventBreakdown, lastUpdated } = metrics;

  // Standard CSS pixels to estimated real-world distance (approx 3780 px per meter at 96 DPI)
  const distanceMeters = (eyeDistancePx / 3780).toFixed(2);

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
    <div class="metric-row">
      <span class="metric-label">👀 Eyeball Distance Rolled</span>
      <span class="metric-value">${distanceMeters} m <small style="font-size: 0.75rem; color: #888;">(${eyeDistancePx.toLocaleString()} px)</small></span>
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