# Cellapp product website

This directory is the frontend project inside the main Cellapp repository (React 18, TypeScript, Vite and Tailwind). `src/` contains application code, `public/` contains public assets, and `scripts/` contains build and verification tools. This directory's `package.json` and `pnpm-lock.yaml` cover its own dependencies, independently of the parent npm workspace. `dist/` contains complete pre-rendered pages with inline application scripts and styles. Every article, `/deploy/`, English `/docs/`, and Simplified Chinese `/docs/zh-CN/` has its own static entry. JavaScript enhances pricing, navigation and copying; page content and language links are available without it. The public Agent runbook and user-app design guidance are shipped in English and Simplified Chinese at `/deploy/AGENTS.md`, `/deploy/AGENTS.zh-CN.md`, `/deploy/app-design.md`, and `/deploy/app-design.zh-CN.md`.

## Project standards

Read the [standards index](docs/standards/README.md) for the [project conventions](docs/standards/project.md), [coding conventions](docs/standards/code.md), and [design guidelines](docs/standards/design.md). These documents distinguish the current implementation, requirements for future changes, and known maintenance gaps.

## Develop

From `apps/website`, install dependencies with `corepack pnpm install --frozen-lockfile --registry=https://registry.npmjs.org`. Run `corepack pnpm run dev -- --host 127.0.0.1` to start development. The predev hook generates the embedded font styles automatically.

## Build and preview

Run `corepack pnpm run build`. It type-checks the source, builds with Vite and Parcel, inlines the application with HTML-inline, and exports complete HTML pages to `dist/`. The build copies both language editions of the public deployment runbook and user-app design guide alongside the static pages. Intermediate builds and the inline template are confined to the ignored `.build/` directory; the build does not install dependencies.

Run `corepack pnpm run preview` to serve the complete `dist/` folder at `http://127.0.0.1:4181`. Set `PORT` to choose a different port. The route directories support direct visits and refreshes without a server rewrite. Deploy the **whole dist folder**, including both language editions of the deployment runbook and design guide under `/deploy/`. `dist/bundle.html` is the portable single-file version; it uses query-based navigation when opened as a local file.

## Cloudflare deployment

The public site is deployed to the Cloudflare Pages project `cellapp-website`, with `main` as its production branch and `https://cellapp.ai` as its custom domain. This is a Direct Upload project; pushing to GitHub does not deploy the site automatically. After building and verifying a release, upload the complete static output with `npx wrangler pages deploy dist --project-name cellapp-website --branch main`. Wrangler must be logged into the Cloudflare account that owns the project. Check the deployment on both `https://cellapp-website.pages.dev` and `https://cellapp.ai`, including direct visits to `/docs/` and `/docs/zh-CN/`.

## Verify

With that preview running, run `corepack pnpm test`. The browser check covers all nine routes, refreshes, metadata and document language, four viewport widths, accessibility checks, keyboard actions, pricing handoff, exact clipboard text, the bilingual guides and section-preserving language links, the deployment prompt and the public Agent Markdown guides, no-JavaScript reading/navigation, and local-file bundle navigation. It saves screenshots and a validation report in `test-results/`. Set `TARGET_URL` and `PW_ARTIFACT_DIR` to override the preview URL or artifact folder. Set `CHROME_PATH` to an installed Chrome executable or install the Playwright browser with `corepack pnpm exec playwright install chromium`.

## Content

Edit plans, documentation sections and articles in `src/content.ts`. Adding an article to that typed list generates its card, detail page, metadata and static route on the next build. The square icon (`public/logos/cellapp-square.svg`) and Michroma `ce//app` horizontal wordmark (`public/logos/cellapp-landscape.svg`) are embedded in `src/logo.ts` for portable output. The navigation and footer use the horizontal wordmark; the square icon remains the favicon. The wordmark uses vector outlines and needs no font download at runtime. The navigation wordmark links home and highlights its slashes on the homepage. Its right-hand action reads “Deploy for free” until a same-origin session check confirms the visitor is signed in, then reads “Dashboard”; either state opens the existing dashboard entry and sign-in flow.

## Release review

Pricing names, dollar values, limits and billing terms are **drafts** and labeled as a plan preview. Selecting a plan opens the documentation; it does not create a subscription. The deployment guide uses the verified `@cellapp/cli` package and links to the public client repository. Paid app sales remain future work. See `RELEASE_REVIEW.md` for the prior release's content review.
