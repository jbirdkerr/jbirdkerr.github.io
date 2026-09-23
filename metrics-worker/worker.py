"""
Python equivalent of the PostHog proxy + Supabase metrics Cloudflare Worker.

Requires (wrangler.jsonc):
  "compatibility_flags": ["python_workers"]
  "main": "src/entry.py"

And in pyproject.toml:
  dependencies = ["workers-runtime-sdk"]
"""

import base64
import gzip
import json
from datetime import datetime, timezone
from urllib.parse import parse_qs, unquote, urlparse

from pyodide.ffi import create_proxy
from workers import Response, WorkerEntrypoint, fetch

ALLOWED_ORIGINS = {
    "https://jbirdkerr.net",
    "https://www.jbirdkerr.net",
    "https://jbirdkerr.github.io",
}

POSTHOG_API_HOST = "https://us.i.posthog.com"
POSTHOG_ASSETS_HOST = "https://us-assets.i.posthog.com"


class Default(WorkerEntrypoint):
    async def fetch(self, request):
        url = urlparse(request.url)
        origin = request.headers.get("Origin")
        cors_headers = get_cors_headers(origin)

        if request.method == "OPTIONS":
            # Preflight: echo back whatever headers the browser says it's about
            # to send, rather than a hardcoded list. A hardcoded
            # Access-Control-Allow-Headers that doesn't cover every header the
            # actual request sends (e.g. PostHog SDK headers) causes the
            # browser to fail the preflight with "CORS Missing Allowed Header".
            requested_headers = request.headers.get("Access-Control-Request-Headers")
            preflight_headers = dict(cors_headers)
            if requested_headers:
                preflight_headers["Access-Control-Allow-Headers"] = requested_headers
            return Response(None, status=204, headers=preflight_headers)

        if url.path == "/health":
            return Response.json(
                {
                    "status": "ok",
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                },
                headers=cors_headers,
            )

        if url.path == "/metrics" and request.method == "GET":
            return await handle_metrics(self.env, cors_headers)

        # PostHog's browser SDK sends capture traffic through api_host.
        # We forward it to PostHog and independently persist the same events
        # to Supabase so the metrics path does not depend on PostHog query latency.
        if is_posthog_path(url.path):
            return await proxy_posthog(request, url, self.env, cors_headers, self.ctx)

        return Response("Not Found", status=404, headers=cors_headers)


def get_cors_headers(origin):
    headers = {
        "Access-Control-Allow-Methods": "GET,POST,OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type,Authorization,X-Requested-With",
        "Vary": "Origin",
    }

    if origin and origin in ALLOWED_ORIGINS:
        headers["Access-Control-Allow-Origin"] = origin

    return headers


def is_posthog_path(pathname):
    return (
        pathname.startswith("/static/")
        or pathname.startswith("/array/")
        or pathname.startswith("/e/")
        or pathname.startswith("/s/")
        or pathname.startswith("/flags/")
        or pathname == "/decide"
        or pathname == "/capture"
        or pathname == "/capture/"
        or pathname == "/batch"
        or pathname == "/batch/"
        or pathname == "/engage"
        or pathname.startswith("/i/")
    )


async def proxy_posthog(request, url, env, cors_headers, ctx):
    is_asset = url.path.startswith("/static/") or url.path.startswith("/array/")

    target_host = POSTHOG_ASSETS_HOST if is_asset else POSTHOG_API_HOST
    query = f"?{url.query}" if url.query else ""
    target_url = f"{target_host}{url.path}{query}"

    # request.headers is a JS Headers object exposed via FFI; it is iterable
    # as (name, value) pairs, and dict() will consume that directly.
    headers = dict(request.headers)
    headers.pop("host", None)
    headers.pop("origin", None)
    # request.arrayBuffer() transparently decompresses a gzip-encoded body
    # (common for PostHog's /s/ session-recording and /i/v0/e/ capture
    # traffic), so by the time we forward it the bytes are plain, not gzip.
    # Forwarding the original Content-Encoding header would tell PostHog to
    # gunzip already-plain data, producing an empty/garbage result and a
    # JSON parse failure on their end. Content-Length is dropped too since
    # the runtime recalculates it from the actual outgoing body anyway.
    headers.pop("content-encoding", None)
    headers.pop("content-length", None)

    body = None
    if request.method not in ("GET", "HEAD"):
        # .bytes() (not .arrayBuffer() — that's the raw JS method name and
        # doesn't exist on this Pythonic Request wrapper) reads the body as
        # raw bytes, binary-safe for compressed/non-UTF-8 payloads, and
        # returns native Python bytes directly.
        body = await request.bytes()

    # Make a best-effort copy of the event into Supabase. PostHog still gets
    # the original request even if the Supabase write fails.
    #
    # Path note: current posthog-js sends event captures to /i/v0/e/ by
    # default; /capture and /batch are older/alternate endpoints kept here
    # for compatibility. /s/ (session recordings) is deliberately excluded —
    # those aren't discrete "events" in the same shape. Confirm in your
    # Network tab which path your actual event captures use (a 200, not the
    # session-recording /s/ traffic) and add it here if it differs.
    persistable_paths = {
        "/capture",
        "/capture/",
        "/batch",
        "/batch/",
        "/i/v0/e/",
        "/i/v0/e",
        "/e/",
        "/e",
    }
    print(
        f"posthog proxy: method={request.method} path={url.path!r} "
        f"is_asset={is_asset} persistable={url.path in persistable_paths}"
    )
    if request.method == "POST" and not is_asset and url.path in persistable_paths:
        try:
            payload_bytes = body if body is not None else b""
            ctx.waitUntil(
                create_proxy(persist_posthog_payload(payload_bytes, env, url.query))
            )
        except Exception as error:
            print(f"Unable to queue Supabase persistence: {error}")

    response = await fetch(
        target_url,
        method=request.method,
        headers=headers,
        body=body,
        redirect="follow",
    )

    response_headers = dict(response.headers)
    response_headers.update(cors_headers)

    if is_asset:
        response_headers["Cache-Control"] = "public, max-age=86400"

    return Response(response.body, status=response.status, headers=response_headers)


