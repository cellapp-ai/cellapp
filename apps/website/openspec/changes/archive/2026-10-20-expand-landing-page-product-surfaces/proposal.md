# Proposal

## Why

CellApp's current page communicates the core deployment promise well, but it behaves like a single campaign page: visitors cannot quickly compare plans, learn the product in a structured way, or discover ongoing product thinking. Expanding the site into a small product surface will make the landing experience more useful for evaluation, adoption, and sharing while preserving its sharp, terminal-inspired identity.

## What Changes

- Refresh the landing page information architecture with clearer entry points for product value, deployment workflow, pricing, documentation, and editorial content.
- Add a pricing surface with plan comparison, feature coverage, billing cadence toggle, and a clear primary conversion path.
- Add a documentation surface with getting-started guidance, CLI/API reference navigation, copyable code examples, and contextual links back to signup or deployment.
- Add a blog surface with featured article treatment, category/date metadata, article cards, and an article detail layout suitable for static content.
- Add shared site navigation, footer, active states, responsive behavior, metadata, and accessible focus/keyboard states across the new surfaces.
- Keep the initial implementation content-driven and static-friendly so it can ship without introducing a backend or CMS dependency.

## Capabilities

### New Capabilities

- `product-site-surfaces`: Provides the shared site shell and expanded landing-page information architecture.
- `pricing-surface`: Provides plan comparison and pricing conversion behavior.
- `documentation-surface`: Provides browsable, task-oriented product documentation.
- `blog-surface`: Provides editorial listing and article detail presentation.

### Modified Capabilities

None. The project has no existing OpenSpec capabilities to modify.

## Impact

- Primary implementation target is the existing static site entry at `dist/index.html`, with any supporting assets or content modules needed for navigation and page states.
- The visual system will extend the existing dark ink, lime accent, blue secondary accent, Space Grotesk/DM Mono typography, terminal motifs, and supplied CellApp logo assets.
- No backend, authentication, payment processor, CMS, or external content API is required for the first release; plan data, docs content, and blog content remain local/static.
- New routes or page states must work on static hosting, with a fallback strategy documented for direct navigation and refreshes.
