# jbirdkerr.net & Metrics Infrastructure

Personal interactive website hosted on **GitHub Pages** with a real-time event analytics and persistence pipeline powered by **Cloudflare Workers (Python)**, **Supabase (PostgreSQL)**, and **PostHog**.

---

## 🏗️ System Architecture

```mermaid
flowchart TD
    subgraph Client["🌐 Browser / Client (GitHub Pages)"]
        UI["Interactive Henry Canvas\n(Boop, Bark, Eyes, Party, Treats)"]
        PH_SDK["PostHog JS SDK\n(Batched Event Capture)"]
        HTMX["HTMX Dashboard\n(5s Polling + Visibility Aware)"]
    end

    subgraph CF["☁️ Cloudflare Edge (metrics-api.jbirdkerr.net)"]
        Worker["Python Worker\n(metrics-worker/worker.py)"]
        EdgeCache[("Cloudflare Edge Cache\n- 3s /metrics cache\n- 24h static asset cache")]
    end

    subgraph Storage["⚡ Supabase (PostgreSQL)"]
        RPC["RPC: record_site_events()"]
        T_Metrics[("public.site_metrics\n(Singleton totals & combos)")]
        T_Counts[("public.site_event_counts\n(Top event breakdown)")]
        T_Events[("public.site_events\n(Raw audit log)")]
    end

    subgraph PostHog["📊 PostHog Cloud"]
        PH_API["PostHog Ingestion API\n(us.i.posthog.com)"]
        PH_Assets["PostHog CDN\n(us-assets.i.posthog.com)"]
    end

    %% Client Interactions
    UI -->|Triggers events| PH_SDK
    UI -->|Opens Stats modal| HTMX

    %% Ingestion Flow
    PH_SDK -->|POST /i/v0/e/ or /batch| Worker
    Worker -->|1. Forward original payload| PH_API
    Worker -->|2. Async persist via ctx.waitUntil| RPC
    RPC --> T_Metrics
    RPC --> T_Counts
    RPC --> T_Events

    %% Metrics Query Flow
    HTMX -->|GET /metrics| EdgeCache
    EdgeCache -->|Cache Miss| Worker
    Worker -->|SELECT site_metrics + site_event_counts| Storage
    Worker -.->|Rendered HTML with data-utc| HTMX

    %% Asset Proxying
    PH_SDK -.->|GET /static/array.js| Worker
    Worker -.->|Proxy + CORS sanitize| PH_Assets
```

---

## 🔄 Metrics Flow & Sequence

### 1. Event Capture & Dual Ingestion Pipeline

When a user interacts with the canvas (boops the snoot, rolls the eyes, engages party mode, or showers treats), events are captured and written to both PostHog and Supabase in parallel:

```mermaid
sequenceDiagram
    autonumber
    actor User as 👤 Visitor
    participant Browser as 🌐 Browser (js/main.js)
    participant Worker as ☁️ Cloudflare Worker
    participant PostHog as 📊 PostHog Cloud
    participant Supabase as ⚡ Supabase (PostgreSQL)

    User->>Browser: Boops Snoot / Triggers Action
    Note over Browser: Plays WebAudio SFX & Spawns Particles (0ms)
    Browser->>Browser: PostHog SDK buffers & batches payload
    Browser->>Worker: POST /i/v0/e/ (JSON or gzip/base64 payload)
    
    par Forward to PostHog
        Worker->>PostHog: POST https://us.i.posthog.com/i/v0/e/
        PostHog-->>Worker: 200 OK
    and Persist to Supabase (Background)
        Worker->>Worker: Decode & normalize payload (JSON/Gzip/Base64)
        Worker->>Supabase: POST /rest/v1/rpc/record_site_events
        Note over Supabase: Atomic transaction updates:<br/>1. public.site_events (insert log)<br/>2. public.site_event_counts (upsert count)<br/>3. public.site_metrics (increment totals & combos)
        Supabase-->>Worker: 200 OK (Updated metrics JSON)
    end
    
    Worker-->>Browser: 200 OK
```

### 2. Live Dashboard Rendering Flow

When the metrics modal is opened, HTMX queries the Cloudflare Worker which retrieves the latest metrics from Supabase:

