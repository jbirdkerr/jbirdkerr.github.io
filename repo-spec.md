# Deployment Spec: jbirdkerr.net Monorepo

Status: draft
Owner: jbirdkerr
Scope: the main site, subdomain projects, Cloudflare Workers, and the Supabase schema, all built and deployed from one repository.

## 1. Goals

- One repository is the source of truth for everything that ships under jbirdkerr.net.
- A change builds and tests only what it affects, locally and in CI.
- Changes that span units (a migration plus the Worker that reads it) deploy in a defined order.
- Every deployable unit gets a preview before it reaches production.
- Production deploys can be rolled back quickly.
- The CI logic is not tied to GitHub Actions. Actions is a trigger, not the place where the build graph lives.
- Costs stay on free tiers unless a unit explicitly needs more.

## 2. Non-goals

- Multi-region or high-availability infrastructure. This is a personal site.
- Replacing the Talos/Kubernetes lab. Cluster-hosted workloads are a later decision, handled by GitOps when they exist.
- Hermetic cross-language builds for languages not yet in use. Go and Rust are future work, and this spec should not block on them.

## 3. Current state

- Blog: Astro static site, moving into `blog/`. It publishes to GitHub Pages on `jbirdkerr.net` and keeps the Giscus comments component. Giscus is the reason the blog stays on GitHub Pages rather than moving to Cloudflare.
- Metrics worker: `metrics-worker/`, Python Worker, deployed with Wrangler.
- Supabase: `supabase/migrations/`, applied manually.
- Boopmaster (the snoot booper): currently served on henry.jbirdkerr.net, moving to boopmaster3000.jbirdkerr.net. Its hosting location needs to be confirmed before migration.
- `AGENTS.md` describes an older vanilla HTML layout with no build step. It must be updated to match the Astro source before the pipeline is built, since agents and tooling read it.

## 4. Repository layout

```
.
├── BUILD.pants              # root Pants config (or pants.toml only)
├── pants.toml
├── blog/                    # Astro site: blog, about, projects index. Publishes to GitHub Pages
├── apps/
│   ├── metrics-worker/      # Python Worker
│   ├── url-shortener/       # FastAPI service (planned, go.jbirdkerr.net)
│   └── boopmaster/          # snoot booper static site (boopmaster3000.jbirdkerr.net): index.html, css/, js/, img/
├── packages/
│   └── shared/              # shared Python code, if any
├── db/
│   └── supabase/            # migrations and config
├── deploy/
│   └── units.toml           # deployable unit manifest (see section 5)
└── docs/
```

The blog lives in `blog/` as its own Pants unit. The old dog site files at the repo root (`index.html`, `css/`, `js/`, `img/`) move into `boopmaster/`, which becomes its own unit published to `boopmaster3000.jbirdkerr.net`. After the move, the repo root holds only the blog's tooling and shared config.

The existing `.github/workflows/static.yml` publishes the repo root. It must be replaced by the blog's workflow in the same change that moves the dog files. Otherwise the root files keep publishing to `jbirdkerr.net` and conflict with the blog.

Keep the blog's URL paths stable (`/blog/...`, `/projects/`, `/about/`). Giscus maps threads by pathname, so changing paths would orphan existing comment threads.

## 5. Deployable units

Every deployable unit is declared in `deploy/units.toml`. This file is the contract between the build graph and the deploy stage.

Each unit has:

- `name`: stable identifier, used in logs and preview URLs
- `paths`: the Pants targets or path globs that define the unit
- `host`: `github-pages`, `cloudflare-pages`, or `cloudflare-worker`
- `build`: the Pants target that produces the artifact
- `deploy`: the command or Pants target that publishes it
- `depends_on`: other units that must deploy first
- `environments`: which of `preview` and `production` apply

Planned units:

| Unit | Host | Depends on |
| --- | --- | --- |
| `blog` | GitHub Pages | none |
| `boopmaster` | Cloudflare Pages | none |
| `metrics-worker` | Cloudflare Worker | `db` |
| `url-shortener` | Cloudflare Worker (FastAPI) | `db` |
| `db` | Supabase | none |

