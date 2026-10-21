# AGENTS.md

Static Astro blog and portfolio for jbirdkerr.net. Experiments live in separate repos on subdomains.

## Commands
- `npm run dev`: local dev server (drafts are visible)
- `npm run build`: static build to `dist/` (drafts excluded)
- Deploys via `.github/workflows/deploy.yml` on push to `main`. Do not deploy unless asked.

## Layout
- `src/content/blog/*.md|mdx`: posts (schema in `src/content.config.ts`)
- `src/content/projects/*.md`: one entry per experiment, shown on `/` and `/projects/`
- `src/components/Giscus.astro`: comments, enabled by `PUBLIC_GISCUS_*` env vars
- `src/styles/global.css`: terminal theme tokens and `.post-body` Markdown styling

## Conventions
- Keep it static and JS-light. No framework islands unless asked.
- Privacy-conscious: no new third-party scripts or trackers without being asked. Fonts are self-hosted.
- Keep the dark terminal look (JetBrains Mono, blue/purple/green accents); respect `prefers-reduced-motion`.
- New posts need `title`, `description`, `pubDate`; set `draft: true` to hide in production.
