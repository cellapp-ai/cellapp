# Proposal

## Why

The homepage describes the outcome but makes a first-time visitor infer both what Cellapp provides and the first action to take. The current Docs page has the right section names but mostly short summaries, so it does not yet serve as a readable guide from preparation through verification and recovery.

## What Changes

- Set the homepage slogan to the approved line, “your apps, your rules.”
- Put one explicit product sentence and one explicit usage sentence in the homepage's first screen, with a direct path to the deployment prompt and detailed guide.
- Use the approved four-part vision “Shape, Ship, Share and Sell.” Explain the first three through the current workflow and mark Sell as a future capability rather than an available sales service.
- Align the deployment page and metadata with those sentences while preserving the Agent-first flow: the Agent prepares and publishes a static frontend locally, the user approves browser authorization, and Cellapp returns an HTTPS app URL.
- Expand Docs into complete English and Simplified Chinese task-oriented guides with prerequisites, a first deployment walkthrough, expected results, verification, access and configuration guidance, a factual reference section, and troubleshooting with recovery steps. Give readers a visible language choice on both versions.
- Improve document hierarchy, scanning, cross-links, and narrow-screen reading without changing the static delivery model or adding unsupported product claims.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `product-site-surfaces`: States the product value and first-use action clearly at the public entry points.
- `documentation-surface`: Gives readers a complete, navigable path in their chosen English or Simplified Chinese version to deploy, verify, share, and recover from common failures.

The earlier `expand-landing-page-product-surfaces` change established these capabilities in the main website specs before this change's additional requirements were synced.

## Impact

- Website copy and structure in `src/App.tsx`, `src/DeployPage.tsx`, and `src/content.ts`; reading styles in `src/site.css` and `src/refinements.css`; static route export, localized route metadata and relevant website browser checks.
- Existing `/`, `/deploy/`, and English `/docs/` URLs remain stable; add `/docs/zh-CN/` for the complete Simplified Chinese guide. Other website pages remain English. The public Agent Markdown and CLI/API contracts remain authoritative sources for exact instructions.
- No new backend, CMS, dependency, payment behavior, or publishing flow. Cloudflare Pages release is a separate step after implementation and verification.
