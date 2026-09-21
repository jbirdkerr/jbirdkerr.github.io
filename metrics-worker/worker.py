import json
import time
from datetime import datetime, timezone
from typing import Any
from urllib.parse import urlparse

from pyodide.http import pyfetch
from workers import DurableObject, Response, WorkerEntrypoint

ALLOWED_ORIGINS = {
    "https://jbirdkerr.net",
    "https://www.jbirdkerr.net",
    "https://jbirdkerr.github.io",
    "http://localhost:8000",
    "http://127.0.0.1:8000",
    "http://localhost:8787",
}

METRICS_TTL = 5
CACHE_KEY = "metrics"
REFRESH_ALARM_DELAY_MS = 100


def get_origin(request) -> str | None:
    headers = getattr(request, "headers", {})
    if hasattr(headers, "get"):
        return headers.get("origin") or headers.get("Origin")
    return None


def cors_headers(origin: str | None) -> dict[str, str]:
    is_allowed = bool(origin and origin in ALLOWED_ORIGINS)
    effective_origin = origin if is_allowed else "https://jbirdkerr.net"
    return {
        "Access-Control-Allow-Origin": effective_origin,
        "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
        "Access-Control-Allow-Headers": (
            "Content-Type, Authorization, X-Requested-With, "
            "HX-Request, HX-Current-URL, HX-Target, HX-Trigger, HX-Trigger-Name, X-PostHog-Token"
        ),
        "Access-Control-Allow-Credentials": "true",
        "Vary": "Origin, HX-Request",
    }


async def fetch_metrics_from_posthog(key: str, project_id: str) -> dict[str, Any]:
    url = f"https://us.posthog.com/api/projects/{project_id}/query/"
    stats_query = {
        "query": {
            "kind": "HogQLQuery",
            "query": """
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
        """,
        }
    }
    breakdown_query = {
        "query": {
            "kind": "HogQLQuery",
            "query": "SELECT event, count() as cnt FROM events GROUP BY event ORDER BY cnt DESC LIMIT 10",
        }
    }
    headers = {"Authorization": f"Bearer {key}", "Content-Type": "application/json"}

    stats_res = await pyfetch(
        url, method="POST", headers=headers, body=json.dumps(stats_query)
    )
    breakdown_res = await pyfetch(
        url, method="POST", headers=headers, body=json.dumps(breakdown_query)
    )

    if stats_res.status != 200 or breakdown_res.status != 200:
        raise RuntimeError(
            f"PostHog API error: stats={stats_res.status}, breakdown={breakdown_res.status}"
        )

    stats_json = await stats_res.json()
    breakdown_json = await breakdown_res.json()

    row = stats_json.get("results", [[]])[0] if stats_json.get("results") else [0] * 8
    breakdown_map = {
        event: count for event, count in breakdown_json.get("results", [])
    }

    return {
        "totalBoops": int(row[0]) if row[0] else 0,
        "topCombo": round(float(row[1])) if row[1] else 0,
        "totalEvents": int(row[2]) if row[2] else 0,
        "eyeDistancePx": round(float(row[7])) if row[7] else 0,
        "featureToggles": {
            "eyeTracking": int(row[3]) if row[3] else 0,
            "googlyEyes": int(row[4]) if row[4] else 0,
            "partyMode": int(row[5]) if row[5] else 0,
            "treats": int(row[6]) if row[6] else 0,
        },
        "eventBreakdown": breakdown_map,
        "lastUpdated": datetime.now(timezone.utc).isoformat(),
    }


