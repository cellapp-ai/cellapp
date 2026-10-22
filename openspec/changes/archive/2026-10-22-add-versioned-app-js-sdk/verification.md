# Verification record (2026-10-22)

## Passed locally

- `openspec validate add-versioned-app-js-sdk --strict`.
- `npm run check:app-api-versions`, including a negative run against an empty changelog that failed as intended.
- Main workspace: `npm ci`, `npm run typecheck`, `npm test`, `npm run build`.
- Go: `go vet ./apps/server/...`, `go test ./apps/server/... -count=1`. Version registry unit tests passed.
- Isolated local PostgreSQL/PostgREST: `go test -race -count=1 ./apps/server/...` with dedicated test DSNs passed, including pinned identity, admission, data CRUD, query fields, revocation and control catalog. One initial PostgREST test attempt ran during service startup and failed its readiness probe; a warm rerun and the full suite passed.
- HTTPS browser acceptance: `RUN_BROWSER_TESTS=1 ... go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting` passed against isolated PostgreSQL, PostgREST and a private local MinIO bucket. The notes demo used `@cellapp/sdk`; browser-observed identity and table requests all carried `v=1.1`. Login, visitor approval/revocation, CRUD, cross-app isolation and unversioned anonymous form access passed.
- After adding explicit per-contract identity/table adapters, the targeted real PostgREST tests and browser acceptance passed again. The isolated test containers and volumes were removed after verification.
- After moving the public table route to `/_cellapp/db/{table}`, the skills workspace typecheck, 26 tests, build, Skill/boundary/package checks and notes demo build passed. The SDK unit test confirmed no `/rest/v1` table request reaches the network. The Go suite passed with isolated PostgreSQL/PostgREST under `-race`, and HTTPS browser acceptance passed with both direct origin and edge Worker flows. The browser confirmed pinned SDK requests use the new route, while an old `/rest/v1` request receives 404 even with an HTML accept header. The Worker unit suite (19 tests) passed.
- Website lint, build and static-preview browser checks passed after the public guide and changelog route update. OpenSpec strict validation and registry/changelog check passed.
- The console browser checks passed after updating their guard against app-table fetches to the new path. An app-host route audit found no other active public API outside `/_cellapp`; the control-domain authentication, internal edge transport, and unsupported Supabase probes were kept in their existing namespaces. The control API routing delta records this boundary for main-spec sync.
- Skills workspace: `npm ci`, `npm run typecheck`, `npm test` (26 tests), `npm run build`, `npm run check:skill`, `npm run check:boundary`, `npm run check:package`, and `npm pack --workspace @cellapp/sdk --dry-run --json`. The notes demo installed and built successfully.
- Website: frozen pnpm install, lint, production build, and static preview browser test with local Google Chrome. The preview test fetched the English and Chinese version guides and the changelog and returned 200 for each. Existing lint warnings remained.
- `git diff --check` in both repositories.

## Published and checked (2026-10-22)

- The public version guide and changelog were uploaded to Cloudflare Pages. Both returned 200 on the deployment URL, `cellapp-website.pages.dev`, and `cellapp.ai` during release acceptance.
- Cloud Run deployed the final `deploy/main` commit `457d9a3` with image digest `sha256:047068a4525bffdc189af5a1d87b1c4078d972d212ca4dae8dbdf4336ee83b18`. Migration execution `cellapp-migrate-vt2wt` succeeded. Revision `cellapp-00027-rwc` became ready and received 100% traffic; the deployment script checked the control login page and unknown application host.
- Read-only public checks returned 200 for the control login page and an existing visitor app, 401 for unauthenticated versioned `/_cellapp/me` and `/_cellapp/db/notes`, 400 `unsupported_api_version` for `v=1.2`, and 404 for the former `/rest/v1/notes` path. An authenticated CLI read returned stable/latest `1.1` and date alias `261022`, and listed existing apps.
- Published skills `main` at `df91430`, `@cellapp/sdk@0.1.0`, and `@cellapp/cli@0.4.0` in that order after service acceptance. Both npm version metadata and tarballs became publicly available. An isolated install of the CLI package passed `cellapp --help` and packaged Skill checks; the notes example installed the published SDK and built in an isolated directory. The SDK's jsDelivr `+esm` URL returned 200.
- Pushed public main at `a88fb39` and private `deploy/main` at `457d9a3`; both remote heads were verified.

## Not yet verified

- A real static rollback restoring an older HTML marker, and a stable/default switch between two distinct supported API contracts. Only the baseline `1.1 = 261022` contract exists today, so the two-contract scenario cannot yet run.
- An authenticated production visitor's table CRUD and access revocation across versions. Local isolated browser tests passed these paths, while the production checks above were read-only and did not write to an existing app's data.

The remaining three OpenSpec tasks stay unchecked. The user explicitly requested archive and publication on 2026-10-22; this archive preserves the outstanding acceptance work. Do not describe multi-version switching and rollback as production verified until those scenarios pass.
