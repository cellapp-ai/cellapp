# Tasks

## 1. Runtime API contracts and public version record

- [x] 1.1 Add a Go registry with one-to-one `major.minor = yymmdd` aliases per contract, latest and configured stable pointers, strict `v` validation, and `unsupported_api_version`; verify both aliases resolve identically and server tests reject duplicate aliases, invalid dates, missing/duplicate/malformed/unknown/retired `v`, and invalid stable configuration.
- [x] 1.2 Route `/_cellapp/me` through the selected contract while sharing current session, admission, and app-user logic; verify browser/API tests for admitted, signed-out, pending, denied, revoked, and cross-app requests under pinned and unversioned calls.
- [x] 1.3 Route `/_cellapp/db/{table}`, select and strip `v` before PostgREST forwarding while retaining present query fields and permission checks; reject old `/rest/v1/{table}` without static fallback. Verify isolated database tests for CRUD, eq/order/range, anonymous gates, revocation, forged headers, wrong Origin, and unsupported versions.
- [x] 1.4 Publish the default stable/supported-version control read with both aliases for build clients and update the public HTTP contract; verify authentication/response tests and a server startup failure when the configured stable ID is unsupported.
- [x] 1.5 Update the public Agent-readable API version guide and changelog for `/_cellapp/db/{table}`, keep both runbook and internal doc links, and verify public static-preview links, the `1.1 = 261022` baseline, one-to-one alias rule, errors, migration and support-status content.
- [x] 1.6 Add a repository check that compares registry alias pairs with changelog entries and validates Agent guide links; run the check and demonstrate it fails when either alias lacks a changelog entry.

## 2. Build-time pinning and rollback behavior

- [x] 2.1 Add CLI project/flag support for either explicit API alias, resolve it to canonical `major.minor`, and resolve the platform stable contract before build when absent, without persisting the resolved default as an explicit preference; verify CLI tests for both aliases, override, changing default, unsupported version, discovery failure, and retry.
- [x] 2.2 Insert or update one canonical `major.minor` meta marker in every output HTML entrypoint before manifest hashing and upload, including prebuilt plain HTML; verify CLI tests for SPA and multi-page outputs, idempotent retry, conflicting/malformed markers, file hashes, and the reported canonical/date alias pair.
- [ ] 2.3 Document build-time selection and prebuilt-output behavior in the CLI/Skill usage material, then verify the examples produce a pinned output and a static rollback serves an older marker without a deployment database version field.

## 3. Official browser SDK and Agent usage

- [x] 3.1 Add a public ESM browser SDK package in `skills/packages/` with its declared Supabase table-client dependency and package checks; verify clean install, typecheck, build, and package boundary checks in the skills workspace.
- [x] 3.2 Implement SDK initialization from the HTML version marker and current-user access/error handling with same-origin credentials and `?v`; verify unit tests for successful identity, missing/invalid marker, each access error, malformed response, and no credential persistence.
- [x] 3.3 Rewrite Supabase table-builder requests inside the SDK to `/_cellapp/db/{table}` with version propagation that preserves query parameters and reports data errors; verify client contract tests for select/insert/update/delete, filter/order/range, write-after-select, and unsupported operations.
- [x] 3.4 Update the runnable notes demo and English/Chinese deployment Skill for the unified `/_cellapp` public namespace; verify the demo build, Skill checks, and Agent instructions for version diagnosis and unsupported Supabase features.

## 4. Cross-service acceptance

- [ ] 4.1 Run isolated PostgreSQL/PostgREST and browser acceptance with a pinned SDK app, an unversioned legacy page, current stable/default changes, an already open page, and restoration of older static files; record pass/fail/skipped results and confirm access revocation still takes effect for every version.
- [ ] 4.2 Verify rollout order and real candidate behavior: public guide/changelog reachable without login, compatible server and edge routing deployed before versioned SDK clients, CLI/Skill release after server acceptance, and no claimed production support until these checks pass.

## Workflow follow-up

- After implementation and required acceptance, sync the delta specs and archive the change under the project workflow.
- Commit the reviewed cross-repository changes in skills-first order, then follow Gitflow merge and push rules; Cloud Run, website, and npm publication are separate delivery steps.