```mermaid
sequenceDiagram
    autonumber
    actor User as 👤 Visitor
    participant Modal as 📊 Metrics Modal (HTMX)
    participant Cache as ⚡ Edge Cache
    participant Worker as ☁️ Worker (/metrics)
    participant DB as ⚡ Supabase Tables

    User->>Modal: Clicks "📊 Metrics" or presses [M]
    Note over Modal: Immediately flushes pending eye distance
    Modal->>Cache: GET https://metrics-api.jbirdkerr.net/metrics
    
    alt Cache Hit (within 3 seconds)
        Cache-->>Modal: Return cached HTML response
    else Cache Miss
        Cache->>Worker: Invoke handle_metrics()
        par Query Summary & Events
            Worker->>DB: GET /rest/v1/site_metrics?select=*
            DB-->>Worker: Summary row JSON
        and Query Event Breakdown
            Worker->>DB: GET /rest/v1/site_event_counts?order=count.desc&limit=30
            DB-->>Worker: Event counts JSON
        end
        Worker->>Worker: Filter out internal ($*) events & render HTML
        Worker-->>Cache: HTML response (Cache-Control: s-maxage=3)
        Cache-->>Modal: Swaps HTML into #metrics-container
    end

    Note over Modal: JavaScript converts UTC timestamp to visitor's local timezone
```

---

## 📁 Repository Structure

```
.
├── index.html                   # Clean, semantic HTML skeleton
├── css/
│   └── style.css                # All layout, animations, modals & retro styles
├── js/
│   └── main.js                  # Audio synthesis, particle engine, tracking & UI logic
├── img/
│   ├── henrybeard.png           # Main interactive dog photo
│   └── henrybeard.jpg           # High-res photo source
├── metrics-worker/
│   ├── worker.py                # Cloudflare Python Worker (PostHog proxy + Supabase metrics)
│   ├── pyproject.toml           # Python worker runtime configuration
│   └── wrangler.toml            # Cloudflare Worker deployment settings
└── supabase/
    ├── migrations/              # PostgreSQL schema & RPC function migrations
    └── config.toml              # Supabase local & remote configuration
```

---

## 🛰️ Worker API Endpoints

| Method | Endpoint | Description | Caching Strategy |
| :--- | :--- | :--- | :--- |
| `GET` | `/metrics` | Fetches aggregated statistics from Supabase and returns styled HTML for HTMX. | `public, max-age=2, s-maxage=3` (Cloudflare edge cached) |
| `POST` | `/i/v0/e/`, `/batch` | Proxies analytics captures to PostHog and persists events to Supabase via RPC. | `no-store` |
| `GET` | `/static/*`, `/array/*` | Proxies PostHog JavaScript SDK static scripts to avoid ad-blocker issues. | `public, max-age=86400` (24h cache, CORS: `*`) |
| `GET` | `/health` | Service health check returning UTC ISO timestamp. | `no-store` |
| `OPTIONS` | `*` | Dynamic CORS preflight handler reflecting allowed origins and headers. | Handled via preflight headers |

---

## 🔐 Environment Variables & Secrets

### Cloudflare Worker Configuration (`wrangler.toml`)

* `POSTHOG_PROJECT_ID`: PostHog Project ID.
* `SUPABASE_URL`: Target Supabase project REST URL (`https://<project-ref>.supabase.co`).

### Cloudflare Worker Secrets (CLI Upload)

Store the private database access token securely in Cloudflare:

```bash
# Production Supabase Service Role Key (used for backend RPC and queries)
npx wrangler secret put SUPABASE_SERVICE_ROLE_KEY --env production
```

---

## 🚀 Deployment

### 1. Static Website (GitHub Pages)
Pushes to the `main` branch automatically build and publish static assets (`index.html`, `css/`, `js/`, `img/`) to GitHub Pages via GitHub Actions (`.github/workflows/static.yml`).

### 2. Cloudflare Worker Deployment
Deploy the Python worker to Cloudflare Workers with custom domain routing:

```bash
npx wrangler deploy --env production
```

### 3. Supabase Database Setup
Run the migration script in [`supabase/migrations/20260923042823_01-initial-schema.sql`](supabase/migrations/20260923042823_01-initial-schema.sql) in your Supabase SQL Editor to establish:
1. `site_metrics`: Singleton summary counters.
2. `site_event_counts`: Aggregated event breakdown table.
3. `site_events`: Full historical event log.
4. `record_site_events(p_events)`: Atomic batch ingestion stored procedure.
