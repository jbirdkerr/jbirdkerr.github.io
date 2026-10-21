# AGENTS.md

Guidance for AI coding agents working in this repository. Humans: see `README.md`, `repo-spec.md`, and `docs/`.

## Project overview

**jbirdkerr.net** is a monorepo housing the projects, sites, and services deployed under the `jbirdkerr.net` domain:

| Unit | Directory | Runtime / Host | Purpose |
| --- | --- | --- | --- |
| `blog` | `blog/` | Astro / GitHub Pages | Main site (`jbirdkerr.net`), blog, projects index, Giscus comments |
| `boopmaster` | `apps/boopmaster/` | Vanilla JS / Cloudflare Pages | Interactive dog snoot booper (`boopmaster3000.jbirdkerr.net`) |
| `metrics-worker` | `apps/metrics-worker/` | Python / Cloudflare Worker | PostHog proxy + Supabase stats API (`metrics-api.jbirdkerr.net`) |
| `url-shortener` | `apps/url-shortener/` | FastAPI / Cloudflare Worker | Privacy-conscious shortener (`go.jbirdkerr.net`) |
| `db` | `db/supabase/` | Supabase (PostgreSQL) | Migrations and RPCs for site metrics and shortener |

Deployable units are declared in `deploy/units.toml`.

## Repository layout

```
.
├── BUILD.pants              # Root Pants configuration (or pants.toml)
├── pants.toml               # Pants build tool configuration
├── blog/                    # Astro static site (GitHub Pages)
│   ├── src/
│   ├── astro.config.mjs
│   └── package.json
├── apps/
│   ├── boopmaster/          # Dog canvas interactive site (Cloudflare Pages)
│   │   ├── index.html
│   │   ├── css/
│   │   ├── js/
│   │   └── img/
│   ├── metrics-worker/      # Python Cloudflare Worker
│   │   ├── worker.py
│   │   ├── pyproject.toml
│   │   └── wrangler.toml
│   └── url-shortener/       # FastAPI URL shortener Worker
│       └── SPEC.md
├── packages/
│   └── shared/              # Shared Python libraries, if any
├── db/
│   └── supabase/            # Supabase migrations and config
│       ├── migrations/
│       └── config.toml
├── deploy/
│   └── units.toml           # Deployable units manifest
└── docs/                    # Architecture and API documentation
```

## Setup and commands

- **Monorepo build tool:** Pants coordinates testing, linting, and packaging across units.
  - Test all: `pants test ::`
  - Lint all: `pants lint ::`
  - Detect affected: `pants --changed-since=origin/main list`
- **Blog (`blog/`):**
  - Uses Astro and Node.
  - Run dev server: `cd blog && npm run dev`
  - Build static output: `cd blog && npm run build`
- **Python tooling:**
  - Python dependencies are managed with `uv` (`uv.lock`). Run `uv sync` locally.
  - Lockfile sync: Pants consumes requirements exported with `uv export --no-hashes > requirements.txt`.
- **Pre-commit:**
  - `.pre-commit-config.yaml` is present. Run hooks before committing: `uv run pre-commit run --all-files`.
- **Database (`db/supabase/`):**
  - Add migrations as **new timestamped SQL files** in `db/supabase/migrations/`. Never edit an already-applied migration.
  - Applied in CI using `supabase db push`.
- **Deployments:**
  - Automatic on push or merge to `main`. Triggered in dependency order as declared in `deploy/units.toml`.
  - Do not manually deploy, push secrets, or run production migrations unless explicitly requested.

## Unit details and conventions

### 1. Blog (`blog/`)

- Astro static site with Tailwind CSS and MDX.
- Deployed to GitHub Pages via Actions artifact upload (`actions/upload-pages-artifact` and `actions/deploy-pages`).
- The `CNAME` file pointing to `jbirdkerr.net` must remain in `blog/public/CNAME`.
- URL paths must remain stable (`/blog/...`, `/projects/`, `/about/`) to prevent orphaning Giscus comment threads.

### 2. Boopmaster (`apps/boopmaster/`)

- Vanilla HTML/CSS/JS with canvas rendering, WebAudio synthesis, and particle effects.
- Served from `boopmaster3000.jbirdkerr.net` on Cloudflare Pages.
- Audio is synthesized with WebAudio; browsers require a user gesture before playing audio (no autoplay).

### 3. Metrics Worker (`apps/metrics-worker/`)

- Cloudflare Python Worker running at `metrics-api.jbirdkerr.net`.
- Proxies PostHog analytics and writes metrics asynchronously to Supabase via `ctx.waitUntil`.
- Allowed CORS origins: `https://jbirdkerr.net`, `https://www.jbirdkerr.net`, `https://boopmaster3000.jbirdkerr.net`, `https://henry.jbirdkerr.net`, and `https://jbirdkerr.github.io`.
- Non-secret config (`POSTHOG_PROJECT_ID`, `SUPABASE_URL`) lives in `apps/metrics-worker/wrangler.toml`.
- Secrets (`SUPABASE_SERVICE_ROLE_KEY`) are managed via Wrangler secrets or GitHub repository secrets.

### 4. URL Shortener (`apps/url-shortener/`)

- Planned FastAPI service running on Cloudflare Python Workers at `go.jbirdkerr.net`.
- Spec detailed in `apps/url-shortener/SPEC.md`. Uses a dedicated `shortener` database schema and RPCs.

### 5. Supabase Database (`db/supabase/`)

- Multi-row or atomic writes go through PostgreSQL RPCs (e.g. `record_site_events`).
- Rollbacks are forward-only: write a corrective migration rather than a down migration.

## Security and privacy

- Never commit secrets, service-role keys, or `.env` files.
- Privacy-conscious personal site: do not collect personal identifying information (no IPs, raw user agents, or tracking cookies).
- Keep `SUPABASE_SERVICE_ROLE_KEY` server-side only.

## Licensing

Licensed under **AGPL-3.0** (see `LICENSE`). Do not introduce dependencies with incompatible licenses.

## Working style for agents

- Make small, focused changes that match the existing style.
- Edit existing files rather than introducing redundant ones.
- Summarize changes clearly and flag anything that requires deployed testing or external credentials.
