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

## Not yet verified

- A candidate in the target environment with the public guide/changelog, compatible server and edge routing, then CLI/Skill/SDK release, followed by a release switch between two distinct supported API contracts and real static rollback. Only the baseline `1.1 = 261022` contract exists today, so changing stable to a different supported contract cannot yet be exercised end to end. No production release or package publication was performed.

The remaining OpenSpec tasks stay unchecked. The user explicitly requested archive and publication on 2026-10-22; this archive records the implementation and local checks, while the two-contract and real rollout scenarios remain open. Do not describe the version as production supported until those scenarios pass.
