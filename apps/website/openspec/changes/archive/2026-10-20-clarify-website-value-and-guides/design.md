# Design

## Context

See [proposal.md](./proposal.md) for the motivation. The site renders `/`, `/deploy/`, and `/docs/` as complete static HTML, and also exports a single-file review build. `App.tsx` owns the page shell, `DeployPage.tsx` owns the copyable Agent prompt, and `content.ts` owns documentation sections and route metadata. The original Docs sections were Getting Started, Deploy, API Reference, Configuration, and Troubleshooting. The earlier `expand-landing-page-product-surfaces` change established the initial `product-site-surfaces` and `documentation-surface` specifications and was archived before this change.

## Goals / Non-Goals

**Goals:**

- Use the approved slogan “your apps, your rules.” as the homepage's primary brand statement.
- Express the four-part vision with the approved line “Shape, Ship, Share and Sell.” while distinguishing the planned Sell stage from today's product.
- Give the first screen two plainly worded, independent sentences, one for the offer and one for the first-use workflow.
- Make the English `/docs/` and Simplified Chinese `/docs/zh-CN/` routes equally useful as detailed human guides, with a visible language switch, stable section anchors, static rendering, and copy behavior.
- Keep each operational claim tied to the public Agent instructions, published CLI help, or client contracts before it is written into the site.

**Non-Goals:**

- Add a second publishing mechanism, a hosted build service, account signup, payment, CMS, or a new visual system.
- Change the Agent prompt protocol, public Markdown files, CLI/API contracts, or the application's access model.
- Localize the homepage, deployment page, or the rest of the website UI; this change localizes the detailed Docs guides and their own navigation and metadata.

## Decisions

### 1. Place the slogan and both explanatory sentences in the homepage hero

Replace the current hero heading with the exact approved slogan:

> your apps, your rules.

Show the two factual answers as separate sentences immediately below it. Working copy for implementation review:

> Cellapp hosts your static frontend as a shareable HTTPS app with access controls.

> Tell the coding Agent working on your project to follow the Cellapp deployment guide, approve access in your browser, then share the app URL it returns.

The first sentence names the input, service, and output. The second names the actor, approval, and result. Both can be understood without interpreting the terminal illustration. Link the primary action to `/deploy/` and the secondary action to `/docs/`. Keep the slogan as an expression of user choice, not as a claim that Cellapp provides the Agent or unrestricted hosting capabilities. Check the supporting wording against the public guide and live behavior before implementation. An alternative was to put only the slogan in the hero and move details lower; that keeps the current inference burden. Another was a large command block in the hero; the existing deploy page already supplies that interaction and fits it better.

### 2. Show 4S as a product journey below the first-screen explanation

Use the exact public line “Shape, Ship, Share and Sell.” as a compact section heading or lead-in after the hero, without displacing the factual product and usage sentences. Give each word one plain-language explanation: Shape means work with the coding Agent in the user's project; Ship means publish the static result; Share means give intended viewers access to the app link; Sell names the future commercial direction. Visibly mark Sell as “Coming later” and give it no purchase, listing, or checkout action. Existing draft pricing remains a separate preview. This makes the four parts a vision rather than a claim that all four services are live. An alternative was to place the 4S line alone in the hero; that would crowd out the two answers a new visitor needs first.

### 3. Expand the document model into two complete language routes

Keep `/docs/` as the English route and add `/docs/zh-CN/` as the Simplified Chinese route. Preserve the existing English section IDs and use corresponding IDs in Chinese so inbound English section links continue to work and a language switch can retain the current section. Place visible English/中文 links on both guides; use ordinary anchors so switching works before hydration and without JavaScript. Do not auto-redirect by browser language. Give each version full task guidance rather than a short translation summary.

Extend the typed document content in `content.ts` with parallel language content and shared section IDs for prerequisites, numbered actions, expected outcomes, notes, and recovery checklists. Keep rendering in page components and use semantic headings, lists, callouts, and restrained code blocks; preserve the outline and current-section indication in each language. Split a page component only if the added guide makes `App.tsx` difficult to maintain. Explicit routes are needed for stable links, complete static HTML, route-specific metadata, and a choice the reader controls; they do not require a router or CMS migration.

Use the same reading order and depth in both languages: what Cellapp hosts and prerequisites; first deployment via the copyable prompt; browser approval; published result and verification; access and public frontend configuration; reference links; symptom-led troubleshooting. Keep the matching English or Chinese public Agent instructions available near each walkthrough. For readers who want exact CLI syntax, link to the verified public client and show only commands confirmed against it.

### 4. Use progressive detail with factual checks

Lead with short action-and-result paragraphs in each language. Put deeper explanations, boundary conditions, and failure recovery under clear subheadings so a first-time reader can scan or read end to end. Distinguish the Agent's local build/upload from Cellapp's static hosting; distinguish the owner's deployment authorization from a viewer's app access. Avoid implying that a public app is anonymous or that draft plans can be purchased. Review commands, URLs, support links, and feature statements in both versions against current project sources, and compare both versions for missing or conflicting steps.

### 5. Preserve static and single-file behavior

Add both routes to the static export and route metadata. Export English `/docs/` with `html lang="en"` and English title/description, and `/docs/zh-CN/` with `html lang="zh-CN"` and Chinese title/description; set the language correctly during client navigation too. Use real section anchors and site links, with the site's existing `file:` link handling for cross-page navigation and language switching. Each guide's core text, outline, and switch must render in exported HTML before hydration; JavaScript enhances active-section and copy feedback only. Adjust reading width, typography, CJK font fallback, code overflow, and mobile outline in the existing CSS layers. Validate both direct routes and refreshes, no-JavaScript reading and switching, and `bundle.html?page=...` navigation alongside normal browser interactions.

## Risks / Trade-offs

- [More detail becomes stale as CLI or access behavior changes] → Link exact references and check every operational claim against current published or repository sources during implementation and release review.
- [Longer Docs content becomes hard to scan on mobile] → Use a compact outline, clear heading hierarchy, short sections, and a manual 320/390 px reading review.
- [English and Chinese guides drift or one becomes abbreviated] → Keep corresponding sections and check action, result, recovery, and link parity in both languages during implementation and release review.
- [Chinese direct route or language switch works only after hydration] → Export full HTML for both routes and use ordinary links; test direct refresh, disabled JavaScript, and single-file navigation.
- [Sequential changes declare the same capability paths] → The older change was synced and archived first; this delta added distinct requirements to those main specs without losing existing requirements.
- [Visual copy could overstate current capabilities] → Treat the quoted hero lines as proposed editorial wording and check access, hosting, and URL behavior before release.
- [Sell could be mistaken for an available service] → Keep the “Coming later” label adjacent to its 4S explanation and offer no sales action in this change.

## Migration Plan

1. Implement and verify the copy and both Docs guides; retain existing `/docs/` URLs and anchors, and add `/docs/zh-CN/` without redirecting existing readers.
2. Build the complete static `dist/`, inspect both direct Docs routes, their metadata and language declarations, and the single-file output; run the website's lint, build, browser, accessibility, and mobile checks applicable to this change.
3. The earlier website OpenSpec change was synced and archived before this one. Keep Cloudflare Pages upload as a separate release step requiring its own verification.
4. If the published wording causes a problem, restore the prior website static release; the content change has no server-side data migration.