`db` has no deploy target in the build sense. It is a gate: its migrations must apply before any unit that reads the schema.

### Manifest schema (`deploy/units.toml`)

```toml
# deploy/units.toml
# Deployable unit manifest for jbirdkerr.net monorepo.

[units.blog]
name = "blog"
paths = ["blog/**"]
host = "github-pages"
build = "blog:dist"
deploy = "blog:deploy"
depends_on = []
environments = ["production"]

[units.boopmaster]
name = "boopmaster"
paths = ["apps/boopmaster/**"]
host = "cloudflare-pages"
build = "apps/boopmaster:dist"
deploy = "apps/boopmaster:deploy"
depends_on = []
environments = ["preview", "production"]

[units.db]
name = "db"
paths = ["db/supabase/migrations/**"]
host = "supabase"
build = "db/supabase:files"
deploy = "db/supabase:push"
depends_on = []
environments = ["preview", "production"]

[units.metrics-worker]
name = "metrics-worker"
paths = ["apps/metrics-worker/**"]
host = "cloudflare-worker"
build = "apps/metrics-worker:dist"
deploy = "apps/metrics-worker:deploy"
depends_on = ["db"]
environments = ["preview", "production"]

[units.url-shortener]
name = "url-shortener"
paths = ["apps/url-shortener/**"]
host = "cloudflare-worker"
build = "apps/url-shortener:dist"
deploy = "apps/url-shortener:deploy"
depends_on = ["db"]
environments = ["preview", "production"]
```

## 6. Build graph

Pants is the build tool. Each unit maps to Pants targets:

- Python units use `python_sources` and `python_tests`. For Worker packaging, to achieve minimal friction with Cloudflare's Pyodide runtime, Pants packages the worker directory (including `worker.py`, `pyproject.toml`, and `wrangler.toml`) as a bundle artifact (`archive` or file set). Wrangler handles the runtime dependency resolution during upload/deploy.
- Python lockfiles: `uv` is the primary source of truth for developer workflows (`uv.lock`). For Pants consumption, `uv export --no-hashes > requirements.txt` produces the locked requirements file, verified in CI to prevent lockfile drift.
- The Astro blog in `blog/` uses `shell_command` targets for `npm ci` and `npm run build`. Source files are declared as inputs and `dist/` as the output, so Pants caches correctly.
- Migrations use `files` targets, with the migration files as inputs.
- Dependency inference handles Python imports. Other dependencies are declared explicitly.

Affected-target detection in CI:

```
pants --changed-since=origin/main --changed-dependents=transitive list
```

The pipeline builds and tests only the units this command returns. A change to `db/supabase/migrations/` therefore pulls in `metrics-worker` and `url-shortener`, which is the intended behavior.

## 7. Pipeline stages

Deferred until pull requests are required. These stages are the target design, not the first version:

1. **Detect.** Compute the affected targets and the affected units from `units.toml`.
2. **Lint and test.** Run `pants lint` and `pants test` on affected targets only.
3. **Build.** Build affected unit artifacts. Cache hits should be visible in logs.
4. **Preview deploy.** Deploy each affected unit to a preview:
   - Cloudflare Pages: `wrangler pages deploy <dir> --project-name <name> --branch pr-<n>`
   - Worker: `wrangler versions upload` (creates a version without routing traffic to it)
   - Blog on GitHub Pages: no per-PR preview for now. GitHub Pages does not offer one, so revisit this when PRs become required.
5. **Migration check.** If `db/` changed, run migrations against a preview database, not production. Fail the pipeline if they do not apply cleanly.
6. **Smoke test.** Hit each preview's health or main route and assert on status and a known string.
7. **Report.** Post preview URLs to the PR.

First version: GitHub Actions runs on pushes to `main`. Each unit deploys only when its paths changed in the push, using the same affected-target logic as above.

