# Observability Implementation Plan — jbirdkerr.net

## Context

This is a personal interactive website ([jbirdkerr.net](https://jbirdkerr.net)) featuring an interactive dog photo canvas with eye tracking, snoot booping, party mode, and treat showers. The stack is:

- **Frontend**: Static HTML/CSS/JS on GitHub Pages (`index.html`, `css/style.css`, `js/main.js`)
- **Edge Proxy**: Python Cloudflare Worker (`metrics-worker/worker.py`) deployed to `metrics-api.jbirdkerr.net`
- **Database**: Supabase PostgreSQL (tables: `site_events`, `site_event_counts`, `site_metrics`; RPC: `record_site_events`)
- **Analytics**: PostHog Cloud (proxied through the Cloudflare Worker to bypass ad blockers)
- **Live Dashboard**: HTMX modal in the frontend that polls `GET /metrics` every 5 seconds

The existing dashboard (`📊 Metrics` button) shows **product metrics** (boop counts, combos, eye distance, feature toggles). This plan adds **system observability** — performance, reliability, and health monitoring — across all tiers.

---

## Work Item 1: Enable PostHog Web Vitals Autocapture (Frontend)

### Goal
Capture Core Web Vitals (LCP, INP, CLS) and page performance data automatically via PostHog's built-in performance monitoring, with zero custom instrumentation.

### File to modify
`js/main.js` — lines 3–9 (the `posthog.init()` call)

### Changes

Update the PostHog init config object to enable performance capture. Change:

```javascript
posthog.init('phc_zugUWyUvq2VWid5q6XxqRtHnBXsoLDB9BhaaY6VxVuXJ', {
    api_host: 'https://metrics-api.jbirdkerr.net',
    ui_host: 'https://us.posthog.com',
    person_profiles: 'always',
    cross_subdomain_cookie: false,
    request_batching: true
});
```

to:

```javascript
posthog.init('phc_zugUWyUvq2VWid5q6XxqRtHnBXsoLDB9BhaaY6VxVuXJ', {
    api_host: 'https://metrics-api.jbirdkerr.net',
    ui_host: 'https://us.posthog.com',
    person_profiles: 'always',
    cross_subdomain_cookie: false,
    request_batching: true,
    capture_performance: {
        web_vitals: true,
        network_timing: true
    },
    autocapture: {
        capture_copied_text: false
    }
});
```

### What this gives you (for free in PostHog)
- **LCP** (Largest Contentful Paint) — how fast `henrybeard.png` renders
- **INP** (Interaction to Next Paint) — input latency on boop clicks, key presses
- **CLS** (Cumulative Layout Shift) — layout stability during eye/googly asset hydration
- **Network timing** — resource load waterfall for the PostHog SDK, HTMX, main.js, CSS, and the main image
- All viewable in PostHog's built-in **Web Analytics → Web Vitals** dashboard

### Verification
After deploying, open the site, interact with it, then check PostHog → Web Analytics → Web Vitals. You should see LCP/INP/CLS charts populating within minutes.

---

## Work Item 2: Instrument the Cloudflare Worker with Timing & Error Metrics

### Goal
Add lightweight request-level instrumentation to `worker.py` that tracks latencies, error rates, and payload decode outcomes. Persist these to a new Supabase table so the system health tab (Work Item 3) can display them.

### 2A: New Supabase migration — `worker_health_stats` table

Create a new migration file: `supabase/migrations/<next_timestamp>_02-worker-health-stats.sql`

```sql
-- Singleton row for rolling worker health stats (same pattern as site_metrics)
CREATE TABLE "public"."worker_health_stats" (
  "id"                        boolean NOT NULL DEFAULT true,
  "total_requests"            bigint  NOT NULL DEFAULT 0,
  "total_proxy_requests"      bigint  NOT NULL DEFAULT 0,
  "total_metrics_requests"    bigint  NOT NULL DEFAULT 0,
  "total_health_requests"     bigint  NOT NULL DEFAULT 0,
  "total_errors_4xx"          bigint  NOT NULL DEFAULT 0,
  "total_errors_5xx"          bigint  NOT NULL DEFAULT 0,
  "total_supabase_write_failures" bigint NOT NULL DEFAULT 0,
  "total_posthog_proxy_failures"  bigint NOT NULL DEFAULT 0,
  "total_decode_failures"     bigint  NOT NULL DEFAULT 0,
  "total_decode_successes"    bigint  NOT NULL DEFAULT 0,
  "sum_metrics_latency_ms"    bigint  NOT NULL DEFAULT 0,
  "count_metrics_latency"     bigint  NOT NULL DEFAULT 0,
  "sum_proxy_latency_ms"      bigint  NOT NULL DEFAULT 0,
  "count_proxy_latency"       bigint  NOT NULL DEFAULT 0,
  "sum_persist_latency_ms"    bigint  NOT NULL DEFAULT 0,
  "count_persist_latency"     bigint  NOT NULL DEFAULT 0,
  "updated_at"                timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "worker_health_stats_id_check" CHECK ((id = true)),
  CONSTRAINT "worker_health_stats_pkey" PRIMARY KEY (id)
);

-- Seed the singleton row
INSERT INTO "public"."worker_health_stats" (id) VALUES (true);

ALTER TABLE "public"."worker_health_stats" ENABLE ROW LEVEL SECURITY;

-- Allow public read for the HTMX dashboard
CREATE POLICY "public can read worker health stats"
  ON "public"."worker_health_stats"
  FOR SELECT TO "anon", "authenticated"
  USING (true);

-- Service role can update
GRANT SELECT, UPDATE ON TABLE "public"."worker_health_stats"
  TO "service_role", "postgres";
GRANT SELECT ON TABLE "public"."worker_health_stats"
  TO "anon", "authenticated";

-- RPC to atomically record a batch of worker health observations
CREATE OR REPLACE FUNCTION public.record_worker_health(
  p_request_type   text,     -- 'proxy', 'metrics', 'health', 'not_found'
  p_status_code    int,
  p_latency_ms     int DEFAULT NULL,
  p_persist_latency_ms int DEFAULT NULL,
  p_decode_success boolean DEFAULT NULL,
  p_supabase_write_failed boolean DEFAULT false,
  p_posthog_proxy_failed  boolean DEFAULT false
)
  RETURNS void
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO 'public'
AS $function$
BEGIN
  UPDATE public.worker_health_stats SET
    total_requests           = total_requests + 1,
    total_proxy_requests     = total_proxy_requests +
      CASE WHEN p_request_type = 'proxy' THEN 1 ELSE 0 END,
    total_metrics_requests   = total_metrics_requests +
      CASE WHEN p_request_type = 'metrics' THEN 1 ELSE 0 END,
    total_health_requests    = total_health_requests +
      CASE WHEN p_request_type = 'health' THEN 1 ELSE 0 END,
    total_errors_4xx         = total_errors_4xx +
      CASE WHEN p_status_code >= 400 AND p_status_code < 500 THEN 1 ELSE 0 END,
    total_errors_5xx         = total_errors_5xx +
      CASE WHEN p_status_code >= 500 THEN 1 ELSE 0 END,
    total_supabase_write_failures = total_supabase_write_failures +
      CASE WHEN p_supabase_write_failed THEN 1 ELSE 0 END,
    total_posthog_proxy_failures = total_posthog_proxy_failures +
      CASE WHEN p_posthog_proxy_failed THEN 1 ELSE 0 END,
    total_decode_failures    = total_decode_failures +
      CASE WHEN p_decode_success IS NOT NULL AND NOT p_decode_success THEN 1 ELSE 0 END,
    total_decode_successes   = total_decode_successes +
      CASE WHEN p_decode_success IS NOT NULL AND p_decode_success THEN 1 ELSE 0 END,
    sum_metrics_latency_ms   = sum_metrics_latency_ms +
      CASE WHEN p_request_type = 'metrics' AND p_latency_ms IS NOT NULL THEN p_latency_ms ELSE 0 END,
    count_metrics_latency    = count_metrics_latency +
      CASE WHEN p_request_type = 'metrics' AND p_latency_ms IS NOT NULL THEN 1 ELSE 0 END,
    sum_proxy_latency_ms     = sum_proxy_latency_ms +
      CASE WHEN p_request_type = 'proxy' AND p_latency_ms IS NOT NULL THEN p_latency_ms ELSE 0 END,
    count_proxy_latency      = count_proxy_latency +
      CASE WHEN p_request_type = 'proxy' AND p_latency_ms IS NOT NULL THEN 1 ELSE 0 END,
    sum_persist_latency_ms   = sum_persist_latency_ms +
      CASE WHEN p_persist_latency_ms IS NOT NULL THEN p_persist_latency_ms ELSE 0 END,
    count_persist_latency    = count_persist_latency +
      CASE WHEN p_persist_latency_ms IS NOT NULL THEN 1 ELSE 0 END,
    updated_at = now()
  WHERE id = true;
END;
$function$;

REVOKE ALL ON FUNCTION public.record_worker_health(text, int, int, int, boolean, boolean, boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.record_worker_health(text, int, int, int, boolean, boolean, boolean) TO "service_role", "postgres";
```

> [!IMPORTANT]
> Run this migration in Supabase SQL Editor **before** deploying the worker changes.

### 2B: Instrument `metrics-worker/worker.py`

Add timing and health reporting to the worker. The recording call itself should be fire-and-forget via `ctx.waitUntil` so it never adds latency to user-facing responses.

#### 2B-i: Add a timing helper at the top of `worker.py`

After the existing imports (line 16), add:

```python
import time
```

(`time` is available in Pyodide/Cloudflare Python Workers.)

#### 2B-ii: Add the health recording function

Add this new function near the other Supabase helper functions (after `supabase_request`, around line 416):

```python
async def record_worker_health(
    env,
    request_type,
    status_code,
    latency_ms=None,
    persist_latency_ms=None,
    decode_success=None,
    supabase_write_failed=False,
    posthog_proxy_failed=False,
):
    """Fire-and-forget health stats recording to Supabase."""
    if not getattr(env, "SUPABASE_URL", None) or not getattr(
        env, "SUPABASE_SERVICE_ROLE_KEY", None
    ):
        return
    try:
        body = {
            "p_request_type": request_type,
            "p_status_code": status_code,
        }
        if latency_ms is not None:
            body["p_latency_ms"] = int(latency_ms)
        if persist_latency_ms is not None:
            body["p_persist_latency_ms"] = int(persist_latency_ms)
        if decode_success is not None:
            body["p_decode_success"] = decode_success
        if supabase_write_failed:
            body["p_supabase_write_failed"] = True
        if posthog_proxy_failed:
            body["p_posthog_proxy_failed"] = True

        await fetch(
            f"{env.SUPABASE_URL}/rest/v1/rpc/record_worker_health",
            method="POST",
            headers={
                "Content-Type": "application/json",
                "apikey": env.SUPABASE_SERVICE_ROLE_KEY,
                "Authorization": f"Bearer {env.SUPABASE_SERVICE_ROLE_KEY}",
            },
            body=json.dumps(body),
        )
    except Exception as error:
        print(f"Worker health recording failed: {error}")
```

#### 2B-iii: Instrument the main `fetch` handler in `class Default`

Wrap the main handler (lines 54–89) with timing and health reporting. The updated method should:

1. Record `time.monotonic()` at the start.
2. After producing a response, compute elapsed milliseconds.
3. Call `ctx.waitUntil(create_proxy(record_worker_health(...)))` with the appropriate type, status, and latency.

Here is the updated `Default.fetch` method:

```python
async def fetch(self, request):
    start_time = time.monotonic()
    url = urlparse(request.url)
    origin = request.headers.get("Origin")
    cors_headers = get_cors_headers(origin)

    if request.method == "OPTIONS":
        requested_headers = request.headers.get("Access-Control-Request-Headers")
        preflight_headers = dict(cors_headers)
        if requested_headers:
            preflight_headers["Access-Control-Allow-Headers"] = requested_headers
        return Response(None, status=204, headers=preflight_headers)

    if url.path == "/health":
        response = Response.json(
            {
                "status": "ok",
                "timestamp": datetime.now(UTC).isoformat(),
            },
            headers=cors_headers,
        )
        elapsed_ms = (time.monotonic() - start_time) * 1000
        self.ctx.waitUntil(create_proxy(
            record_worker_health(self.env, "health", 200, latency_ms=elapsed_ms)
        ))
        return response

    if url.path == "/metrics" and request.method == "GET":
        response = await handle_metrics(self.env, cors_headers)
        elapsed_ms = (time.monotonic() - start_time) * 1000
        self.ctx.waitUntil(create_proxy(
            record_worker_health(self.env, "metrics", response.status, latency_ms=elapsed_ms)
        ))
        return response

    if is_posthog_path(url.path):
        response = await proxy_posthog(request, url, self.env, cors_headers, self.ctx)
        elapsed_ms = (time.monotonic() - start_time) * 1000
        posthog_failed = response.status >= 500
        self.ctx.waitUntil(create_proxy(
            record_worker_health(
                self.env, "proxy", response.status,
                latency_ms=elapsed_ms,
                posthog_proxy_failed=posthog_failed,
            )
        ))
        return response

    self.ctx.waitUntil(create_proxy(
        record_worker_health(self.env, "not_found", 404)
    ))
    return Response("Not Found", status=404, headers=cors_headers)
```

#### 2B-iv: Instrument `persist_posthog_payload` for decode & persist metrics

Update the `persist_posthog_payload` function (lines 289–318) to track:
- Whether payload decoding succeeded or failed
- How long the Supabase RPC call took
- Whether the Supabase write failed

The function already runs inside `ctx.waitUntil`, so it can't use `ctx.waitUntil` again for the health recording. Instead, just `await` the health recording call at the end (it's already in a background context so this is fine).

Updated `persist_posthog_payload`:

```python
async def persist_posthog_payload(payload_bytes, env, query_string=None):
    if not getattr(env, "SUPABASE_URL", None) or not getattr(
        env, "SUPABASE_SERVICE_ROLE_KEY", None
    ):
        print("Supabase environment variables are not configured")
        return

    payload = decode_posthog_body(payload_bytes, query_string)
    if payload is None:
        print("PostHog payload was not valid JSON or decodable data= form")
        await record_worker_health(env, "proxy", 200, decode_success=False)
        return

    events = normalize_posthog_payload(payload)
    if not events:
        await record_worker_health(env, "proxy", 200, decode_success=True)
        return

    persist_start = time.monotonic()
    supabase_failed = False
    try:
        response = await fetch(
            f"{env.SUPABASE_URL}/rest/v1/rpc/record_site_events",
            method="POST",
            headers={
                "Content-Type": "application/json",
                "apikey": env.SUPABASE_SERVICE_ROLE_KEY,
                "Authorization": f"Bearer {env.SUPABASE_SERVICE_ROLE_KEY}",
            },
            body=json.dumps({"p_events": events}),
        )

        if not response.ok:
            error_text = await response.text()
            print(f"Supabase event persistence failed: {response.status} {error_text}")
            supabase_failed = True
    except Exception as error:
        print(f"Supabase event persistence exception: {error}")
        supabase_failed = True

    persist_ms = (time.monotonic() - persist_start) * 1000
    await record_worker_health(
        env, "proxy", 200,
        decode_success=True,
        persist_latency_ms=persist_ms,
        supabase_write_failed=supabase_failed,
    )
```

> [!NOTE]
> This doubles the number of Supabase RPC calls per request (one for event data, one for health stats). For a personal site this is totally fine. If traffic ever becomes a concern, you could batch health stats into a local buffer and flush periodically, but that's unnecessary for now.

---

## Work Item 3: Add a "System Health" Tab to the Metrics Modal

### Goal
Extend the existing HTMX-powered metrics modal to show a second tab with system health data sourced from the `worker_health_stats` table built in Work Item 2.

### 3A: New `/health-metrics` endpoint in `worker.py`

Add a new handler in the `Default.fetch` method (alongside the existing `/health` and `/metrics` routes):

```python
if url.path == "/health-metrics" and request.method == "GET":
    response = await handle_health_metrics(self.env, cors_headers)
    elapsed_ms = (time.monotonic() - start_time) * 1000
    self.ctx.waitUntil(create_proxy(
        record_worker_health(self.env, "metrics", response.status, latency_ms=elapsed_ms)
    ))
    return response
```

Then add the handler function (near `handle_metrics`):

```python
async def handle_health_metrics(env, cors_headers):
    if not getattr(env, "SUPABASE_URL", None) or not getattr(
        env, "SUPABASE_SERVICE_ROLE_KEY", None
    ):
        return Response.json(
            {"error": "Supabase configuration missing"},
            status=500,
            headers=cors_headers,
        )

    try:
        response = await supabase_request(env, "/rest/v1/worker_health_stats?select=*")
        if not response.ok:
            raise Exception(f"Supabase health query failed: {response.status}")

        rows = await response.json()
        stats = rows[0] if rows else {}

        return Response(
            render_health_metrics_html(stats),
            headers={
                **cors_headers,
                "Content-Type": "text/html; charset=utf-8",
                "Cache-Control": "public, max-age=2, s-maxage=3, stale-while-revalidate=5",
            },
        )
    except Exception as error:
        print(f"Health metrics query failed: {error}")
        return Response(
            '<div class="error">⚠️ Unable to load health metrics</div>',
            status=503,
            headers={
                **cors_headers,
                "Content-Type": "text/html; charset=utf-8",
                "Cache-Control": "no-store",
            },
        )
```

And the HTML renderer:

```python
def render_health_metrics_html(stats):
    total_requests = number(stats.get("total_requests"))
    total_proxy = number(stats.get("total_proxy_requests"))
    total_metrics = number(stats.get("total_metrics_requests"))
    total_health = number(stats.get("total_health_requests"))
    errors_4xx = number(stats.get("total_errors_4xx"))
    errors_5xx = number(stats.get("total_errors_5xx"))
    sb_failures = number(stats.get("total_supabase_write_failures"))
    ph_failures = number(stats.get("total_posthog_proxy_failures"))
    decode_fail = number(stats.get("total_decode_failures"))
    decode_ok = number(stats.get("total_decode_successes"))

    # Compute averages
    sum_metrics_lat = number(stats.get("sum_metrics_latency_ms"))
    cnt_metrics_lat = number(stats.get("count_metrics_latency"))
    avg_metrics_ms = (sum_metrics_lat / cnt_metrics_lat) if cnt_metrics_lat > 0 else 0

    sum_proxy_lat = number(stats.get("sum_proxy_latency_ms"))
    cnt_proxy_lat = number(stats.get("count_proxy_latency"))
    avg_proxy_ms = (sum_proxy_lat / cnt_proxy_lat) if cnt_proxy_lat > 0 else 0

    sum_persist_lat = number(stats.get("sum_persist_latency_ms"))
    cnt_persist_lat = number(stats.get("count_persist_latency"))
    avg_persist_ms = (sum_persist_lat / cnt_persist_lat) if cnt_persist_lat > 0 else 0

    # Error rate
    error_rate = ((errors_4xx + errors_5xx) / total_requests * 100) if total_requests > 0 else 0

    # Decode success rate
    total_decode = decode_ok + decode_fail
    decode_rate = (decode_ok / total_decode * 100) if total_decode > 0 else 100

    html = f"""
    <h3>📡 Request Volume</h3>
    <div class="metric-row"><span class="metric-label">Total Requests</span><span class="metric-value">{fmt(total_requests)}</span></div>
    <div class="feature-grid">
      <div class="feature-box"><div class="feature-name">PostHog Proxy</div><div class="feature-count">{fmt(total_proxy)}</div></div>
      <div class="feature-box"><div class="feature-name">Dashboard</div><div class="feature-count">{fmt(total_metrics)}</div></div>
      <div class="feature-box"><div class="feature-name">Health Checks</div><div class="feature-count">{fmt(total_health)}</div></div>
      <div class="feature-box"><div class="feature-name">Error Rate</div><div class="feature-count">{error_rate:.1f}%</div></div>
    </div>
    <h3>⏱️ Avg Latency</h3>
    <div class="metric-row"><span class="metric-label">Dashboard Render</span><span class="metric-value">{avg_metrics_ms:.0f} ms</span></div>
    <div class="metric-row"><span class="metric-label">PostHog Proxy Round-Trip</span><span class="metric-value">{avg_proxy_ms:.0f} ms</span></div>
    <div class="metric-row"><span class="metric-label">Supabase Persist (bg)</span><span class="metric-value">{avg_persist_ms:.0f} ms</span></div>
    <h3>🛡️ Pipeline Health</h3>
    <div class="metric-row"><span class="metric-label">Payload Decode Success Rate</span><span class="metric-value">{decode_rate:.1f}%</span></div>
    <div class="metric-row"><span class="metric-label">4xx Errors</span><span class="metric-value">{fmt(errors_4xx)}</span></div>
    <div class="metric-row"><span class="metric-label">5xx Errors</span><span class="metric-value">{fmt(errors_5xx)}</span></div>
    <div class="metric-row"><span class="metric-label">Supabase Write Failures</span><span class="metric-value">{fmt(sb_failures)}</span></div>
    <div class="metric-row"><span class="metric-label">PostHog Proxy Failures</span><span class="metric-value">{fmt(ph_failures)}</span></div>
    """

    raw_updated_at = stats.get("updated_at") or datetime.now(UTC).isoformat()
    updated_at = stats.get("updated_at")
    if updated_at:
        try:
            updated = datetime.fromisoformat(updated_at.replace("Z", "+00:00")).strftime("%H:%M:%S")
        except ValueError:
            updated = updated_at
    else:
        updated = "—"

    html += f'<div style="margin-top:16px;padding-top:12px;border-top:1px solid rgba(0,212,255,.2);font-size:.8rem;color:#666;">Last updated: <span class="metrics-timestamp" data-utc="{escape_html(raw_updated_at)}">{escape_html(updated)}</span></div>'
    return html
```

### 3B: Add tabs to the modal in `index.html`

Replace the metrics modal content area (lines 81–89) with a tabbed layout:

```html
<!-- Metrics Modal -->
<div class="modal-overlay" id="metrics-modal">
    <div class="modal-content" id="modal-content">
        <button class="modal-close" title="Close">✕</button>
        <h2>🐶 BoopMaster 3000 Stats</h2>
        <div class="tab-bar">
            <button class="tab-btn active" data-tab="engagement">📊 Engagement</button>
            <button class="tab-btn" data-tab="health">🛠️ System Health</button>
        </div>
        <div id="tab-engagement" class="tab-panel active">
            <div id="metrics-container" hx-swap="innerHTML">
                <div class="loading" id="metrics-loading">📊 Loading metrics...</div>
            </div>
        </div>
        <div id="tab-health" class="tab-panel">
            <div id="health-container" hx-swap="innerHTML">
                <div class="loading">🛠️ Loading health data...</div>
            </div>
        </div>
    </div>
</div>
```

### 3C: Tab styles in `css/style.css`

Add these styles at the end of the file:

```css
/* Tab Bar */
.tab-bar {
    display: flex;
    gap: 0;
    margin-bottom: 16px;
    border-bottom: 2px solid rgba(0, 212, 255, 0.3);
}
.tab-btn {
    flex: 1;
    background: none;
    border: none;
    color: #888;
    font-family: monospace;
    font-size: 0.85rem;
    padding: 10px 8px;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -2px;
    transition: all 0.2s;
}
.tab-btn.active {
    color: #00d4ff;
    border-bottom-color: #00d4ff;
}
.tab-btn:hover {
    color: #fff;
}
.tab-panel {
    display: none;
}
.tab-panel.active {
    display: block;
}
```

### 3D: Tab switching + health polling in `js/main.js`

Add tab switching logic inside the main IIFE, near the existing modal code (after line 91). This needs to:

1. Wire up tab button clicks to swap `.active` between panels.
2. When the "System Health" tab is selected, fire an HTMX request to `/health-metrics` and start polling it.
3. When switching back or closing the modal, stop health polling.

```javascript
// Tab switching
const tabBtns = document.querySelectorAll('.tab-btn');
let healthPollInterval = null;

tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
        // Swap active tab button
        tabBtns.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        // Swap active panel
        document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
        const targetPanel = document.getElementById('tab-' + btn.dataset.tab);
        if (targetPanel) targetPanel.classList.add('active');

        // Start/stop appropriate polling
        if (btn.dataset.tab === 'health') {
            startHealthPolling();
        } else {
            stopHealthPolling();
        }
    });
});

function startHealthPolling() {
    stopHealthPolling();
    window.htmx.ajax('GET', 'https://metrics-api.jbirdkerr.net/health-metrics', '#health-container');
    healthPollInterval = setInterval(() => {
        if (document.hidden || !metricsModal.classList.contains('active')) return;
        const activeTab = document.querySelector('.tab-btn.active');
        if (!activeTab || activeTab.dataset.tab !== 'health') return;
        window.htmx.ajax('GET', 'https://metrics-api.jbirdkerr.net/health-metrics', '#health-container');
    }, 5000);
}

function stopHealthPolling() {
    if (healthPollInterval) {
        clearInterval(healthPollInterval);
        healthPollInterval = null;
    }
}
```

Also update `closeMetricsModal()` to stop health polling:

```javascript
function closeMetricsModal() {
    metricsModal.classList.remove('active');
    if (metricsPollInterval) {
        clearInterval(metricsPollInterval);
        metricsPollInterval = null;
    }
    stopHealthPolling();
}
```

And update the `htmx:afterSwap` / `htmx:afterSettle` listeners to also localize timestamps in the health container:

```javascript
document.addEventListener('htmx:afterSwap', (e) => {
    if (e.target && (
        e.target.id === 'metrics-container' ||
        e.target.id === 'health-container' ||
        e.target.querySelector('#metrics-container') ||
        e.target.querySelector('#health-container')
    )) {
        localizeMetricsTimestamp(e.target);
    }
});

document.addEventListener('htmx:afterSettle', (e) => {
    if (e.target && (
        e.target.id === 'metrics-container' ||
        e.target.id === 'health-container' ||
        e.target.querySelector('#metrics-container') ||
        e.target.querySelector('#health-container')
    )) {
        localizeMetricsTimestamp(e.target);
    }
});
```

---

## Work Item 4: Enable Cloudflare Observability Traces

### Goal
Turn on Cloudflare's built-in distributed tracing, which is already partially configured in `wrangler.toml` but currently disabled.

### File to modify
`wrangler.toml` — lines 36–39

### Change

```toml
[observability.traces]
enabled = true
head_sampling_rate = 1
persist = true
```

(Change `enabled = false` to `enabled = true`.)

### What this gives you
- Per-invocation traces visible in the **Cloudflare Dashboard → Workers & Pages → your worker → Logs & Traces**
- Request duration, CPU time, subrequest count, and status code breakdowns
- No code changes required — it's a Cloudflare platform feature

---

## Deployment Order

1. **Run the Supabase migration** (Work Item 2A) — creates the `worker_health_stats` table and RPC
2. **Deploy worker changes** (Work Items 2B, 3A, and 4) — `npx wrangler deploy --env production`
3. **Deploy frontend changes** (Work Items 1, 3B, 3C, 3D) — push to `main` for GitHub Pages auto-deploy

> [!IMPORTANT]
> The Supabase migration **must** be applied before the worker deploy, or the worker's `record_worker_health` calls will 404 against a nonexistent RPC.

---

## Verification Checklist

- [ ] PostHog → Web Analytics → Web Vitals shows LCP/INP/CLS data
- [ ] `curl https://metrics-api.jbirdkerr.net/health` still returns `{"status":"ok",...}`
- [ ] `curl https://metrics-api.jbirdkerr.net/health-metrics` returns HTML with request counts and latencies
- [ ] Metrics modal has two tabs (📊 Engagement / 🛠️ System Health) and both poll correctly
- [ ] Health tab shows non-zero request counts after a few interactions
- [ ] `worker_health_stats` row in Supabase shows incrementing counters
- [ ] Cloudflare Dashboard → Workers → Traces shows per-request traces
- [ ] Existing engagement metrics (`/metrics` tab) still work identically
- [ ] CI passes (`ruff check`, `ruff format --check`, unit tests)

## Files Changed Summary

| File | Action | Work Item |
|:-----|:-------|:----------|
| `js/main.js` | Edit `posthog.init` config; add tab logic & health polling | 1, 3D |
| `metrics-worker/worker.py` | Add `time` import, `record_worker_health()`, `handle_health_metrics()`, `render_health_metrics_html()`; instrument `Default.fetch` and `persist_posthog_payload` | 2B, 3A |
| `supabase/migrations/<timestamp>_02-worker-health-stats.sql` | New file — table, RPC, policies | 2A |
| `index.html` | Replace modal body with tabbed layout | 3B |
| `css/style.css` | Add `.tab-bar`, `.tab-btn`, `.tab-panel` styles | 3C |
| `wrangler.toml` | Flip `observability.traces.enabled` to `true` | 4 |
