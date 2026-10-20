# Design

## Context

See `proposal.md` for the motivation and `specs/web-control-plane/spec.md` for behavior. `apps/web` is a React/TypeScript owner console with a 260 px sidebar, centered 1180 px content container, shared panel styles, and one large `App.tsx`. The application detail already reads owner-scoped data state from `GET /api/console/apps/{app}/data`, but that response contains only state, version, and digest. The separate application data database stores each successfully applied, versioned manifest, and `appdata.Latest` already reads it. The existing design standard requires a dark Cellapp palette, local fonts, visible focus, responsive behavior, and truthful data.

## Goals / Non-Goals

**Goals:**

- Make each owner task readable and actionable on wide screens, mobile widths, keyboard navigation, and zoom while retaining the existing brand system and management routes.
- Provide a small, read-only browser contract for the latest applied table structure, with explicit ownership and failure boundaries.
- Preserve current session, deployment, application data write, and visitor authorization behavior.

**Non-Goals:**

- A theme switch, chart dashboard, usage estimates, application creation, browser deployment, or artificial preview data.
- Table row browsing, CRUD, schema editing, SQL execution, or database administration in the dashboard.
- New component library or visual dependency unless implementation proves a concrete need.

## Decisions

### 1. One restrained visual system across the console

Retain the existing ink/panel/paper/lime/blue tokens, Space Grotesk and DM Mono, and Cellapp assets. Refine surface elevation through subtle tone and border contrast rather than gradients, large shadows, or repeated cards. Keep lime for focus, the active destination, and primary action; keep status legible in text independently of color. Use a stable desktop sidebar and a content area aligned near it, with a readable maximum width that does not create a large central gap on ultrawide displays. Keep the existing mobile menu behavior and adapt the content layout at the current 800 px boundary.

Alternative: switch to a light generic SaaS theme. This would conflict with the user's chosen dark direction and the current brand standard.

### 2. Organize pages around real decisions

The applications page uses a compact heading and a structured list: name and `@namespace/slug`, current URL, publication state, and explicit `Manage`/`Open application` links. A row is not a giant link containing other interactive controls. On narrow screens, the same information reflows into readable stacked rows without hiding actions. No dashboard metrics are introduced; a count may be derived from the loaded list.

Application detail puts identity, current URL, publication state, and `Open application` at the top. Group release, tables, access, and settings in stable sections with in-page navigation if useful; avoid a new nested route family only to switch visual sections. Keep destructive actions separate from primary navigation and preserve existing confirmation, uncertain-write recovery, and one-time key lifecycle. Credentials, device authorization, and profile reuse the same page heading, section, status, action, and feedback patterns. The device approval screen keeps the code comparison and explicit decision foremost.

Alternative: add many summary cards and ornamental charts. Those would increase noise and imply data or actions that are not currently provided.

### 3. Add a dedicated owner-only structure read

Add `GET /api/console/apps/{app}/data/structure` rather than changing the existing `/data` status response. The handler uses the same browser session wrapper and owned-app lookup as other console reads, then calls `appdata.Latest` in the separate application data database. Return only the applied version, application data readiness state, and a bounded projection of manifest tables: names; column name/type/nullable; primary keys; index columns/uniqueness; select/insert/update/delete policy modes and owner column where applicable. Do not expose raw database schema names, connection details, the manifest's destructive-apply flag, rows, or control metadata. A missing applied manifest returns version 0 and an empty table array. The existing manifest limits bound the response; still validate the response in the browser client.

The route inherits owner-only 404 semantics, private no-store responses, and no visitor-host access. If the data database read fails, return a stable unavailable error; the UI must not turn it into an empty list. If readiness is pending or error but the latest manifest is readable, show that definition with a visible readiness warning rather than calling it a verified live structure. The status route remains compatible for existing callers.

Alternative: browser-side table introspection through the application's `/rest/v1` endpoint. That would couple owner structure inspection to visitor row policies and could fetch data rows. Returning the applied manifest projection is safer and reflects the versioned source of truth.

### 4. Keep interactions semantic and recoverable

Use links for navigation and buttons for operations. Make hover, focus, disabled, loading, empty, success, and error states visibly distinct. Preserve actual hrefs, history, refresh, main-content focus, mobile menu Escape, and reduced-motion behavior. Avoid hidden hover-only controls. Refresh operations are reads; uncertain writes remain explicit and are never retried automatically. Keep the currently revealed access key mounted only in its existing detail-page lifecycle and prevent visual section changes from persisting it after leaving the application.

Alternative: rely on animation and icon-only controls to create polish. This would weaken discoverability and keyboard use.

## Risks / Trade-offs

- [Wide-screen refinements harm small-screen density] → Review 320, 390, 768, 800/801, 1440 px, long Chinese names and URLs, and 200% zoom against the production build.
- [Structure metadata leaks across applications] → Reuse owner session and per-app lookup; test signed-out, cross-owner, and visitor-host requests, and return no structure on failure.
- [Applied definition is mistaken for verified database readiness] → Return readiness separately and label pending/error states explicitly.
- [Page redesign changes sensitive action behavior] → Retain confirmation, focus restoration, key clearing, and uncertain-result messaging in behavior tests.
- [A style-only pass leaves other pages inconsistent] → Include credentials, authorization, and profile in one visual review and browser acceptance pass.

## Migration Plan

1. Add and verify the read-only structure contract and client parser without changing the existing status endpoint or table write APIs.
2. Refine the shared shell and page patterns, then rebuild application list/detail and the remaining management pages using actual responses.
3. Update the Web API and design documentation, run frontend checks and isolated server/browser authorization checks, and inspect production build layouts and accessibility states.
4. Deploy Go and Web artifacts together through the existing release process. Roll back the paired service and Web revision if acceptance fails; no schema migration or data rollback is required.
