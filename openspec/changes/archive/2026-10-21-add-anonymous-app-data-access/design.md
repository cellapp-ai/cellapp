# Design

## Context

See `proposal.md` for motivation. `app-key-access` currently makes login mandatory even for public applications. The Go gateway calls `appSession` before static delivery and in `serveData`. `serveData` issues a per-app PostgREST JWT role and user ID only after application admission. Table manifests support `none`, `all-admitted-users` and `own-rows`; the current per-app API role is shared by all admitted users. Production static delivery uses a signed, 60-second edge grant bound to an app-session cookie. The control database has numbered, checksummed migrations. The app-data database and PostgREST remain private behind Go.

## Goals / Non-Goals

**Goals:** An explicit operator gate for anonymous inserts, separate owner gates for anonymous pages/read/insert, explicit table-level anonymous policies, and a viable public-page path through both direct Go and edge delivery. Every anonymous data request uses current settings, while existing applications remain login-only without migration work by owners.

**Non-Goals:** Anonymous update/delete, anonymous ownership of rows, a stable anonymous identity, public PostgREST or object storage, a challenge/CAPTCHA service, or a general runtime configuration API for operators.

## Decisions

### Configuration and application state

Add `ALLOW_ANONYMOUS_APP_DATA_WRITES=true|false` to Go service configuration, with absent meaning false and other values rejected. It is an operator deployment setting: a changed value becomes effective when the service rolls to the new configuration. This matches `PUBLICATION_PAUSED` and avoids storing a platform-wide security switch in a user-manageable table. Document the rollout and rollback in environment and operations examples. The owner API returns the effective write state so the dashboard can distinguish “owner enabled” from “operator enabled.”

Add three booleans to `apps` through the next numbered control migration: `anonymous_pages`, `anonymous_read`, `anonymous_write`, all `NOT NULL DEFAULT false`. A constraint or transaction-checked invariant requires public visibility for any true value and `anonymous_pages` for read/write. Changing visibility to restricted atomically clears all three; changing an app setting checks owner and current visibility within one transaction. Do not reinterpret the existing `public` value, since it already means any signed-in user. This new layer avoids silently exposing every existing public application.

Alternative considered: one `anonymous` toggle. It cannot express read-only public pages or submission-only forms and would make accidental table disclosure more likely.

### Table policies and database roles

Extend the versioned manifest with `policies.anonymousSelect` and `policies.anonymousInsert`, using explicit `none` or `all` values; omitted means `none`. Keep the current signed-in policy fields unchanged for old clients. The provisioner creates a separate per-app anonymous API role with `NOLOGIN` and `NOBYPASSRLS`, grants it only to the private authenticator, and gives it schema/table privileges only where the manifest explicitly permits. Anonymous row policies target that role alone. Existing policies must be scoped to the signed-in role so `all-admitted-users` cannot leak through a shared role. The gateway signs a short JWT naming the anonymous role without `app_user_id`; it never issues an app user or grant for anonymous requests. Reconcile existing app schemas and roles idempotently before marking them ready. Do not use PostgREST's global anon role or client-supplied JWT as authorization.

An anonymous insert without select permission returns no inserted representation. The gateway forces `Prefer: return=minimal` on anonymous inserts unless both application and table anonymous read gates are enabled; this avoids a successful insert being reported as failure after the write. With those read gates enabled, `return=representation` remains available. Anonymous reads need both application and table grants. Write operation maps only to POST insert; PATCH/DELETE remain signed-in only.

Alternative considered: reuse the existing per-app API role with a missing user ID. Existing `all-admitted-users` policies use a broad `true` expression, so that would expose tables unintentionally.

### Gateway admission and request limits

Separate signed-in and anonymous admission at the app gateway. If a valid app-session cookie is supplied, keep the signed-in path and do not fall back to anonymous after session or grant denial. A request with no app session may use the anonymous path only after reading current public visibility, app state and operation settings. For data, check the operator gate on every anonymous POST, enforce same-host `Origin`, method and body limits, then proxy with the anonymous role and server-selected schema. Do not create a platform or app user record. Preserve source isolation: control-origin cookies, client Authorization, profile headers and claimed IDs are ignored.

