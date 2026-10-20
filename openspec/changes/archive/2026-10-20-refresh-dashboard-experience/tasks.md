# Tasks

## 1. Owner table structure contract

- [x] 1.1 Add an owner-session `GET /api/console/apps/{app}/data/structure` read that projects the latest applied manifest and readiness without rows or database internals; verify isolated Go tests for owned, signed-out, cross-owner, empty-version, pending, and unavailable cases.
- [x] 1.2 Add strict browser response parsing for the structure payload and loading/empty/error handling; verify malformed payloads fail visibly and valid tables render only the documented fields in frontend tests.
- [x] 1.3 Document the read response and error semantics in `docs/web-api.md`; verify examples and path names against the handler and client parser.

## 2. Shared dashboard shell and application list

- [x] 2.1 Refine semantic tokens, sidebar, content alignment, page heading, focus, status, and action styles across desktop and mobile; verify the production build at 320, 390, 768, 800/801, 1440 px and 200% zoom without page-wide overflow.
- [x] 2.2 Rebuild the real application list with explicit manage/open destinations, readable address and publication state, and truthful loading/empty/error feedback; verify long English/Chinese names, unpublished and suspended states, keyboard operation, and navigation in browser acceptance checks.
- [x] 2.3 Update `docs/standards/design.md` only for durable design convention changes; verify the documented tokens, layout rules, and brand naming match the delivered interface.

## 3. Application detail and tables

- [x] 3.1 Reorganize application detail into clear identity, release, tables, access, and settings sections while preserving address copy, slug editing, confirmations, uncertain-write feedback, and one-time key clearing; verify the corresponding existing browser behaviors and direct detail refresh.
- [x] 3.2 Render the read-only table list and selected table structure with columns, keys, indexes, policies, and applied version; verify no row request or schema mutation occurs and pending, empty, and failed states are distinct in browser tests.
- [x] 3.3 Extend isolated browser and server tests for table structure authorization, stale/readiness presentation, long table identifiers, narrow layout, and keyboard navigation; verify those checks pass on built Web artifacts.

## 4. Remaining management pages

- [x] 4.1 Apply the shared hierarchy and interaction patterns to credentials, device authorization, and profile without changing their underlying actions; verify real/fixture-backed loading, empty, error, confirmation, and uncertain-result states for each page.
- [x] 4.2 Review account identity, sign-out, active navigation, page focus, mobile menu, and the device-code approval warning as one flow; verify history, direct links, Escape, reduced motion, and keyboard focus in browser acceptance checks.

## 5. Cross-service acceptance

- [x] 5.1 Run frontend typecheck, lint, build, and production-artifact browser acceptance; record pass/fail/skipped states plus desktop, mobile, zoom, axe, and manual focus observations.
- [x] 5.2 Run isolated server integration and race checks plus HTTPS browser acceptance for the new read and existing owner actions; record separate results for local fixtures and actual services, including remaining environment gaps.

## Workflow follow-up

- After implementation and project acceptance, sync the delta to the main spec and archive this change under the repository workflow.
