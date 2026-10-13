# Design

## Context

See `proposal.md` for motivation and the six delta specs for behavior. Today the Go service uses one PostgreSQL connection pool and an initial `IF NOT EXISTS` migration without version tracking. Its `owners` and browser sessions serve control-domain GitHub login; app-domain access is an anonymous share-key cookie. The app gateway serves static files only. The published CLI and Skill in the `skills` submodule handle static deployment, and the web console handles owner operations. Control and app origins are on different registrable domains, so their cookies cannot be shared. Another worktree is separately planning a Supabase-backed control environment; no design here depends on where that control database is hosted.

## Goals / Non-Goals

**Goals:** Keep platform control data and user-created app tables in distinct PostgreSQL databases; enforce app and user boundaries in both the gateway and database; make static frontends usable with a documented `supabase-js` table CRUD subset; give owners a repeatable table and policy workflow through CLI and Skill; migrate existing keys without granting anonymous sessions identity.

**Non-Goals:** Full Supabase API equivalence, Supabase Auth/Realtime/Storage/RPC, direct PostgreSQL access, arbitrary app server or SQL function execution, payment collection, automatic purchase grants, or cross-app user identifiers. A public app means accessible to any signed-in Cellapp user, not anonymous web access.

## Decisions

### 0. Public application namespace

An owner has one personal namespace handle; each application has an alias unique within that namespace. Display names remain labels and may repeat. A new owner may choose a lowercase ASCII handle on first creation, and each new app may choose an alias; omitted handles use an unrelated random namespace ID, and omitted aliases derive from the display name with a conflict suffix. Existing owners and apps receive generated handles and `app-<id>` aliases through an additive control migration. Both identifiers are immutable in this release. Deleted aliases stay occupied to prevent stale share links being rebound to another app.

The reference `@namespace/alias` is shown in the console; its HTTPS share URL is `/a/namespace/alias` on the control origin. The resolver only redirects to the existing app-ID host. Sessions, cookies, admission, data roles, schemas and CLI project associations continue to use immutable app IDs. The random generated namespace is separate from the platform owner ID so sharing an app does not reveal that ID. This leaves room for explicit team namespaces later without changing current app hosts.

### 1. Separate database, schema, and roles

Configure a second PostgreSQL DSN for application data. It names a database distinct from the control DSN; deployment validation rejects the same database target even if both are in one cluster. The control runtime role has no app-data table access. A narrowly privileged provisioner creates one deterministic schema per app ID and corresponding non-login API and DDL roles. The API role gets `USAGE` only on its own schema and only the table privileges granted by policy. The DDL role cannot create schemas, roles, extensions, foreign data wrappers or objects outside its schema. Revoke ambient `PUBLIC` access and default function execution; keep PostgREST infrastructure objects private. App identifiers are never accepted as raw SQL identifiers without strict validation and quoting. User-defined relation names are scoped to each app schema: the same `users` table name may exist in several apps without referring to platform users. Manifest validation rejects duplicate names within an app, the reserved `rpc` API path, PostgreSQL system column names, and names that collide with generated indexes before a migration starts. The CLI and server enforce the same rules.

The Go process coordinates provisioning with a state record in the control database. Create the control app row first with a pending data state, provision idempotently, then mark ready only after schema, roles and API visibility are verified. Publication and data management cannot claim full readiness while provisioning is incomplete. Retriable reconciliation handles partial failure because PostgreSQL cannot transact across two databases. On delete, mark the control app deleted first so gateway and data requests fail immediately; asynchronously drop only that app's schema and roles, recording retry state. Backups and restore procedures include both databases and the object store and check control-to-schema consistency.

Alternative: one database with app schemas. Rejected because it does not meet the requested independent control/app database boundary. One database per app is stronger isolation but creates a disproportionate connection, migration and operations cost at current scale; per-app schemas plus roles meet the table-isolation requirement.

### 2. PostgREST behind the Go gateway

Run an internal PostgREST instance against the app-data database, not reachable from public networks. The Go gateway owns `https://<app-host>/rest/v1/*`: it validates host, platform session, app state and grant on every request, strips client-supplied authorization/profile/identity headers, selects the app schema, and issues a short-lived internal JWT containing only the app API role and application-scoped user ID. PostgREST verifies this JWT, impersonates that app role and applies PostgreSQL grants/RLS. Bind the selected schema on the server side; do not trust a client `Accept-Profile` or `Content-Profile`. The frontend receives only its app URL and a non-secret publishable identifier required by `createClient`; that value never authenticates or authorizes a request. Reject unsupported paths and methods rather than proxying them to PostgREST.