Use existing per-IP/app request limiting as a starting point, adding a dedicated, stricter anonymous data-write bucket plus anonymous page/data read limits. Bound request concurrency and response size as for existing data traffic. IP is an abuse-control key, never an identity or billing unit. Keep anonymous requests out of `app_visitor_admissions`, which has a non-null platform user ID and guarantees distinct signed-in visitors only. Expose separate anonymous request counts to operations if existing metrics permit; do not claim unique anonymous visitors.

Alternative considered: mint anonymous user IDs and count them against the monthly visitor cap. Cookies are resettable and IPs can be shared, so neither gives a reliable unique visitor count or fair monthly limit.

### Static page and edge delivery

The direct Go gateway serves pages/assets anonymously only for an active, published public app with `anonymous_pages=true`. Its SPA and path validation remain unchanged. The edge grant gains an explicit signed access mode. A signed-in grant stays bound to the existing session digest. An anonymous grant has no session digest and is accepted only as a short-lived grant for an exact host, app and deployment; the edge never treats it as a login or data credential. The private R2 object lookup remains behind grant verification. The edge obtains a new anonymous grant from the protected origin when an anonymous page request has no usable grant, using the current application state. The edge still forwards `/rest/v1` and control mutations to Go. Disabling anonymous pages stops new grants immediately; issued grants expire within 60 seconds. The mode must be signed and validated so a user grant cannot be rewritten as anonymous.

Alternative considered: bypass edge authorization for anonymous-enabled apps. That would make private release objects and caches rely on a broader public path and weaken the current isolation model.

### Owner and client surfaces

Add owner-scoped browser and Bearer API reads/writes for the three settings, including effective operator state. Reuse the existing ownership and same-origin write checks; use explicit booleans and stable validation/error codes. The dashboard shows separate page/read/write controls, their dependencies, and a confirmation before opening access. The CLI and shared contract expose owner settings and new manifest policy fields. Update Skill examples for anonymous form and public read cases, including a no-login browser check and a negative read/update/delete check. Keep current signed-in `/_cellapp/me` semantics; an anonymous page receives an unauthenticated result, never a synthetic user ID.

## Risks / Trade-offs

- [Automated anonymous submissions consume storage or backend capacity] → Default all gates off; enforce dedicated rate, size and concurrency bounds; preserve operator kill switch and per-app owner off switch; observe rejection and usage metrics.
- [A broad table policy leaks rows] → Separate database role and explicit anonymous table policy; test old manifests, cross-app attempts and signed-in policy isolation against real PostgreSQL/PostgREST.
- [Invalid session silently receives anonymous rights] → Choose anonymous path only with no app-session credential; test expired, revoked and forged cookies.
- [A setting changes while requests run] → Check settings in the authorization transaction for each data request; serialize owner changes with the row decision so a completed disable blocks later writes. A request already authorized before the change may finish; document this narrow in-flight window.
- [Operator switch takes time to roll out] → Treat it as a deployment configuration rather than pretending it is an instant remote toggle; verify every serving revision has the intended value before reporting the switch effective.
- [Anonymous public pages are copyable] → Preserve private storage and 60-second grants, but acknowledge that content already downloaded by visitors cannot be recalled.

## Migration Plan

1. Apply the next checksummed control migration with false defaults and invariants. Upgrade the app-data provisioner and reconcile roles/policies while all anonymous gates remain off. Verify existing manifests and signed-in CRUD still work.
2. Release compatible Go, edge Worker, dashboard, CLI/contracts and Skill in the repository's service/client order. Keep `ALLOW_ANONYMOUS_APP_DATA_WRITES=false` throughout initial rollout. Verify private and anonymous-negative paths first.
3. On isolated test resources, enable one public app's anonymous pages/read/insert and explicit table policies, then enable the operator gate in a controlled service revision. Exercise real PostgREST, private storage and browser/edge paths, including disabling each gate and rollback to false.
4. Rollback by setting the operator gate false and disabling affected app settings before rolling back binaries. Additive control columns and per-app roles remain; do not drop stored submissions. Avoid rolling back to a version that ignores enabled anonymous settings while the edge still serves anonymous grants.
