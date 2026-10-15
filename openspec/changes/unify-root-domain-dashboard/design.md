# Design

## Context

See [proposal.md](proposal.md) for motivation and the five delta specs for behavior. Today Cloudflare Pages serves the independently built public site on `cellapp.ai`; a Worker forwards all control-host requests from `console.cellapp.ai` to the authenticated Cloud Run origin and separately protects `*.ce11.app`. The Go server selects control versus visitor behavior by exact Host. The console's Vite build and Go routes assume root-level page and asset paths. OAuth uses `${CONTROL_ORIGIN}/auth/callback`, device links use `${CONTROL_ORIGIN}/device`, and the CLI stores both project origin and origin-keyed local credentials.

## Goals / Non-Goals

**Goals:**

- Keep the existing independent Pages and Go release artifacts while presenting one root-domain product entry.
- Preserve the existing Bearer HTTP protocol and owner-scoped credential model; change its host and browser page locations.
- Make migration of official old project associations safe, repeatable, and explicit about reauthorization.

**Non-Goals:**

- Move the website into the Go image, serve user builds from the platform server, or change `ce11.app` application hosts.
- Replace the deployment runtime with Agent-authored HTTP calls, remove its manual/developer command entry, or turn browser cookies into deployment credentials.
- Keep credential-bearing APIs on `console.cellapp.ai` after cutover or automatically transfer old local tokens.

## Decisions

### 1. Route the apex by a declared path inventory

Keep the Pages custom domain for public paths. Attach Worker routes to the apex for control paths, including `/dashboard` and descendants; `/api/console/*`; `/auth/*`; `/device-authorizations` and polling; the existing Bearer `/apps` tree, `/limits`, `/credentials/current`, and `/a/*`; and any explicit service probe path needed for release checks. The Worker forwards these paths to Cloud Run with the existing authenticated original-host assertion. Root `/`, `/deploy/`, `/pricing/`, `/docs/`, `/blog/`, and public assets stay with Pages. Add an inventory test comparing registered Go endpoints, browser requests, and edge route patterns so a new private endpoint cannot silently fall through to Pages. Prefer path-specific routes over an apex catch-all: Pages can remain the public origin without introducing a second website proxy hop. Validate route precedence against the actual Pages custom domain in a candidate environment before cutover.

Avoid public/console asset collisions by building the console with a `/dashboard/` asset base and serving its assets under `/dashboard/assets/` and `/dashboard/logos/`. Never route arbitrary `/assets/*` or `/logos/*` from the apex to the control service. Reject unknown dashboard descendants and API routes rather than returning index HTML. Keep control responses private/no-store and public website files on their own cache policy.

### 2. Make `/dashboard` the only management page namespace

Move management page links to `/dashboard`, `/dashboard/applications`, `/dashboard/applications/:id`, `/dashboard/credentials`, `/dashboard/device`, `/dashboard/profile`, and `/dashboard/login`. Keep `/api/console/*` and existing Bearer API paths stable so this is a host migration, not a request-schema migration. Change the device verification URI to `/dashboard/device?code=...`; set `CONTROL_ORIGIN=https://cellapp.ai` and the GitHub OAuth callback to `https://cellapp.ai/auth/callback`. Restrict login returns to known dashboard paths plus the existing app handoff, preserve device query data, and retain host-only `__Host-` cookies. Existing cookies on the old host do not cross to the apex, so browser login occurs again.

If the retired host remains routed for bookmark migration, only allow safe GET/HEAD browser management links to redirect to mapped apex paths with validated query strings. Return an explicit failure for old API methods and paths; never forward authorization headers or request bodies across hosts. The cutover can also retire the host entirely after documenting that browser redirects are unavailable; API retirement is mandatory either way.

### 3. Keep the deployment runtime, remove user-facing setup

The Skill and public Agent guide invoke a published, verified client through on-demand package execution. The Agent chooses the project build and output, obtains the runtime, handles setup errors, runs deployment, and reports only the approval URL/code and final result to the user. Do not require a global installation or instruct the user to type `cellapp` commands. The CLI remains the tested implementation of artifact checks, secret storage, idempotency, retries, and application data operations, as well as an optional manual/developer interface. A pure prose/API implementation would duplicate sensitive deployment behavior across Agents.

Use the new root origin as the default for fresh projects. When no explicit custom origin overrides selection, normalize the exact official old origin in supported current and legacy project files before their consistency check, preserving IDs, pending fingerprints/request IDs, build settings, and unknown compatible fields. Write migrated non-secret state atomically only after a safe comparison; do not alter arbitrary origins or treat two different app IDs as equivalent. An explicit old `--origin` or environment override produces an actionable retired-origin error instead of a silent rewrite. The local credential filename is origin-derived: do not copy its old token. The first new-origin operation starts device authorization and saves a new owner-scoped token, reusable across that owner's projects. If the user authorizes a different account, the server's ownership check stops the update without creating a new app.

### 4. Release and verify the coupled boundary

Prepare the compatible Go/Web revision, Worker path routes, Pages content, GitHub OAuth callback, and published client/Skill before moving production traffic. Use an isolated candidate hostname or equivalent controlled edge route to test the combined behavior without replacing production prematurely. Verify the new website and dashboard, Secure Cookie scope, OAuth callback, safe return, device approval, browser and Bearer APIs, a migrated existing project, and visitor access on `ce11.app`. Record versions independently. Enable apex control routes only after the candidate is sound; then stop old API routing and run live checks. Revert the coordinated service/edge/Pages configuration if acceptance fails, retaining database and release objects. Since the old API is intentionally retired, rollback of new-origin clients requires restoring a compatible old entry temporarily or pausing deployment and reporting the recovery path; do not claim a service-only rollback is sufficient.

## Risks / Trade-offs

- [A declared edge path is missing or matches a public page] → Maintain an endpoint/route inventory test and verify public 404, dashboard deep links, assets, API methods, and Pages path precedence in the candidate environment.
- [Shared-host cookies or cache expose owner data] → Keep `__Host-` cookies, explicit Origin checks, `private, no-store` control responses, and negative browser/visitor tests.
- [Existing project is stranded by old origin or mismatched account] → Migrate only the exact official origin, preserve IDs and pending state, require new approval, and fail clearly on ownership mismatch or custom-origin conflict.
- [New client reaches apex before compatible routes are active] → Publish and switch the client only after root-host service and edge acceptance; stage versions and keep a coordinated rollback runbook.
- [On-demand package execution still needs Node/npm and network access] → Agent verifies prerequisites and reports a concrete blocker; no fallback to ad hoc upload logic.

## Migration Plan

1. Build and verify the Go/Web route changes and client migration in isolated environments; prepare Pages and Worker candidate routing without switching the production apex.
2. Configure the root control origin and GitHub callback, deploy the compatible service and edge candidate, then run end-to-end approval and existing-project tests alongside public-site and visitor-domain checks.
3. Activate root control routes, update public deployment guidance/Skill and client default, disable the old API host, and verify live behavior. Browser bookmark redirects are optional but cannot restore old API behavior.
4. On failure, revert the coordinated route/service/website versions while preserving data and release objects; explicitly pause or restore a compatible deployment entry until the client and active host match again.
