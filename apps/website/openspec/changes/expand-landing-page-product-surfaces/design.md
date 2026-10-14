# Design

## Context

The current implementation is a single static `dist/index.html` with inline CSS and markup. It already establishes a strong dark ink/lime/blue palette, Space Grotesk plus DM Mono typography, terminal-led storytelling, and responsive behavior. There are no existing specs, package manifest, app router, backend, CMS, or payment integration to preserve.

## Goals / Non-Goals

**Goals:**

- Turn the current landing page into a small static product site with shared navigation and reusable page patterns.
- Preserve the existing visual identity while introducing enough hierarchy and surface variation for pricing, docs, and editorial content.
- Keep content local and easy to extend; support stable URLs and static-hosting refreshes.
- Use a React 18 + TypeScript + Vite + Tailwind + shadcn/ui structure when implementation begins, matching the named web-artifacts-builder workflow, while emitting a deployable static artifact.
- Make the key interactions real: responsive navigation, pricing cadence toggle, docs code-copy feedback, and route-aware active states.

**Non-Goals:**

- No user accounts, checkout, payment processing, authenticated docs, search backend, CMS, analytics redesign, or comment system.
- No rewrite of CellApp's deployment service or CLI behavior.
- No requirement to populate a large editorial archive; representative static content is sufficient for the first release.

## Decisions

### Use a shared shell with route-level content

Implement a single site shell (header, navigation, footer, CTA treatment, tokens) and route-level surfaces for Home, Pricing, Docs, Blog, and Blog Article. This avoids duplicating global behavior while retaining distinct page compositions. A single long-scroll page was rejected because pricing, docs, and articles need stable deep links and different reading/navigation patterns.

### Keep content in typed local data

Represent plans, documentation navigation/sections, and articles as typed local data consumed by the views. This keeps the first release static-friendly and makes content additions predictable. A CMS or remote API was rejected because it adds operational dependency without being required by the requested surface expansion.

### Prefer progressive enhancement for interactions

Use normal links for navigation and accessible buttons for the pricing cadence and code-copy behaviors. Route transitions may be client-side where available, but every supported URL must have a static fallback. This preserves usability if JavaScript is delayed or unavailable and fits the current static deployment target.

### Extend, do not replace, the existing visual system

Retain the dark ink background, lime primary accent, blue secondary accent, mono labels, angular/terminal motifs, restrained border radius, and supplied logo assets. New pages should vary layout through editorial grids, comparison rows, docs sidebar/code panels, and article typography rather than introducing generic card grids or heavy gradients.

### Build as a shareable frontend artifact

During implementation, initialize the frontend with the web-artifacts-builder scaffold, develop the multi-page experience there, and bundle the result to a single HTML artifact for review/deployment where appropriate. The source structure remains the maintainable implementation; bundling is a delivery/review step, not a replacement for it.

## Risks / Trade-offs

- [Risk] A single static bundle can become large as docs and blog content grow → keep content concise, split source modules by surface, and measure bundle size before release.
- [Risk] Deep links may fail on a static host without rewrite support → include a host-compatible fallback strategy and verify direct URL loads for every supported route.
- [Risk] Pricing copy may become stale or imply unsupported commercial behavior → keep prices and plan capabilities in one clearly named content source and label any placeholder values before launch.
- [Risk] Rich visual treatment can reduce reading clarity on docs/articles → prioritize typography, contrast, spacing, and keyboard behavior over decorative effects.

## Migration Plan

1. Capture the current landing page behavior and preserve its existing CTA destinations and supplied assets.
2. Introduce the shared shell and home surface in the new frontend structure.
3. Add Pricing, Docs, Blog listing, and Blog Article states with static fallback URLs.
4. Validate responsive layouts, keyboard access, copy feedback, direct navigation, and representative content.
5. Bundle/export the finished artifact and replace the current static entry only after visual and functional checks pass.
6. Roll back by restoring the prior `dist/index.html` if the new bundle fails validation or deployment checks.

## Open Questions

- Resolved for this delivery: on 2026-09-28 the product owner approved the preview and explicitly retained the pricing draft labels. Commercial names, limits and billing terms still require public-launch confirmation.
- Static fallback resolved: every supported route is exported as a complete `index.html` in its own directory. Direct loads and refreshes work on a plain static server without rewrite configuration.