def decode_posthog_body(payload_bytes, query_string=None):
    """Decode a PostHog capture payload into a Python dict/list.

    Handles raw gzip bytes, direct JSON, form-encoded / base64 payloads,
    and query string data without corrupting base64 characters.
    """
    if isinstance(payload_bytes, memoryview):
        payload_bytes = bytes(payload_bytes)

    # 1. Try direct gzip decompress
    if payload_bytes:
        try:
            return json.loads(gzip.decompress(payload_bytes))
        except Exception:
            pass

    # 2. Try direct JSON parse
    if payload_bytes:
        try:
            return json.loads(payload_bytes)
        except Exception:
            pass

    # 3. Try string representations from body or query string
    candidates = []
    if payload_bytes:
        try:
            candidates.append(payload_bytes.decode("utf-8").strip())
        except UnicodeDecodeError:
            try:
                candidates.append(payload_bytes.decode("latin-1").strip())
            except Exception:
                pass

    if query_string:
        candidates.append(query_string.strip())

    for text in candidates:
        tokens = [text]
        if "data=" in text:
            # Extract the raw data parameter value without converting '+' into spaces
            extracted = text.split("data=", 1)[1].split("&", 1)[0]
            tokens.append(extracted)
            tokens.append(parse_qs(text).get("data", [""])[0])

        for token in tokens:
            if not token:
                continue

            # Try token directly as JSON
            try:
                return json.loads(token)
            except Exception:
                pass

            # Try URL-unquoted token as JSON
            try:
                unquoted = unquote(token)
                return json.loads(unquoted)
            except Exception:
                pass

            # Try Base64 (both verbatim and with spaces turned back to '+')
            for b64_cand in [token, token.replace(" ", "+")]:
                try:
                    padded = b64_cand + "=" * (-len(b64_cand) % 4)
                    decoded = base64.b64decode(padded)
                    try:
                        return json.loads(gzip.decompress(decoded))
                    except Exception:
                        pass
                    try:
                        return json.loads(decoded)
                    except Exception:
                        pass
                except Exception:
                    pass

    return None


async def persist_posthog_payload(payload_bytes, env, query_string=None):
    if not getattr(env, "SUPABASE_URL", None) or not getattr(
        env, "SUPABASE_SERVICE_ROLE_KEY", None
    ):
        print("Supabase environment variables are not configured")
        return

    payload = decode_posthog_body(payload_bytes, query_string)
    if payload is None:
        print("PostHog payload was not valid JSON or decodable data= form")
        return

    events = normalize_posthog_payload(payload)
    if not events:
        return

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


def normalize_posthog_payload(payload):
    if isinstance(payload, dict) and isinstance(payload.get("batch"), list):
        raw_events = payload["batch"]
    elif isinstance(payload, dict) and payload.get("event"):
        raw_events = [payload]
    else:
        raw_events = []

    events = []
    for item in raw_events:
        event = item.get("event")
        if not isinstance(event, str):
            continue
        properties = item.get("properties")
        events.append(
            {
                "event": event,
                "properties": properties if isinstance(properties, dict) else {},
                "occurred_at": item.get("timestamp"),
                "distinct_id": (
                    item.get("distinct_id")
                    if isinstance(item.get("distinct_id"), str)
                    else None
                ),
            }
        )
    return events


