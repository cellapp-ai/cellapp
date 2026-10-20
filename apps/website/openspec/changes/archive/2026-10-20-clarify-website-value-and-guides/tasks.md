# Tasks

## 1. Clear public entry

- [x] 1.1 Set the homepage hero slogan to the exact approved line “your apps, your rules.”; verify its rendered wording and placement, then verify the separate product and usage sentences against the current public deployment guide, CLI behavior, and access rules and confirm each answers its question without relying on the terminal preview.
- [x] 1.2 Align `/deploy/` introductory copy, homepage CTA labels, and page metadata with those lines; verify navigation reaches the existing copyable prompt and does not imply server-side builds or available paid sales.
- [x] 1.3 Add the exact line “Shape, Ship, Share and Sell.” with four concise explanations; verify Shape, Ship and Share match current behavior, Sell is visibly marked “Coming later”, and no Sell purchase action is offered.
- [x] 1.4 Update relevant website browser assertions for the exact slogan and 4S line, visible first-screen explanation, Sell status, CTA destinations, and prompt availability; verify the checks pass on the built static pages.

## 2. Detailed documentation

- [x] 2.1 Expand typed Docs content into complete, corresponding English and Simplified Chinese guides with prerequisites, first deployment, browser approval, published-result verification, access/configuration, accurate reference links, and symptom-led recovery; verify commands and claims against current public client or project sources and compare the two versions for equal action, result, and recovery coverage.
- [x] 2.2 Render both guides with semantic steps, headings, lists, notes, links, and optional copyable examples; retain existing English section IDs and matching Chinese IDs, add a visible English/中文 switch on both pages, and verify section links, current-section indication, switch context, and copy success/failure behavior.
- [x] 2.3 Add `/docs/zh-CN/` to route resolution and static export while retaining English `/docs/`; localize each route's title, description, and document language and verify direct load, refresh, no-JavaScript guide and language switch, and `bundle.html?page=...` navigation for both languages.
- [x] 2.4 Refine Docs reading width, hierarchy, spacing, outline, CJK font fallback, and code overflow within existing styles; verify both languages at 320, 390, 768, and 1440 px, keyboard focus, and no page-level horizontal scrolling.
- [x] 2.5 Extend website browser coverage for both complete guides, their language switch, direct routes, no-JavaScript reading, single-file navigation, and a representative troubleshooting path in each language; verify the built-site tests pass and inspect both guides manually on desktop and mobile.

## 3. Integration and release readiness

- [x] 3.1 Run the website's lint, full build, and preview browser suite; verify generated `/`, `/deploy/`, `/docs/`, `/docs/zh-CN/`, route-specific metadata and language declarations, public Agent Markdown, and single-file output are correct, recording any failure or skipped check.
- [x] 3.2 Review website standards and public release notes for claims or links affected by the new copy; update only stale guidance and verify links and examples against the delivered static pages.

## Workflow follow-up

- `expand-landing-page-product-surfaces` was synced and archived first; this change's two capability deltas were then merged into the main specs without losing earlier requirements.
- Archive this change after implementation and required verification. Follow the repository's commit and push rules; a Cloudflare Pages upload and production acceptance are separate delivery steps.