1. Detect which units changed in the push, using `git diff` against the previous `main` commit or `pants --changed-since`.
2. **Migrate.** Apply `db/` migrations to production. This runs first, because units that depend on the schema must not deploy against a schema they don't expect.
3. **Deploy in dependency order.** Use `depends_on` to order units. Units with no dependency between them can deploy in parallel.
4. **Worker rollout.** Deploy the new version with gradual rollout (for example 10%, then 100%) if the rollout gate passes. Otherwise deploy directly.
5. **Post-deploy smoke test.** Hit `/health` on the Worker and the main routes on each site.
6. **Rollback on failure.** If the smoke test fails, roll back the Worker with `wrangler rollback` and revert the Pages deployment to the previous deployment. Migrations are not rolled back automatically. See section 9.

Production deploys run automatically on push or merge to `main`. Once pull requests are required (milestone 8), the same checks run on PRs and must pass before merge. No manual approval gate is required for pushes to `main`.

## 7a. Build and deploy split

Pants builds, tests, and packages. It does not publish. Publishing is a separate step that consumes a verified artifact. GitHub Actions is the runner and trigger for both.

Jobs:

1. **Build.** Checks out with `fetch-depth: 0` so `--changed-since` can see history. Restores the Pants cache. Runs `pants lint`, `pants test`, and `pants package` on the affected targets. Uploads each unit's artifact, plus a manifest with the unit name, commit SHA, and artifact hash.
2. **Deploy.** Runs once per affected unit, after the build job, and downloads that unit's artifact. Deploys are gated by a GitHub environment (`production`), so approvals can be added later without changing the pipeline. Deploy targets run through `pants run`, and credentials come from GitHub secrets exposed as environment variables.

Deploy targets:

- Each unit's `deploy` field in `units.toml` points to a Pants target that wraps the vendor CLI. Examples: Wrangler for Workers and Pages, the Supabase CLI for migrations, and the GitHub Pages actions for the blog.
- Deploy targets declare their dependencies in the graph. A Worker's deploy target depends on the `db` migration target, so migrations always run first.
- Check the exact shell-command target name against the installed Pants version before writing BUILD files.

Idempotency:

- A deploy is keyed to the artifact hash. If the hash matches the currently deployed version recorded in the deploy manifest, the deploy is skipped.
- Re-running a deploy with the same artifact produces the same result.

Versioning and caching:

- The Pants version is pinned in `pants.toml`.
- The Pants local cache is saved with `actions/cache`, keyed on the lockfiles and `pants.toml`. Consider a remote cache later if builds get slow.
- Secrets never go in `pants.toml` or BUILD files.

## 8. Environments and configuration

- **Preview (deferred):** separate Supabase project or schema. Separate Worker name or route. Preview URLs are private or unlisted.
- **Production:** the existing Supabase project and the `metrics-api.jbirdkerr.net` route.
- Non-secret config (`POSTHOG_PROJECT_ID`, `SUPABASE_URL`) lives in `wrangler.toml` per environment, as it does today.
- Secrets (`SUPABASE_SERVICE_ROLE_KEY`, Cloudflare API token, GitHub tokens for Pages) live in the CI secret store and in Wrangler secrets. They are never committed.
- Least privilege: the Cloudflare API token used by CI is scoped to Pages and Workers for this account, not account-wide.

## 9. Database migrations

- Migrations are new timestamped files in `db/supabase/migrations/`. Applied migrations are never edited (per AGENTS.md).
- Applied via Supabase CLI (`supabase db push`) in CI, authenticated via `SUPABASE_ACCESS_TOKEN` and `SUPABASE_DB_PASSWORD` (or direct connection).
- Preview migrations (deferred with preview environments) run against a preview database on every PR that touches `db/`.
- Production migrations run on merge, before dependent units deploy.
- Destructive migrations require an explicit label on the PR and a manual approval (once PRs are enforced).
- Rollback is forward-only: a corrective migration, not a down migration. Keep the RPC-based write path (`record_site_events`) stable across migrations so Workers on the previous version still function during the overlap.

## 10. Cloudflare Pages details