class MetricsCache(DurableObject):
    """Single global, durable 10-second application cache for PostHog metrics."""

    def __init__(self, ctx, env):
        super().__init__(ctx, env)
        self.storage = ctx.storage
        self.env = env
        self.cached_metrics = None
        self.last_refresh = 0.0

        async def initialize():
            record = await self.storage.get(CACHE_KEY)
            if record:
                self.cached_metrics = record.get("metrics")
                self.last_refresh = float(record.get("lastRefresh", 0))

        self.ctx.blockConcurrencyWhile(initialize)

    def fresh(self) -> bool:
        return (
            self.cached_metrics is not None
            and time.time() - self.last_refresh < METRICS_TTL
        )

    async def refresh(self):
        key = getattr(self.env, "POSTHOG_API_KEY", None)
        project_id = getattr(self.env, "POSTHOG_PROJECT_ID", None)
        if not key or not project_id:
            raise RuntimeError("Missing PostHog configuration")

        metrics = await fetch_metrics_from_posthog(key, project_id)
        self.cached_metrics = metrics
        self.last_refresh = time.time()
        await self.storage.put(
            CACHE_KEY,
            {
                "metrics": metrics,
                "lastRefresh": self.last_refresh,
            },
        )
        return metrics

    async def get_metrics(self):
        if not self.fresh():
            return await self.refresh()
        return self.cached_metrics

    async def alarm(self, alarm_info=None):
        """Refresh outside the HTTP request path; retain last-good data on failure."""
        try:
            if self.cached_metrics is None or not self.fresh():
                await self.refresh()
        except Exception as exc:
            print(f"Metrics refresh alarm failed: {exc}")
            await self.storage.setAlarm(int(time.time() * 1000) + METRICS_TTL * 1000)


class Default(WorkerEntrypoint):
    async def fetch(self, request):
        url = urlparse(request.url)
        origin = get_origin(request)
        headers = cors_headers(origin)

        if origin and origin not in ALLOWED_ORIGINS and request.method != "OPTIONS":
            return Response(
                json.dumps({"error": "Unauthorized origin"}),
                status=403,
                headers={**headers, "Content-Type": "application/json"},
            )

        if request.method == "OPTIONS":
            return Response(None, headers=headers)

        if url.path == "/metrics" and request.method == "GET":
            try:
                stub = self.env.METRICS_CACHE.getByName("global")
                metrics = await stub.get_metrics()
                return Response(
                    render_metrics_html(metrics),
                    status=200,
                    headers={
                        **headers,
                        "Content-Type": "text/html; charset=UTF-8",
                        "Cache-Control": "no-cache, no-store, must-revalidate",
                    },
                )
            except Exception as exc:
                print(f"Metrics Worker error: {exc}")
                return Response(
                    f'<div class="error">⚠️ Unable to load metrics<br><small>{exc}</small></div>',
                    status=500,
                    headers={**headers, "Content-Type": "text/html; charset=UTF-8"},
                )

        if url.path == "/health" and request.method == "GET":
            return Response(
                json.dumps({"status": "ok"}),
                headers={**headers, "Content-Type": "application/json"},
            )

        posthog_paths = [
            "/static/",
            "/array/",
            "/e/",
            "/s/",
            "/flags/",
            "/decide",
            "/capture",
            "/engage",
            "/i/",
        ]
        if any(url.path.startswith(path) for path in posthog_paths):
            return await self.posthog_proxy(request, url, headers)

        return Response(
            json.dumps({"error": "Not found"}),
            status=404,
            headers={**headers, "Content-Type": "application/json"},
        )

    async def posthog_proxy(self, request, url, cors):
        is_asset = url.path.startswith("/static/") or url.path.startswith("/array/")
        host = "us-assets.i.posthog.com" if is_asset else "us.i.posthog.com"
        target = f"https://{host}{url.path}" + (f"?{url.query}" if url.query else "")

        req_headers = {}
        headers_obj = getattr(request, "headers", {})
        if hasattr(headers_obj, "entries"):
            for k, v in headers_obj.entries():
                req_headers[k.lower()] = str(v)
        elif hasattr(headers_obj, "items"):
            for k, v in headers_obj.items():
                req_headers[k.lower()] = str(v)

        req_headers["host"] = host

        cf_ip = headers_obj.get("cf-connecting-ip") if hasattr(headers_obj, "get") else None
        if cf_ip:
            req_headers["x-forwarded-for"] = str(cf_ip)

        for drop_header in ["content-length", "cf-ray", "cf-connecting-ip", "cf-visitor", "connection"]:
            req_headers.pop(drop_header, None)

        body = None
        if request.method not in ("GET", "HEAD"):
            if hasattr(request, "bytes"):
                body = await request.bytes()
            elif hasattr(request, "arrayBuffer"):
                body = await request.arrayBuffer()
            else:
                body = await request.text()

        try:
            res = await pyfetch(
                target, method=request.method, headers=req_headers, body=body
            )

            res_headers = {}
            if hasattr(res.headers, "entries"):
                for k, v in res.headers.entries():
                    res_headers[k.lower()] = str(v)
            elif hasattr(res.headers, "items"):
                for k, v in res.headers.items():
                    res_headers[k.lower()] = str(v)

            for drop in [
                "content-encoding",
                "content-length",
                "transfer-encoding",
                "access-control-allow-origin",
                "access-control-allow-credentials",
                "access-control-allow-methods",
                "access-control-allow-headers",
                "access-control-expose-headers",
                "access-control-max-age",
            ]:
                res_headers.pop(drop, None)

            res_headers.update(cors)

            if is_asset:
                res_headers["Cache-Control"] = "public, max-age=86400"

            content = await res.bytes()
            return Response(content, status=res.status, headers=res_headers)
        except Exception as exc:
            print(f"PostHog proxy error: {exc}")
            return Response(
                json.dumps({"error": "Proxy failed"}),
                status=502,
                headers={**cors, "Content-Type": "application/json"},
            )


