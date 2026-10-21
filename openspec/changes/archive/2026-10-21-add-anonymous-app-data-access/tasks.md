# Tasks

## 1. Operator and owner settings

- [x] 1.1 Add default-false `ALLOW_ANONYMOUS_APP_DATA_WRITES` parsing and validation in Go, plus local/production environment and operations guidance; verify absent/true/false/invalid configuration cases in config tests and check the documented rollout command.
- [x] 1.2 Add the next numbered, checksummed control migration for `anonymous_pages`, `anonymous_read` and `anonymous_write`, false defaults and valid-state invariants; verify upgrade, repeat migration and restricted-app transitions against isolated PostgreSQL.
- [x] 1.3 Add owner-scoped browser and Bearer API reads/writes for the three settings, atomic visibility interaction and effective operator status; verify owner, other-owner, missing-session, wrong-Origin, invalid-combination and uncertain-retry cases in hosting tests, and update HTTP contracts.

## 2. Anonymous data isolation

- [x] 2.1 Extend manifest validation, planning and schema application with default-deny `anonymousSelect`/`anonymousInsert` modes and separate role-scoped RLS policies; verify old manifests remain valid and signed-in policies cannot be used by the anonymous role in isolated PostgreSQL tests.
- [x] 2.2 Provision and reconcile a separate per-app anonymous API role and private PostgREST grants without exposing a global anonymous database role; verify idempotent provisioning, cross-app/control-table denial, table-grant denial and cleanup with real PostgreSQL/PostgREST.
- [x] 2.3 Route anonymous GET/HEAD and POST insert through the data gateway only when operator, app and table gates allow; force minimal responses for insert-only tables, keep PATCH/DELETE signed-in, and reject credential fallback; verify successful insert/read, no-login rejection, forged identity/profile, invalid Origin, revocation and no row disclosure in hosting tests.
- [x] 2.4 Add dedicated anonymous write/read rate and concurrency limits and stable errors, while keeping body/response/time bounds; verify bursts, oversized requests and backend failure paths, and document the supported anonymous CRUD and error behavior.

## 3. Anonymous pages and edge delivery

- [x] 3.1 Admit no-session visitors to active, published public apps with `anonymous_pages=true` in the direct Go gateway, preserving protected paths and signed-in error behavior; verify page, asset, HEAD, SPA, restricted/disabled, invalid-session and suspended/deleted cases in gateway tests.
- [x] 3.2 Extend signed edge grants and Worker verification with an explicit anonymous mode bound to exact host, app and deployment for at most 60 seconds; keep object storage private and `/rest/v1` at the Go origin; verify renewal, expiry, host/version replay, off-switch and origin-failure cases in edge tests.
- [x] 3.3 Apply anonymous request limits without adding fabricated user IDs to monthly visitor tables; verify parallel anonymous traffic, 429 responses and unchanged signed-in monthly counting in isolated admission tests, and document the distinct anonymous accounting model.

## 4. Owner and Agent experience

- [x] 4.1 Add accessible dashboard controls and effective-state feedback for anonymous pages/read/write, with confirmation and uncertain-result refresh; verify owner flows, restricted-app state, operator-off display and narrow-screen keyboard operation in relevant Web tests and browser acceptance.
- [x] 4.2 Update the skills submodule CLI/shared validation, manifest types, commands, HTTP contracts and bilingual deployment Skill examples for anonymous settings and table policies; verify client unit tests, typecheck and build, including old-command compatibility and cross-owner errors.
- [x] 4.3 Update operations and app-data documentation for configuration, safe rollout/rollback and anonymous form verification; check links, names, commands and examples against the implemented API and public contracts.

## 5. Cross-service acceptance

- [x] 5.1 Run isolated PostgreSQL and real PostgREST integration/race checks for successful anonymous insert/read, insert-only response, private tables, cross-app access, gate-off behavior and signed-in regression; record pass/fail/skip distinctly.
- [x] 5.2 Run browser and production-equivalent edge acceptance with private storage for anonymous deep links and assets, form submit, origin rejection, rate limits and gate revocation; confirm old apps still require login and document any environment-dependent checks not run.

## Workflow follow-up

- After implementation and required acceptance, sync the delta specs, archive this change and follow the repository's commit, submodule and push rules.