Pin and test an explicit `supabase-js` major/minor range and the `select`/`insert`/`update`/`delete`, column selection, equality filter, ordering, pagination, and write-return behavior in the public contract. Document exact header, result, error and unsupported-method behavior. PostgREST's schema list and cache must be updated after provisioning or table changes, with a readiness probe before reporting success. Use internal JWT signing material separate from the existing session secret; rotate with a controlled overlap. Apply request-body, row, response-size, statement-timeout, connection-pool and per-app rate limits. Data-write storage limits need database-enforced accounting or reservations so concurrent writes cannot exceed the configured bound.

Alternative: implement the PostgREST query dialect in Go. Rejected because even table-only `supabase-js` compatibility includes filtering, response preferences, error shapes and pagination that would be easy to diverge from. The internal PostgREST process adds deployment and schema-cache maintenance; the gateway keeps the public service and authorization boundary in Go. PostgREST supports multi-schema exposure, role-based authorization and reloadable schema configuration, but these remain independent security layers: roles restrict the database even if a profile header is forged.

### 3. Controlled table definitions and row policies

Use a versioned, non-secret project manifest, managed by CLI commands such as `cellapp data plan`, `cellapp data apply`, `cellapp data status`, and `cellapp data verify`. The manifest describes tables, columns, keys, indexes and per-operation policy modes. The initial policy modes are `none`, `all-admitted-users`, and `own-rows` using a required application-user column. The server validates identifiers/types/limits, computes a digest and generates transactional DDL and RLS; it does not execute arbitrary submitted SQL. `own-rows` policies use both `USING` and `WITH CHECK` against the server-signed app-user claim so clients cannot forge ownership on inserts or updates. New tables and operations default to deny. A separate app-data migration ledger tracks version and digest under an app-level lock; repeated identical apply is idempotent, changed content at an applied version is a conflict. Destructive table changes require a surfaced plan and explicit owner confirmation.

The platform control database needs a real version-tracked migration runner before adding control tables or fields; do not edit the historical `001_initial.sql` to upgrade existing installations. The app-data database uses its own bootstrap and per-app ledger. Owner credentials are authorized against the control database before any schema operation. Data DDL uses the app's restricted role, while provisioning uses an operator-only role not available to browser or CLI. CLI stores the manifest and app association, never database passwords or a service-role key.

Alternative: allow owner-submitted SQL migrations. Rejected for this stage because arbitrary functions, views, grants and extensions would substantially expand the security and resource-control surface and blur the static-only hosting boundary. Additional schema features can be added to the manifest when backed by validation and tests.

### 4. Platform login and app-scoped identity

Keep GitHub OAuth and the control-domain login as the platform identity entry. On app navigation, redirect unauthenticated users to a control-domain login handoff with a single-use, short-lived state bound to the app and a validated relative return path. After platform login, transfer only an opaque one-use handoff ticket to the app host; the app gateway consumes it before serving uploaded HTML, sets a Secure, HttpOnly, host-only app cookie, and redirects to a clean URL. The ticket cannot authorize another app, be replayed, or be read by uploaded JavaScript. The app session references the platform login session so global logout/expiry invalidates app access on subsequent requests; key rotation does not end a user grant.

Persist a random app-user ID mapping `(app_id, platform_user_id)` in the control database, rather than deriving it from the deployment secret; it remains stable across secret rotation and devices. The same platform account gets different IDs for different apps. A same-origin platform endpoint such as `/_cellapp/me` returns only that ID and access state after admission. No email, GitHub subject or global owner ID is exposed by default. The app API receives only the scoped ID in an internal claim. Limit login return targets to the initiating app, reject open redirects, preserve `SameSite` and origin checks, and never share the control cookie with app domains.

Alternative: expose the existing global `owners.id` or reuse the control cookie on app hosts. Rejected for privacy and because the domains intentionally isolate credentials.

### 5. Access grants and owner controls