def render_metrics_html(metrics: dict[str, Any]) -> str:
    distance = metrics.get("eyeDistancePx", 0)
    toggles = metrics.get("featureToggles", {})
    html = f"""
    <div class="metric-row"><span class="metric-label">Total Boops</span><span class="metric-value">{metrics.get('totalBoops', 0)}</span></div>
    <div class="metric-row"><span class="metric-label">Total Events Tracked</span><span class="metric-value">{metrics.get('totalEvents', 0)}</span></div>
    <div class="metric-row"><span class="metric-label">Max Boop Combo</span><span class="metric-value">×{metrics.get('topCombo', 0)}</span></div>
    <div class="metric-row"><span class="metric-label">👀 Eyeball Distance Rolled</span><span class="metric-value">{distance / 3780:.2f} m <small>({distance:,} px)</small></span></div>
    <h3>🎮 Feature Toggles</h3><div class="feature-grid">
      <div class="feature-box"><div class="feature-name">Eye Tracking</div><div class="feature-count">{toggles.get('eyeTracking', 0)}</div></div>
      <div class="feature-box"><div class="feature-name">Googly Eyes</div><div class="feature-count">{toggles.get('googlyEyes', 0)}</div></div>
      <div class="feature-box"><div class="feature-name">Party Mode</div><div class="feature-count">{toggles.get('partyMode', 0)}</div></div>
      <div class="feature-box"><div class="feature-name">Treats</div><div class="feature-count">{toggles.get('treats', 0)}</div></div>
    </div><h3>📈 Event Breakdown</h3>
    """
    for event, count in metrics.get("eventBreakdown", {}).items():
        html += f'<div class="metric-row"><span class="metric-label">{event}</span><span class="metric-value">{count}</span></div>'
    updated = metrics.get("lastUpdated", "")
    try:
        updated = datetime.fromisoformat(updated).strftime("%H:%M:%S")
    except ValueError:
        updated = "—"
    return (
        html
        + f'<div style="margin-top:16px;padding-top:12px;border-top:1px solid rgba(0,212,255,.2);font-size:.8rem;color:#666;">Last updated: {updated}</div>'
    )