async def handle_metrics(env, cors_headers):
    if not getattr(env, "SUPABASE_URL", None) or not getattr(
        env, "SUPABASE_SERVICE_ROLE_KEY", None
    ):
        return Response.json(
            {"error": "Supabase configuration missing"},
            status=500,
            headers=cors_headers,
        )

    try:
        summary_response = await supabase_request(env, "/rest/v1/site_metrics?select=*")
        events_response = await supabase_request(
            env,
            "/rest/v1/site_event_counts?select=event,count&order=count.desc&limit=30",
        )

        if not summary_response.ok or not events_response.ok:
            raise Exception(
                f"Supabase metrics query failed: "
                f"{summary_response.status}/{events_response.status}"
            )

        summary_rows = await summary_response.json()
        event_rows = await events_response.json()
        summary = summary_rows[0] if summary_rows else {}

        updated_at = summary.get("updated_at") or datetime.now(timezone.utc).isoformat()

        return Response(
            render_metrics_html(summary, event_rows),
            headers={
                **cors_headers,
                "Content-Type": "text/html; charset=utf-8",
                "Cache-Control": "public, max-age=2, s-maxage=3, stale-while-revalidate=5",
                "X-Metrics-Generated": updated_at,
            },
        )
    except Exception as error:
        print(f"Metrics query failed: {error}")
        return Response(
            '<div class="error">⚠️ Unable to load metrics</div>',
            status=503,
            headers={
                **cors_headers,
                "Content-Type": "text/html; charset=utf-8",
                "Cache-Control": "no-store",
            },
        )


async def supabase_request(env, path, method="GET"):
    return await fetch(
        f"{env.SUPABASE_URL}{path}",
        method=method,
        headers={
            "apikey": env.SUPABASE_SERVICE_ROLE_KEY,
            "Authorization": f"Bearer {env.SUPABASE_SERVICE_ROLE_KEY}",
        },
    )


def render_metrics_html(summary, event_rows):
    total_boops = number(summary.get("total_boops"))
    max_boop_combo = number(summary.get("max_boop_combo"))
    total_events = number(summary.get("total_events"))
    eye_tracking_toggles = number(summary.get("eye_tracking_toggles"))
    googly_eye_toggles = number(
        summary.get("googly_eyes_toggles", summary.get("googly_eye_toggles"))
    )
    party_mode_toggles = number(summary.get("party_mode_toggles"))
    treat_showers = number(summary.get("treat_showers"))
    total_eye_distance = number(summary.get("total_eye_distance"))

    distance_m = total_eye_distance / 3780

    raw_updated_at = summary.get("updated_at") or datetime.now(timezone.utc).isoformat()
    updated_at = summary.get("updated_at")
    if updated_at:
        try:
            updated = datetime.fromisoformat(
                updated_at.replace("Z", "+00:00")
            ).strftime("%H:%M:%S")
        except ValueError:
            updated = updated_at
    else:
        updated = "—"

    html = f"""
    <div class="metric-row"><span class="metric-label">Total Boops</span><span class="metric-value">{fmt(total_boops)}</span></div>
    <div class="metric-row"><span class="metric-label">Total Events Tracked</span><span class="metric-value">{fmt(total_events)}</span></div>
    <div class="metric-row"><span class="metric-label">Max Boop Combo</span><span class="metric-value">×{fmt(max_boop_combo)}</span></div>
    <div class="metric-row"><span class="metric-label">👀 Eyeball Distance Rolled</span><span class="metric-value">{distance_m:.2f} m <small>({total_eye_distance:,.0f} px)</small></span></div>
    <h3>🎮 Feature Toggles</h3><div class="feature-grid">
      <div class="feature-box"><div class="feature-name">Eye Tracking</div><div class="feature-count">{fmt(eye_tracking_toggles)}</div></div>
      <div class="feature-box"><div class="feature-name">Googly Eyes</div><div class="feature-count">{fmt(googly_eye_toggles)}</div></div>
      <div class="feature-box"><div class="feature-name">Party Mode</div><div class="feature-count">{fmt(party_mode_toggles)}</div></div>
      <div class="feature-box"><div class="feature-name">Treats</div><div class="feature-count">{fmt(treat_showers)}</div></div>
    </div><h3>📈 Event Breakdown</h3>
    """
    custom_events = [
        r for r in event_rows if (r.get("event") or "").strip() and not (r.get("event") or "").strip().startswith("$")
    ][:10]
    for row in custom_events:
        event = escape_html(row.get("event") or "")
        count = fmt(number(row.get("count")))
        html += f'<div class="metric-row"><span class="metric-label">{event}</span><span class="metric-value">{count}</span></div>'
    html += f'<div style="margin-top:16px;padding-top:12px;border-top:1px solid rgba(0,212,255,.2);font-size:.8rem;color:#666;">Last updated: <span class="metrics-timestamp" data-utc="{escape_html(raw_updated_at)}">{escape_html(updated)}</span></div>'
    return html


def number(value):
    try:
        return float(value) if value is not None else 0
    except (TypeError, ValueError):
        return 0


def fmt(value):
    return f"{value:,.0f}" if value == int(value) else f"{value:,}"


def escape_html(value):
    return (
        str(value)
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
        .replace("'", "&#039;")
    )