Add explicit app visibility (`restricted` default, `public` optional), app-user grants, access requests and key-redemption records in the control database with unique constraints and transactional state changes. Owners always have access to their own apps. A public app admits any signed-in user without silently creating a lasting grant; switching it to restricted immediately leaves only explicit grants. Request approval, direct owner grant and key redemption converge on the same user-bound grant. Request approval/denial and redemption are idempotent under retries; concurrent revoke/delete cannot resurrect a grant. Grant source is recorded so a future purchase can issue a grant, but no payment API is built now.

Preserve each current app key hash as the initial access key. On upgrade, an old anonymous cookie is insufficient: prompt for platform login, then permit redemption of the still-valid old key. Key rotation updates the hash and invalidates future redemption only. Existing grants continue; owner revocation acts on a specific user and is checked on every page, resource and data request. Control UI shows real visibility, request and grant state; owner mutations retain `/api/console` session, Origin and resource-ownership checks. CLI operations and `skills/contracts` change together, with an explicit compatibility note for the breaking visitor flow and the changed key-reset semantics.

Alternative: keep key rotation as a global revocation. Rejected because the user confirmed grants must outlive a rotated key and future paid grants must not disappear during key maintenance.

### 6. Delivery and validation boundaries

The public Skill becomes the main operational workflow: inspect existing app association and manifest; plan table changes; apply with owner authorization; verify allowed and denied data operations; update static frontend using the documented `supabase-js` subset and `/_cellapp/me`; publish the frontend; report table and release results separately. The Go server remains responsible for all runtime identity and data enforcement. No custom application backend, build execution or private database credential is deployed with frontend assets.

Tests use isolated control and app-data databases plus a real PostgREST process to verify role grants, RLS, table-version conflicts, rollback, concurrent redemption/revoke/delete, cross-app denial, key migration, login handoff, CORS/origin protections, and `supabase-js` CRUD from a real browser. Run applicable race checks and record real versus stubbed identity/storage boundaries. Test deletion failure/retry and restore consistency. Do not treat a PostgREST mock as proof of protocol compatibility.

## Risks / Trade-offs

- [Schema/config cache can lag behind a successful migration] → Reload PostgREST configuration and cache, probe the new table through the app role, then mark the migration ready; expose explicit retry state on failure.
- [Two databases cannot commit app creation or deletion atomically] → Persist pending/retry states, make provision and cleanup idempotent, and let control-state denial take effect before asynchronous cleanup.
- [Public app admission may be mistaken for public data] → Keep every table operation deny-by-default and require explicit operation and row policies.
- [Internal role or JWT misconfiguration could cross app boundaries] → Use separate per-app roles, deny default schema privileges, no anonymous PostgREST role, internal-only network binding, gateway header stripping and direct database negative tests.
- [A lost or reset access key may be confused with revocation] → UI and Skill state that rotation stops new redemption; show separate per-user revoke controls.
- [Manifest expressiveness is limited] → Start with common table and policy forms; fail clearly for unsupported DDL instead of accepting unsafe SQL.
- [Concurrent writes or expensive queries can exhaust shared resources] → Enforce connection, statement, response and per-app write limits in the database and gateway, and test concurrency.

## Migration Plan

1. Add version tracking to the control migration runner and apply additive control tables/columns. Add the distinct app-data database, least-privilege roles, private PostgREST service, backups, health/readiness checks and configuration. Preserve old keys and control IDs.
2. Provision or reconcile schemas for existing apps idempotently. Keep legacy serving active until the login/grant cutover is verified. Backfill no anonymous identity or grants from old cookies; retain existing key hashes for signed-in redemption.
3. Ship coordinated Go gateway, control UI, CLI/Skill and contract changes. Switch app navigation to mandatory platform login and user grants, explicitly invalidate legacy anonymous sessions, then enable `/rest/v1` only after database isolation and compatibility checks pass. New apps default to restricted.
4. Verify isolated PostgreSQL and PostgREST, cross-user/app denial, old-key redemption, public/restricted switching, owner approval and revocation, deletion and cleanup, real browser CRUD, build/static deployment and documented restore procedures. Record unavailable external dependencies as unverified, not passed.
5. Rollback application binaries and frontend assets together. New database and additive control migrations remain in place; do not restore anonymous key-only access as a casual rollback because it would bypass user-bound grants. If an emergency legacy rollback is required, first disable app-data API and explicitly decide how to protect already-created app data and grants. Never drop the new database or schemas as a rollback step.
