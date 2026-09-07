# jbirdkerr.github.io & Metrics Worker API

This repository contains the personal website hosted on **GitHub Pages** along with a **Cloudflare Worker** service (`metrics-api.jbirdkerr.net`) that proxies PostHog analytics for event capture and live metric rendering.

---

## 🛠️ Components

### 1. GitHub Pages Site
- Static website files served from root (`index.html`).
- Automated deployment to GitHub Pages via GitHub Actions (`.github/workflows/static.yml`).

### 2. PostHog Metrics Worker (`metrics-worker/`)
- A Cloudflare Worker deployed to `https://metrics-api.jbirdkerr.net`.
- Built as an ES Module worker (`metrics-worker/worker.js`).
- Proxies PostHog analytics without exposing sensitive API keys to the browser, and avoids client-side ad-blockers.

---

## 🛰️ Worker API Routes

| Method | Route | Description |
| :--- | :--- | :--- |
| `GET` | `/metrics` | Fetches recent events from PostHog, aggregates totals, and returns HTML for HTMX integration. |
| `POST` | `/capture` | Ingests client analytics events and forwards them to PostHog (`https://us.i.posthog.com/capture/`). |
| `GET` | `/static/*` | Proxies PostHog JS SDK static assets (`array.js`) to bypass ad-blockers. |
| `GET` | `/health` | Health check endpoint returning `{"status": "ok"}`. |

---

## 🔐 Required Secrets & Environment Variables

### 1. Environment Variables (`wrangler.toml`)
* `POSTHOG_PROJECT_ID`: Your PostHog Project ID (e.g. `566829`).

### 2. Cloudflare Secrets (Uploaded via CLI)

| Secret Name | Key Type | Purpose | How to generate in PostHog |
| :--- | :--- | :--- | :--- |
| `POSTHOG_API_KEY` | Personal API Key (`phx_...`) | Reading project events (`GET /metrics`). | **Account Settings** -> **Personal API Keys** -> Create key with `read:events` scope. |
| `POSTHOG_PROJECT_KEY` | Project API Key (`phc_...`) | Ingesting analytics events (`POST /capture`). | **Project Settings** -> **Project API Key**. |

### 3. GitHub Repository Secrets (For CI/CD Auto-Deploy)

Add this secret under **Settings** -> **Secrets and variables** -> **Actions** in your GitHub repository:

| Secret Name | Description |
| :--- | :--- |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API Token with Workers deployment permissions (allows GitHub Actions to run `wrangler deploy`). |

## 🚀 Setup & Deployment

### 1. Add Secrets to Cloudflare Workers

Run the following commands in your terminal to store the secrets securely in Cloudflare:

```bash
# Upload Personal API Key (phx_...) for reading metrics
npx wrangler secret put POSTHOG_API_KEY --env production

# Upload Project API Key (phc_...) for sending/ingesting events
npx wrangler secret put POSTHOG_PROJECT_KEY --env production
```

### 2. Deploy Worker to Cloudflare

Deploy the worker and custom domain route (`metrics-api.jbirdkerr.net`):

```bash
npx wrangler deploy --env production
```

---

## 💻 Local Development

Create a `.dev.vars` file in the root directory for local testing with `wrangler dev`:

```env
POSTHOG_PROJECT_ID=566829
POSTHOG_API_KEY=phx_your_personal_key_here
POSTHOG_PROJECT_KEY=phc_your_project_key_here
```

Start the local worker dev server:

```bash
npx wrangler dev
```
