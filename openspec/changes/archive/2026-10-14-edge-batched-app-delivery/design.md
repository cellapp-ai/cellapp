# Design

## Context

See `proposal.md`. `serveApp` currently performs a PostgreSQL transaction and `Storage.Get` for every static GET. The Cloudflare Worker forwards all requests with an authenticated original-host assertion and marks responses `private, no-store`. The S3 adapter already supports an explicit HTTPS endpoint; production uses GCS and local development uses MinIO. Application data routes, login handoff, grants, and the current per-file byte counter depend on the control database.

The existing specs promise immediate static-resource revocation and byte-limited distribution. The deltas in this change deliberately replace those behaviors with at most 60 seconds of static-grant validity and a monthly distinct-visitor admission limit. Data API authorization remains request-time.

## Goals / Non-Goals

**Goals:** Remove static response bodies and routine static-file authorization from Cloud Run, keep the bucket private, make ordinary one-minute renewal invisible to the visitor, and limit new reach to 1,000 authenticated people per app per UTC month. Preserve the current host, SPA, content-type, login, data API, and operator suspension behaviors within the stated 60-second bound.

**Non-Goals:** Anonymous application access, immediate static-resource revocation, exact byte accounting, preventing a visitor from copying content after it has been delivered, and moving dynamic application data or the console to R2.

## Decisions

### 1. Route static GET/HEAD through the existing Worker

Keep the control domain and all application auth, `/_cellapp/*`, `/rest/v1/*`, and mutation routes forwarded to Cloud Run. For an application-host static GET/HEAD, the Worker validates a short grant before any object or cache lookup. Without a usable grant it calls an edge-only admission endpoint on Cloud Run using the existing original-host proof and host-scoped application session cookie. Cloud Run applies `appSession`, `appAccess`, application state and visitor admission, then returns a signed grant or the existing login/access outcome. The Worker attaches the grant cookie to the requested static response. A normal HTML response sets the cookie before the browser loads its referenced assets; a direct asset request also goes through admission.

The grant is an `HttpOnly`, `Secure`, `SameSite=Lax`, host-only `__Host-` cookie with a maximum validity of 60 seconds. Its signed claims include the exact request host, app ID, active deployment ID, SPA setting, expiry, key ID and a digest of the opaque app-session cookie. The Worker verifies the signature and that the accompanying app cookie still matches the digest. Only Cloud Run holds the signing private key; the Worker has the verification public key. Use a separate key from the origin assertion and application secrets, and overlap old/new verification keys during rotation. The grant expiry cannot exceed the app-session expiry. On the next request after expiry, the Worker renews on demand; an in-flight response is not cut off. Parallel renewal may duplicate admission calls, which must be idempotent. Reissue before expiry on a request when practical, without a background browser timer.

An earlier option was a per-file authorization callback. It preserves immediate revocation but leaves Cloud Run and PostgreSQL on every asset request. A longer signed grant would reduce callbacks further but exceed the accepted suspension delay. A KV-based revocation list is unnecessary for the accepted 60-second bound and has eventual-consistency behavior.

### 2. Serve only the granted release from private R2

Bind one private R2 bucket to the Worker. Use the current deployment-ID/path object-key layout and the same path safety rules as the Go gateway; derive the key only from verified grant claims and normalized request path. Each published release also has a private platform manifest object listing allowed paths; the copy tool creates it for older releases. The Worker checks this allowlist before reading a hashed object key, so an unlisted object in the same prefix cannot be served. The Worker handles SPA fallback only for HTML navigation to an extensionless missing path, responds with the same content types, and returns 404 for missing assets. GET streams the object; HEAD and conditional requests expose no body and still require the grant. Authenticated, immutable release objects may be cached at the edge under deployment ID and path, with authorization checked **before** every cache lookup. The first implementation reads R2 directly without a shared asset cache. Browser-facing responses remain private; never cache login, grant, data, or denied responses as public assets. Cloudflare's Cache API is local to each data center, so cache hits are an optimization rather than a correctness dependency.

Cloud Run uploads and cleanup continue through the S3 interface, configured to use R2's account endpoint with region `auto` and scoped bucket credentials. The Worker uses its R2 binding, not these credentials. Local MinIO behavior remains. A Worker-to-GCS fetch was considered, but would still transfer every miss out of GCS to Cloudflare and require another origin credential or signed URL path. Public R2 URLs were rejected because they bypass the grant.