- One Pages project per subdomain project, all in the same account.
- Custom domain attached in the Pages project. Remove any existing GitHub Pages CNAME for that subdomain first.
- Direct upload from CI, not Cloudflare's Git build, so the build runs in Pants.
- Branch aliases give each branch a preview URL.
- boopmaster moves to `boopmaster3000.jbirdkerr.net`. Attach the new domain to its Pages project. No redirect from `henry.jbirdkerr.net` is needed, since that address was never published. Its project entry in `src/content/projects/` (`henry.md`) needs the new URL.
- Watch-path and build-cache behavior are controlled by Pants, not by Pages.

## 10a. GitHub Pages details

- Publish with the Actions source: `actions/upload-pages-artifact` uploads the blog's built output, then `actions/deploy-pages` deploys it. The "deploy from branch" option only accepts the repo root or `/docs`, so it cannot publish `blog/`'s build output directly.
- The artifact is built by Pants from `blog/`, so the Pages workflow runs `pants package` on the blog target and uploads the result.
- The `CNAME` file for `jbirdkerr.net` must be included in the uploaded artifact, or the custom domain is dropped on deploy.
- Only one Pages site per repo. This is the reason the subdomain projects move to Cloudflare Pages.

## 11. Observability

- Each deploy records: unit, version or deployment ID, commit SHA, timestamp, and outcome. Start with a log file in CI artifacts. Move to a `deploys` table in Supabase later, if useful.
- Worker logs go to Cloudflare Workers Logs.
- The existing `/metrics` and `/health` endpoints serve as the first post-deploy check.

## 11a. Rollback runbook

| Unit Type | Mechanism | Procedure |
| --- | --- | --- |
| **Worker** (`metrics-worker`, `url-shortener`) | `wrangler rollback` | Automated on smoke-test failure. Restores the prior deployment version ID immediately. |
| **Cloudflare Pages** (`boopmaster`) | Deployment alias restore | Rollback to the previous deployment ID via `npx wrangler pages deployment list` and alias pointer, or redeploying the prior commit's artifact. |
| **GitHub Pages** (`blog`) | Workflow rerun / commit revert | Re-run previous successful GitHub Pages deploy workflow or push git revert. |
| **Database** (`db`) | Forward-only migration | Automated rollback is explicitly avoided. Errors are mitigated by deploying a forward corrective migration script. |

## 11b. Secrets and credentials matrix

Secrets live in GitHub Actions Repository Secrets and Wrangler secrets:

| Secret | Target / Scope | Usage |
| --- | --- | --- |
| `CLOUDFLARE_API_TOKEN` | GitHub Secrets | Cloudflare Worker and Pages deployment via Wrangler (scoped to account Pages & Workers) |
| `CLOUDFLARE_ACCOUNT_ID` | GitHub Secrets | Cloudflare account identifier for Wrangler commands |
| `SUPABASE_ACCESS_TOKEN` | GitHub Secrets | Authenticates Supabase CLI for `supabase db push` |
| `SUPABASE_DB_PASSWORD` | GitHub Secrets | Database role password for applying migrations |
| `SUPABASE_SERVICE_ROLE_KEY` | Wrangler Secret | Injected into `metrics-worker` and `url-shortener` runtime for privileged RPC writes |
| `GITHUB_TOKEN` | Built-in GitHub Actions | Used by `actions/deploy-pages` and PR status reporting |

## 11c. Local developer workflow

Developers can develop and test units individually without invoking full Pants CI runs:

- **Blog (`blog/`):**
  - Dev server: `cd blog && npm run dev`
  - Build check: `cd blog && npm run build`
- **Workers (`apps/metrics-worker/`, `apps/url-shortener/`):**
  - Python environment: managed with `uv sync` in the worker directory.
  - Local runtime: `npx wrangler dev` (runs local Miniflare simulation).
  - Tests & linting: `uv run pytest` / `uv run ruff check .`
- **Pants commands (monorepo-wide):**
  - Run all tests: `pants test ::`
  - Run linters: `pants lint ::`
  - Check affected targets: `pants --changed-since=HEAD~1 list`
- **Sync dependencies:**
  - Export locked dependencies: `uv export --no-hashes > requirements.txt`

## 12. Risks

