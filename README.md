# jbirdkerr.net

Personal interactive website hosted on **GitHub Pages** featuring an interactive canvas, WebAudio synthesis, particle engine, and a real-time event analytics and persistence pipeline powered by **Cloudflare Workers (Python)**, **Supabase (PostgreSQL)**, and **PostHog**.

---

## 🐶 Highlights

- **Interactive Canvas (`js/main.js`)**: Interactive dog photo with dynamic eye-tracking, snoot booping, treat showers, party mode, and WebAudio SFX.
- **Privacy-Friendly Edge Proxy (`metrics-worker/`)**: Python Cloudflare Worker that proxies PostHog SDK scripts and event ingestion to bypass ad blockers.
- **Dual-Write Pipeline**: Simultaneously dispatches analytics to PostHog Cloud and executes atomic batch upserts into Supabase PostgreSQL.
- **Live HTMX Dashboard**: Real-time stats modal powered by HTMX with 5-second polling and 3-second edge caching.

---

## 📚 Documentation

Detailed architectural overviews, sequence diagrams, and API specifications are available in the [`docs/`](docs/) directory:

- 🏗️ [**System Architecture**](docs/architecture.md): Multi-tier architecture diagram and component breakdown.
- 🔄 [**Metrics & Sequence Flows**](docs/metrics-pipeline.md): Sequence diagrams for event capture, background batch ingestion, and live HTMX dashboard rendering.
- 🛰️ [**Worker API Reference**](docs/api.md): Cloudflare Worker endpoint definitions, caching rules, CORS handling, and environment secrets.

---

## 📁 Repository Structure

```
.
├── index.html                   # Clean, semantic HTML skeleton
├── css/
│   └── style.css                # Layout, animations, modals & retro styles
├── js/
│   └── main.js                  # Audio synthesis, particle engine, tracking & UI logic
├── img/
│   ├── henrybeard.png           # Main interactive dog photo
│   └── henrybeard.jpg           # High-res photo source
├── docs/                        # Architecture & pipeline documentation
│   ├── README.md                # Documentation index
│   ├── architecture.md          # System architecture & component breakdown
│   ├── metrics-pipeline.md      # Sequence flows & ingestion pipeline
│   └── api.md                   # Cloudflare Worker API & secrets reference
├── metrics-worker/
│   ├── worker.py                # Cloudflare Python Worker (PostHog proxy + Supabase metrics)
│   ├── pyproject.toml           # Python worker runtime configuration
│   └── wrangler.toml            # Cloudflare Worker deployment settings
└── supabase/
    ├── migrations/              # PostgreSQL schema & RPC function migrations
    └── config.toml              # Supabase configuration
```

---

## 🚀 Quickstart & Deployment

### 1. Static Website (GitHub Pages)
Pushes to the `main` branch automatically build and publish static assets (`index.html`, `css/`, `js/`, `img/`, `docs/`) to GitHub Pages via GitHub Actions (`.github/workflows/static.yml`).

### 2. Cloudflare Worker Deployment
Deploy the Python worker to Cloudflare Workers with custom domain routing:

```bash
npx wrangler deploy --env production
```

Set required secrets in Cloudflare:
```bash
npx wrangler secret put SUPABASE_SERVICE_ROLE_KEY --env production
```

### 3. Supabase Database Setup
Execute the migration script in [`supabase/migrations/20260923042823_01-initial-schema.sql`](supabase/migrations/20260923042823_01-initial-schema.sql) in your Supabase SQL Editor to establish the schema and atomic RPC procedure (`record_site_events`).
