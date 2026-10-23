# Design

## Context

The shared header is rendered both into static HTML and in the browser. Its wordmark is a Base64-embedded SVG so the single-file bundle works. The dashboard already checks the same-origin `/api/console/session` endpoint and shows its sign-in flow on a 401 response. The Pages preview domain does not route that API to the control service.

## Goals / Non-Goals

**Goals:** Preserve static rendering and hydration, keep a usable action before the session check completes, and show the homepage state on the logo without replacing its supplied outline.

**Non-Goals:** Change the session API, authentication flow, pricing page, or dashboard implementation.

## Decisions

- Remove the Home text link and mark the header wordmark as the current page on `/`. Overlay the two existing slash outlines in the site's accent color only on that route. This keeps the original embedded logo and its portable rendering intact.
- Render “Deploy for free” initially in static HTML. On the production `cellapp.ai` host, request `/api/console/session` with same-origin credentials and no caching. Switch to “Dashboard” only after a successful JSON session response with a valid owner ID. A 401 or network error keeps the initial label; Pages preview and local-file mode do not make a session request. Abort an in-flight request when the header unmounts.
- Keep the action target `/dashboard` in both states. The control console already turns a signed-out dashboard visit into its login screen, and the shared Link component handles this target in the portable file bundle.
- Use browser acceptance checks for the static state, simulated authenticated state, current-page semantics, narrow layouts, and single-file navigation.

## Risks / Trade-offs

- A signed-in visitor may briefly see “Deploy for free” while the session request completes. The static-first label avoids hiding the action and never incorrectly claims the visitor is signed in.
- The Pages preview domain cannot confirm a production session. Its action retains the signed-out label; production `cellapp.ai` can reach the same-origin API.

## Migration Plan

Build and verify the website, merge and push the main repository, upload the complete `dist/` folder to Cloudflare Pages, and confirm the navigation on the Pages deployment and `cellapp.ai`. A rollback can redeploy the previous Pages output without changing the control service.
