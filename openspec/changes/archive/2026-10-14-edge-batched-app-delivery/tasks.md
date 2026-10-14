# Tasks

## 1. Visitor admission and public limits

- [x] 1.1 Add the next incremental PostgreSQL migration for per-app/UTC-month distinct visitor ledger and count; verify fresh and upgraded isolated database migration tests, including repeat migration and rollback-on-error behavior.
- [x] 1.2 Implement one idempotent visitor-admission operation used by static grant issuance and data-only entry; verify concurrent distinct users stop at 1,000, duplicate requests count once, owner exemption, unauthorized exclusion, existing-user access at the cap, and UTC month rollover with isolated PostgreSQL race tests.
- [x] 1.3 Replace monthly byte gating with `MAX_MONTHLY_VISITORS` and `monthlyVisitors` in server configuration, `/limits`, deployment validation, contracts, client displays where present, and quota documentation; verify server and client contract/typecheck tests and that historical `usage.bytes` remains readable.

## 2. Server-side short grants

- [x] 2.1 Add separate signing-key configuration and a compact signed grant bound to host, app, published deployment, app-session cookie digest and at-most-60-second expiry; verify tamper, wrong-host, wrong-session, expired-session, key-rotation and clock-boundary unit tests without logging tokens.
- [x] 2.2 Add an edge-only admission/renewal route that checks the authenticated edge proof, current app session, access grant, state, release and visitor cap before signing; verify owner, public, restricted, denied, deleted, suspended, expired and concurrent renewal paths in Go integration tests.
- [x] 2.3 Preserve Cloud Run's direct static fallback and request-time data API authorization while sharing visitor admission; verify a browser end-to-end flow for handoff, one-minute renewal, access revocation, data-only entry and login recovery, and document the server/edge protocol and error codes.

## 3. Private Worker delivery

- [x] 3.1 Extend the Worker to forward control, auth, mutation and data routes, validate the host-scoped grant before static access, and renew on demand through the authenticated origin; verify Worker tests for expiry, parallel assets, host/session replay, unknown hosts, origin failure and no forced browser login during valid-session renewal.
- [x] 3.2 Bind a private R2 bucket and implement versioned GET/HEAD, safe paths, content type, SPA fallback, conditional responses and an authorization-before-cache lookup; verify tests for missing assets, unlisted/old releases, anonymous R2 access, cross-app cache keys and no protected bytes after grant expiry.
- [x] 3.3 Add independent edge source throttling, fail-closed Worker behavior and observability for grants, visitor rejections, R2 errors and Worker usage; verify rate-limit/failure tests and document operational thresholds without logging cookies or credentials.

## 4. Storage compatibility and release preparation

- [x] 4.1 Validate the existing S3 adapter against an isolated private R2 bucket (`region=auto`) for upload, read, delete and health while retaining local MinIO coverage; verify anonymous denial and cleanup of random test objects, recording any unavailable live-test resource explicitly.
- [x] 4.2 Build a resumable GCS-to-R2 copy and integrity verifier for active and rollback-retained releases, plus a dual-write or publication-pause mechanism covering the full rollback window; verify digest mismatch, missing object, interrupted copy, retry, and old-revision rollback scenarios without touching production objects in tests.
- [x] 4.3 Update Cloud Run and Worker configuration, secret/key rotation, candidate checks, cutover/rollback commands and operations documentation; verify deployment-script tests, sample configuration validation and a documented release checklist that keeps production changes behind an explicit release request.

## 5. Integrated acceptance

- [x] 5.1 Run the relevant Go race, Worker, client, isolated PostgreSQL and browser suites together; verify authorized static bytes come from R2 without a per-asset Cloud Run call, dynamic data still hits Cloud Run, and the 60-second suspension bound holds in an end-to-end test.
- [x] 5.2 Validate OpenSpec deltas, review changed API/environment/storage compatibility, and record passed, failed, skipped and not-run checks plus the remaining production cutover prerequisites before archive.

## Workflow follow-up

- After implementation and acceptance, sync and archive the change under the repository workflow, then commit its code, specs and docs.
- Create production R2 resources, migrate actual objects, deploy Cloud Run and Worker, and switch visitor traffic only after a separate explicit release request and live-environment acceptance.
