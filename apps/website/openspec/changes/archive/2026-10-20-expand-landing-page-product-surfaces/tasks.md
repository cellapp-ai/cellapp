# Tasks

## 1. Frontend foundation and shared shell

- [x] 1.1 Initialize the frontend workspace with the web-artifacts-builder scaffold and verify the React/TypeScript/Tailwind development build starts successfully.
- [x] 1.2 Define shared design tokens, typography, logo treatment, focus styles, and responsive layout primitives by mapping the existing `dist/index.html` visual language; verify the home surface retains its current core CTA and terminal story.
- [x] 1.3 Create typed local content models for plans, docs navigation/sections, and blog articles; verify representative content renders without remote requests.
- [x] 1.4 Implement the shared header, active navigation, footer, primary CTA, and route/deep-link fallback behavior; verify Home, Pricing, Docs, Blog, and an article URL can be opened directly and refreshed.

## 2. Landing page and pricing

- [x] 2.1 Recompose the landing page into the shared shell with clear entry points to Pricing, Docs, and Blog; verify desktop and mobile layouts have no clipped actions or horizontal overflow.
- [x] 2.2 Build the pricing surface with comparable plan cards/rows, capability categories, billing cadence control, and recommended-plan treatment; verify all plans expose consistent price, billing, feature, and action fields.
- [x] 2.3 Implement billing cadence state and accessible selected-state feedback; verify switching cadence updates all applicable prices, labels, and savings context together.
- [x] 2.4 Connect pricing actions to the configured getting-started/contact destination and preserve plan context where supported; verify keyboard activation and visible focus.

## 3. Documentation surface

- [x] 3.1 Build the docs layout with section navigation for Getting Started, Deploy, CLI/API Reference, Configuration, and Troubleshooting; verify the current section and getting-started entry are always discoverable.
- [x] 3.2 Author representative getting-started content covering prerequisites, installation, first deployment, and verification/sharing; verify each step includes the command or expected result needed for progression.
- [x] 3.3 Add labeled, copyable code blocks with accessible success feedback; verify exact command text is copied and feedback is announced or visually clear without navigation.

## 4. Blog surface

- [x] 4.1 Build the blog listing with a featured article and metadata-rich article cards; verify title, summary, category/tag, date, and reading time are visible for each entry.
- [x] 4.2 Build the article detail template with stable URLs, readable heading hierarchy, metadata, optional code/media treatment, and a relevant product/docs next action; verify at least one representative article renders end to end.
- [x] 4.3 Verify adding a valid article to the local content source makes it appear in the listing and resolve through the same detail template without shell changes.

## 5. Quality, artifact, and release checks

- [x] 5.1 Test responsive breakpoints, keyboard navigation, focus visibility, contrast, semantic labels, and reduced-motion behavior across all surfaces; verify no primary flow depends on hover alone.
- [x] 5.2 Test direct navigation and refresh for every supported route on the intended static-host fallback; verify each page returns the correct title/description and shared navigation state.
- [x] 5.3 Bundle the implementation to the shareable static artifact using the web-artifacts-builder bundling workflow and verify the bundled HTML opens without missing assets or runtime errors.
- [x] 5.4 Review final pricing placeholders, CTA destinations, supplied logo assets, and representative docs/blog content with product owners; verify any unresolved commercial values are explicitly labeled before release.

## Verification record

- 2026-09-28: React 18 scaffold, strict TypeScript check, Vite build and skill-provided Parcel/HTML-inline bundling completed.
- `scripts/test-site.cjs` passed on all seven static routes with direct loads and refreshes, metadata, active navigation, WCAG automated scans, and widths 320/390/768/1440.
- Verified keyboard pricing selection, accurate annual amounts/savings, plan context in Docs, exact clipboard contents, docs active section, mobile menu, reduced motion, no-JavaScript content/navigation and file:// single-bundle navigation. No browser runtime or resource errors.
- Every article added to `src/content.ts` rendered its own heading/body and received a generated static entry through the shared template.
- Owner approved preview delivery and retention of pricing draft labels. See `RELEASE_REVIEW.md`; this approval does not authorize public publication or finalize commercial terms.
- Follow-up requested by the owner: frontend source, configuration, and test dependencies now live directly at the repository root; build intermediates use `.build/` and published static pages remain in `dist/`. Verified root-level `npm run dev`, `npm run build`, `npm run preview` and `npm test`; all seven page checks and interactive flows pass with zero browser errors.