### 3. Count distinct admitted visitors at grant issue and data entry

Add a new migration for a ledger keyed by `(app_id, UTC-month, platform_user_id)` and a per-app-month admitted count. Under a row lock for that app/month, check whether the user is already recorded; if not, reject at the configured `MAX_MONTHLY_VISITORS` (default 1,000) or insert once and increment. Exempt the owner. Invoke the same admission path before an application data-only request. Unauthorized attempts do not create ledger entries. At the cap, previously admitted users may renew; new users receive `429` with `resetAt` at the next UTC month. This makes the **admission count** concurrency-safe without metering every asset byte. The existing `usage.bytes` history remains readable but stops gating new delivery after cutover; `/limits`, contracts, docs, and client displays replace `trafficBytes` with `monthlyVisitors` in a coordinated compatibility update.

Keep Cloudflare's edge request limits and the existing Cloud Run dynamic-route limits. The monthly visitor cap is not a substitute for per-source request throttling: one admitted user can still generate many requests. Budget and monitor Worker and R2 usage; set the security-critical Worker route to fail closed if its platform quota is exhausted.

### 4. Bound state changes and make failures explicit

Every new grant queries the current app state and grants. An already issued grant can continue to serve its specific release for at most 60 seconds after revocation, visibility change, deletion, suspension, or replacement of the active deployment. Data API requests retain their immediate checks. Cleanup must retain an old release for longer than the grant lifetime, so an in-flight granted read does not turn into a missing-object error. On authorization, signature, R2, or required cache dependency failure, reject protected content rather than bypassing authorization. Expose metrics for grant issue/renewal, denial by reason, R2 errors, and monthly visitor admission without logging cookies or token material.

## Risks / Trade-offs

- **One-minute state lag** → Keep absolute expiry at or below 60 seconds, test clock boundaries and rotation, and retain operator suspension plus edge source throttling. This cannot prevent copies made by already admitted visitors.
- **Renewal spike or origin outage** → Existing sessions renew without visible login, but the first request after expiry needs a Cloud Run round trip. Make issuance idempotent, test concurrent asset fetches, and fail closed when origin is unavailable.
- **Cross-host or cache leakage** → Bind grant to exact host, release and app-session digest; validate before cache lookup; keep R2 private and test alternate hosts, HEAD and conditional requests.
- **Storage cutover breaks old revisions** → Keep GCS and pause deployment creation, upload and publication for the entire rollback window. Both the new Go revision and Worker block publication. When reverting to an old Go revision, deploy the same Worker in origin-static mode while retaining its publication block; reverting to an older Worker lacking that block would reopen one-sided GCS writes. Do not remove GCS on initial success.
- **Monthly visitors do not bound byte volume** → Preserve edge rate limiting and operational monitoring; revisit byte or request budgets separately if actual traffic warrants them.
- **Worker quota can interrupt service** → Monitor the account's request budget, set fail-closed behavior, and estimate a paid plan before scaling beyond the free tier. This is an availability trade-off rather than permission to bypass the Worker.

## Migration Plan

1. Add the visitor-ledger migration and deploy a server that supports both the existing Cloud Run static path and the new edge admission endpoint. Keep the existing Worker and GCS path active while checking database behavior and old-client compatibility.
2. Create a private R2 bucket and least-privilege S3 credentials for the server, a Worker binding, and a dedicated signing key pair. Probe authenticated S3 writes/reads and anonymous denial without printing credentials.
3. Copy all active and rollback-retained objects from GCS to R2 using deployment IDs as keys. Compare paths, lengths and SHA-256 digests against manifests. Mirror new uploads to both stores throughout the rollback window, or keep new deployment publication paused throughout that window; do not switch if verification fails. Keep GCS until both the rollback window and new-write compatibility period are complete.
4. Deploy and test a candidate Worker/Cloud Run pair on an isolated test route, including real login, one-minute renewal, cap behavior, revoked and unknown hosts, data API, and R2 failure. Switch the visitor route only after both sides agree on signing keys and object layout. Keep a quick route-level fallback to the previous Worker and server revision while the old storage copy is current.
5. Observe Cloud Run request/egress reduction, R2/Worker requests, renewal errors and `429` admissions. Retire old byte enforcement and GCS only after verified rollback coverage and documented acceptance. A failed cutover restores the old Worker and Cloud Run traffic without deleting database migrations or either bucket.