- **Pages and GitHub Pages split.** Two hosting models mean two sets of deploy logic. Keep the abstraction in `units.toml` so the pipeline doesn't branch on host type more than necessary.
- **Python Workers packaging.** Confirm how Cloudflare bundles Python dependencies before committing to a packaging target. Do not assume a standard pex will run.
- **FastAPI on Workers.** The project owner reports FastAPI runs on Python Workers, so `url-shortener` targets Cloudflare Workers. Verify this with a deployed preview before production, and pin the Python and FastAPI versions the Worker bundles, since the runtime's supported packages can change.
- **Lockfile drift.** `uv.lock` and Pants' lockfile must not diverge. Choose one source of truth for Python dependencies and document it.
- **Pants and Astro.** Wrapping npm in `shell_command` works, but caching depends on declaring inputs accurately. Missing inputs cause stale builds.
- **Giscus origins.** Giscus checks the requesting origin against an allowlist in `giscus.json` on the repo's default branch. For now the allowlist holds only the production origin, `https://jbirdkerr.net`. Verify this against current Giscus docs before relying on it.
- **Giscus and discussions.** Discussions must stay enabled on the repo and the Giscus app must stay installed. A change to either breaks comments on every post, not just new ones.
- **GitHub Pages artifact drift.** If the Pages workflow uploads from the wrong directory, the whole site is replaced with the wrong content. Smoke test the built `index.html` and a blog post in CI before the upload step.
- **Scope creep.** The demo should show the pipeline working end to end on a few units. Adding every project before the first one is stable is the most likely way this stalls.

## 13. Milestones

1. **Foundations.** Pants configured. `pants tailor` run. `units.toml` written. `AGENTS.md` updated to match the current repo.
2. **Worker and schema.** `metrics-worker` and `db` built and tested through Pants. Preview deploys working for the Worker and a preview database.
3. **Blog.** Astro build in `blog/` wrapped in Pants. Blog still publishes to GitHub Pages, now from the Pants artifact via the Actions source. Giscus verified on the production origin.
4. **First Pages project.** Migrate boopmaster, or start with a new small project. Preview deploys per PR working.
5. **Dependency-ordered production deploy.** Merge-to-main pipeline with migration gate, ordered deploys, and smoke tests triggered automatically on push.
6. **Rollout and rollback.** Gradual Worker rollout. Automatic rollback on smoke-test failure. Tested by deliberately breaking a deploy in preview.
7. **URL shortener.** Added as a unit. Verify FastAPI on Workers in a deployment before it serves production traffic.
8. **Pull requests and previews.** Require PRs for changes to `main`, with the pipeline's checks as required status checks. Bring back the preview stages (per-PR Pages previews, Worker versions, preview database) at the same time.

## 14. Decisions and open questions

### Resolved decisions

- **Repository layout:** Monorepo using `apps/` (`metrics-worker`, `boopmaster`, `url-shortener`), `blog/`, `db/supabase/`, and `deploy/units.toml`.
- **Worker packaging:** Low-friction directory/archive bundle containing `worker.py`, `pyproject.toml`, and `wrangler.toml`. Wrangler handles Pyodide runtime dependencies at deploy time; Pants handles lint, test, and dependency tracking.
- **Python dependency management:** `uv` is the developer tool of truth (`uv.lock`). Pants consumes requirements generated via `uv export --no-hashes > requirements.txt`, validated in CI.
- **Supabase migrations:** Executed via `supabase db push` in CI before dependent units deploy.
- **Production deployment trigger:** Direct merge/push to `main` triggers production deployment automatically. Manual approval gates are not used; PR branch protection will gate merges in Milestone 8.
- **Boopmaster hosting:** Moved from root files to `apps/boopmaster/`, published to `boopmaster3000.jbirdkerr.net` on Cloudflare Pages.

### Open questions

- **Preview database strategy:** When Milestone 8 introduces preview deploys, is a branch database or schema-per-PR better on Supabase free/pro tier?
- **URL shortener Phase 2 access:** When the web interface is introduced, will signups be invite-only or open with rate limits?
