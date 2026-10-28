# jbirdkerr.net monorepo

This repository contains the Astro personal site at [jbirdkerr.net](https://jbirdkerr.net) and the experiments and services hosted on its subdomains. The site lives in `blog/`; apps and backend services live in `apps/`.

## Projects

| Project | Source | What it is | Hosting / status |
| --- | --- | --- | --- |
| Personal site and blog | [`blog/`](blog/) | Astro blog, about page, and project index | GitHub Pages at `jbirdkerr.net` |
| Boopmaster 3000 | [`apps/boopmaster/`](apps/boopmaster/) | Interactive Henry photo with eye effects, snoot boops, party mode, and a metrics panel | Cloudflare Pages at `boopmaster3000.jbirdkerr.net` |
| DigDog | [`apps/digdog/`](apps/digdog/) | Go and Ebitengine card-matching game compiled to WebAssembly | Cloudflare Pages at `digdog.jbirdkerr.net`; separate workflow |
| Metrics worker | [`apps/metrics-worker/`](apps/metrics-worker/) | Python Cloudflare Worker that proxies PostHog traffic and serves Supabase-backed metrics | `metrics-api.jbirdkerr.net` |
| Metrics database | [`db/supabase/`](db/supabase/) | Supabase configuration and PostgreSQL migrations for site metrics | Used by the metrics worker |
| URL shortener | [`apps/url-shortener/SPEC.md`](apps/url-shortener/SPEC.md) | Design specification; the service is not implemented or in the active deployment workflow | Planned |

More detail: [DigDog](apps/digdog/README.md), [metrics-worker docs](apps/metrics-worker/docs/README.md), and the [URL shortener spec](apps/url-shortener/SPEC.md).

## Repository map

```text
.
├── .github/workflows/       # CI and deployment workflows
├── apps/
│   ├── boopmaster/          # Static JavaScript site
│   ├── digdog/              # Go / WebAssembly game
│   ├── metrics-worker/      # Python Worker, tests, and docs
│   └── url-shortener/       # Current design spec
├── blog/                    # Astro site
├── db/supabase/             # Database config, schemas, and migrations
├── deploy/units.toml        # Main workflow's affected-unit manifest
├── scripts/                 # CI helper scripts
├── pants.toml               # Pants configuration
└── repo-spec.md             # Draft deployment design
```

## Local development

Use the toolchain required by the unit you are working on. The blog requires Node.js 22.12 or newer; DigDog's Go version is declared in `apps/digdog/go.mod`.

### Blog

```sh
cd blog
npm ci
npm run dev      # http://localhost:4321
npm run build
```

### Repository checks

From the repository root:

```sh
uv run pre-commit run --all-files
./pants test ::
```

For DigDog's Go formatting, WebAssembly vet/build, and unit tests:

```sh
cd apps/digdog
./check.sh
```

The metrics-worker tests are also run by the repository's pre-commit and CI workflows. For unit-specific setup, see the README or docs in that unit.

## CI and deployments

- [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml) validates pull requests and pushes targeting `main`. Its production deploy job runs on pushes to `main` and manual dispatch, detects affected units, and handles database migrations, Boopmaster, the metrics worker, and the blog.
- [`.github/workflows/digdog.yml`](.github/workflows/digdog.yml) separately checks DigDog changes and deploys the game from `main` to Cloudflare Pages.
- [`deploy/units.toml`](deploy/units.toml) and [`scripts/detect_changed_units.py`](scripts/detect_changed_units.py) describe how the main workflow maps changed paths to affected units. Check the workflows too: DigDog uses its own workflow.
- The URL shortener is still a specification, not a deployed service.

Production deployments and database migrations run through GitHub Actions. Do not run a manual production deploy or migration unless explicitly requested. Never commit `.env` files, tokens, service-role keys, or other credentials.

## License

This repository is licensed under [AGPL-3.0](LICENSE).
