# jbirdkerr.net

Astro blog and portfolio with a dark terminal theme, Markdown/MDX posts, RSS, sitemap, and giscus comments.

```sh
npm install
npm run dev      # http://localhost:4321
npm run build
```

## Writing a post
Add `src/content/blog/my-post.md`:

```md
---
title: My post
description: One or two sentences (max 220 chars).
pubDate: 2026-10-12
tags: [notes]
---
```

## Comments (giscus)
1. Create a public GitHub repo for the blog and enable Discussions (add a "Comments" category, type Announcement).
2. Install the giscus app on that repo: https://github.com/apps/giscus
3. Open https://giscus.app, enter the repo, pick the category, and copy `data-repo-id` and `data-category-id`.
4. Locally: copy `.env.example` to `.env` and fill it in.
   In GitHub: add the four `PUBLIC_GISCUS_*` values as Actions *variables* (not secrets).

## Deploying to GitHub Pages
Settings > Pages > Source: GitHub Actions, then set the custom domain.
Move the current Henry page to its own repo and subdomain first, so `jbirdkerr.net` is free for the blog.

## Adding an experiment
Create `src/content/projects/<name>.md` with `title`, `description`, `url`, and optional `repo`, `stack`, `order`.
