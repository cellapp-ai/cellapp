# Tasks

## 1. Database foundation and provisioning

- [x] 1.1 Add version tracking to the existing control migration runner before adding an incremental migration; verify fresh install, upgrade from `001_initial.sql`, duplicate run, and concurrent migrators against isolated PostgreSQL.
- [x] 1.2 Add control records for visibility, app-user mappings, grants, requests, app-data provisioning and cleanup state; verify uniqueness, ownership, transactions and compatibility with existing app rows in isolated database tests.
- [x] 1.3 Configure a distinct application-data DSN and bootstrap its migration ledger and least-privilege provisioner/API/DDL roles; verify same-database configuration is rejected and control/runtime roles cannot query each other's database objects.
- [x] 1.4 Implement idempotent per-app schema/role provisioning and readiness reconciliation; verify interrupted creation, retry, concurrent creation and cross-schema denial using real PostgreSQL roles.
- [x] 1.5 Document local/production app-data deployment, credentials, readiness and backup boundaries beside existing operations guidance; verify documented setup against an isolated local environment.

## 2. Platform identity and application access

- [x] 2.1 Add a single-use app login handoff and host-only app session tied to the platform session; verify safe deep-link return, replay/expiry, wrong-app tickets, logout and cross-domain cookie isolation in HTTPS browser tests.
- [x] 2.2 Persist stable random per-app user IDs and expose a minimal same-origin current-user endpoint; verify same user/device consistency, different IDs across apps, and absence of global IDs/provider subjects in responses.
- [x] 2.3 Implement restricted-default/public visibility and user-bound grants from key redemption and owner approval; verify duplicate redemption/request decisions, public-to-restricted transition, concurrent revoke/delete and owner access in transaction tests.
- [x] 2.4 Migrate existing key hashes to signed-in redemption, reject old key-only sessions and make key rotation preserve grants; verify old-key redemption, rotation, individual revocation, page/asset denial and no plaintext key leakage in browser tests.
- [x] 2.5 Update login/access documentation and app gateway error states for unauthenticated, pending, denied and admitted users; verify returned status, redirects and accessible recovery paths in browser tests.

## 3. Managed tables and row policies

- [x] 3.1 Define and validate the versioned non-secret table/policy manifest and plan output, including supported column types, keys, indexes, destructive-change flag and `none`/`all-admitted-users`/`own-rows` modes; verify invalid identifiers, unsupported constructs and version conflicts in unit tests.
- [x] 3.2 Generate transactional app-scoped DDL, grants and RLS from the manifest through a restricted DDL role; verify default deny, owner-column `USING`/`WITH CHECK`, rollback and attempted cross-schema/function/extension escalation against real PostgreSQL.
- [x] 3.3 Add per-app version/digest ledger, locking, idempotent apply and status APIs for owners; verify identical retry, changed same-version conflict, concurrent apply and partial failure recovery in integration tests.
- [x] 3.4 Enforce query/write time, row, response, connection and storage bounds with explicit errors; verify concurrent writes and expensive requests cannot bypass limits or impair the control database.
- [x] 3.5 Document manifest schema, policy examples, data limits and safe destructive changes; validate each published example through the CLI against an isolated app schema.
- [x] 3.6 Verify user-defined table names are scoped per app, reserve unreachable routes and PostgreSQL system columns, and reject generated index-name conflicts consistently in CLI and server manifests.

## 4. Supabase-compatible data gateway

- [x] 4.1 Add an internal-only PostgREST service bound to the application-data database with dynamic schema/config and cache reload; verify new/changed tables become usable only after a successful readiness probe.
- [x] 4.2 Add app-host `/rest/v1` routing that validates platform session, application admission and target host on every request, strips client identity/profile headers, and sends only a short-lived internal app-role claim; verify forged headers, cross-app URLs, direct PostgREST access and revoked grants are denied.
- [x] 4.3 Pin and document the supported `supabase-js` version range and table CRUD/filter/order/page/write-return matrix; run contract tests against real PostgREST for allowed and denied results, error shapes and unsupported Auth/Realtime/Storage/RPC paths.
- [x] 4.4 Supply a browser-safe app URL/publishable identifier and a working static frontend example using `createClient` and current-user access; verify no database, owner or internal signing secret appears in the built assets.

## 5. Owner console and API

- [x] 5.1 Add owner-only `/api/console` endpoints for visibility, access requests, grants, revocation and key rotation with strict Origin and resource checks; verify cross-owner 404, absent/wrong Origin, concurrent decision and uncertain-write recovery behavior.
- [x] 5.2 Add console views for public/restricted state, pending requests, granted users and explicit approve/deny/revoke/rotate confirmations; verify keyboard use, narrow layouts, loading/empty/error states and one-time key result clearing in browser tests.
- [x] 5.3 Expose truthful data-provisioning and table-version status to owners without showing database credentials or other apps' metadata; verify provision failure and unauthorized access are not displayed as success or empty data.
- [x] 5.4 Update the console/API documentation for new operations and changed key semantics; verify every documented request and error against the implemented browser API.

## 6. Public CLI, contracts and Skill

- [x] 6.1 Extend the `skills` submodule CLI and shared validation with app data plan/apply/status/verify commands, versioned manifest handling and explicit separate outcomes from static deploy; verify command output, retry, owner denial and standalone submodule tests.
- [x] 6.2 Update the public HTTP/local-state contracts and client compatibility checks for data management, app access and breaking key-reset semantics; verify contract tests and old project association migration without storing secrets in project files.
- [x] 6.3 Update `cellapp-deploy` Skill to guide table/policy planning, least-privilege verification, `supabase-js` CRUD integration, current-user use and static publish; verify Skill checks and an end-to-end Agent-readable example using only supported commands.
- [x] 6.4 Update client README and installation guidance with data workflow, supported Supabase subset, no custom server, public identifier versus secret, and old-key visitor migration; verify commands and links against the published CLI.

## 7. Lifecycle and cross-system acceptance

- [x] 7.1 Extend app deletion/maintenance to deny data immediately and retry app-schema/role cleanup without affecting other apps; verify unavailable app database, recovery, repeated cleanup and non-reuse of deleted app IDs in isolated integration tests.
- [x] 7.2 Run isolated PostgreSQL race tests and real HTTPS browser acceptance for public/restricted login, access requests, key redemption/rotation, user-scoped RLS, cross-app rejection and `supabase-js` CRUD; record pass/fail/skip and real versus stubbed dependencies.
- [x] 7.3 Run repository typecheck, tests, build, Go vet, CLI/Skill checks and relevant real PostgREST/MinIO regression; record unmet environment gates and update verification/restore guidance from actual results before requesting archive review.

## 8. Application namespace and share alias

- [x] 8.1 Add an incremental control migration for owner namespaces and per-namespace app aliases; backfill existing apps and preserve old ID hosts.
- [x] 8.2 Assign and validate namespaces and aliases during app creation, resolve share links to ID hosts, and prevent duplicate or deleted-alias reuse.
- [x] 8.3 Expose alias fields in CLI contracts, project workflow, owner console, Skill and user documentation.
- [x] 8.4 Verify migration, same/different-owner conflicts, redirects, deletion tombstones, browser login and unchanged app data isolation against real services.

## Workflow follow-up

- Review the completed change and required real-service acceptance before syncing main specs or archiving it.
- Follow the repository's cross-repository commit and remote-availability order when implementation is later completed; this proposal does not authorize implementation or publish operations.
