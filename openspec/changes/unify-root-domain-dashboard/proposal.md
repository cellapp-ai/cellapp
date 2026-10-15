# Proposal

## Why

Cellapp currently splits the public website and owner console between `cellapp.ai` and `console.cellapp.ai`, while deployment guidance exposes a second CLI installation. A single product domain and an Agent-led deployment entry will make the path from discovery to management and publication easier to understand.

## What Changes

- Serve the public website at `https://cellapp.ai/` and make `https://cellapp.ai/dashboard` the canonical owner entry, with all console pages, device approval, assets, OAuth, browser APIs, and deployment APIs routed on the same host. Keep visitor applications on the isolated `ce11.app` domain.
- **BREAKING** Retire `console.cellapp.ai` as an API origin. Old browser bookmarks may redirect to their new management destinations, but old CLI/API requests will not be served there.
- **BREAKING** Change the deployment client's default origin to `https://cellapp.ai`. Migrate only the known official old origin in local project associations without changing app IDs, pending operations, or custom origins. Require a fresh, explicit device approval on the new origin; do not silently copy old deployment tokens.
- Preserve account-scoped deployment authorization: one approved credential per local environment can manage all applications owned by that account until expiration or revocation. The user sees the Skill and browser approval; the Agent runs the published deployment runtime on demand without requiring a global CLI installation or teaching CLI commands.
- Coordinate Cloudflare Pages, Worker, Cloud Run, GitHub OAuth, documentation, and cross-service acceptance so public pages and private management routes can share the root host without weakening the visitor-domain boundary.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-website`: Share the root host with `/dashboard` while keeping public pages and the Agent deployment entry accessible without a console session.
- `web-control-plane`: Move management deep links and login return paths beneath `/dashboard`; retain browser API and CLI protocol paths on the root host with exact routing and origin checks.
- `skill-deployment`: Make Agent-managed, on-demand client execution the default user journey and migrate the official old project origin safely.
- `deployment-authorization`: Keep credentials account-scoped across projects while requiring fresh approval for the new origin and isolating local secrets from project state.
- `cloud-run-deployment`: Route the root host's private paths to the authenticated Cloud Run origin and verify the coordinated website, edge, and service cutover.

## Impact

Affected areas include `apps/website`, `apps/web`, Go hosting/auth routes, Cloudflare Worker routing and deployment configuration, the public `skills` submodule, OAuth callback configuration, client local state and credential handling, and current operational documentation. No database schema change or visitor application hostname change is intended. Publication of Cloud Run, Cloudflare Pages/Worker, and the npm client is a separate release activity after implementation and verification.
