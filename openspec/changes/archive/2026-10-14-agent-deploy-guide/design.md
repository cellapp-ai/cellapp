# Design

## Context

The existing `cellapp-website` project serves `cellapp.ai` from Cloudflare Pages. Its source has been copied into `apps/website` as an independent pnpm project; the original checkout and its uncommitted documents remain untouched. The Go control domain remains at `console.cellapp.ai`. The CLI already handles device authorization, local builds, upload retries and publication. Cellapp hosts static frontends only; application data is a separate optional workflow.

## Decisions

- Export a complete static `/deploy/` page in the website, reachable from `https://cellapp.ai/deploy`. Keep its copyable prompt short and point to `https://cellapp.ai/deploy/AGENTS.md`. Show static hosting limits, browser authorization and the published result in ordinary language.
- Copy the Markdown into the Cloudflare Pages output at `/deploy/AGENTS.md`. The website preview serves it as `text/markdown`; no Go route or origin proxy is needed.
- Preserve the website's own pnpm lockfile and Cloudflare Pages Direct Upload workflow. Do not add it to the control console's npm workspace or publish it as a side effect of a Git push.
- Agent guidance checks local Node.js >=22.12, detects an existing CLI, installs `@cellapp/cli` from npm when absent, verifies `cellapp --help`, and reports environmental installation failures. The Agent does not ask the user to type installation commands unless it cannot proceed.
- Preserve existing CLI and device authorization. The Agent shows the verification URL and code, waits for the user to approve in the browser, and reports success only after the CLI confirms publication. Reuse existing project association for updates.
- Make the public page and guide accurate for static sites and supported application tables; keep selling out of the current feature claims.

## Validation

Check public routes and Markdown response, no-JavaScript reading, copy behavior, mobile and desktop layout, Skill structure and package boundary. Run website lint/build/browser checks and relevant root/client checks.
