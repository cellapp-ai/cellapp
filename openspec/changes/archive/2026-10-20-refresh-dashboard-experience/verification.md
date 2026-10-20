# Verification

Date: 2026-10-20. Worktree: `codex/refresh-dashboard-experience`.

| Result | Check | Evidence and limits |
| --- | --- | --- |
| Passed | `npm test --workspace @cellapp/web` and `npm run lint:web` | Production Vite build, TypeScript check, Chrome UI fixtures and axe checks. Covered 320, 390, 768, 800/801 and 1440 px, long English and Chinese names, 200% root text zoom, reduced motion, keyboard focus, mobile menu, application status, table structure selection and failure recovery. Fixtures are not live authentication. |
| Passed | Manual screenshot review | Reviewed application list and detail at 1440 and 390 px and profile at 1440 and 390 px. Main content remains beside the desktop navigation; mobile content stacks without page-wide overflow. Focus visibility and the scrollable table region were checked alongside the keyboard and axe browser tests. Screenshots live under `test-results/web/` and are generated artifacts. |
| Passed | `go vet ./apps/server/...` | Static Go check for the server change. |
| Passed | `go test -race -p 1 -count=1 ./apps/server/...` | Disposable local PostgreSQL control and application-data databases plus PostgREST, including isolated schema and data authorization tests. |
| Passed | `TestWebDataStructure` under `-race` | Verified empty, applied, pending, error, ready, signed-out, cross-owner, visitor-host, and unavailable cases; checked response excludes destructive flag and internal metadata. |
| Passed | `RUN_BROWSER_TESTS=1 go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting` | Local HTTPS Chrome with real disposable PostgreSQL, PostgREST, and private MinIO. Existing login, grant/request, key rotation, visitor CRUD/isolation, deletion, and control flows passed against paired Go/Web artifacts. |
| Passed | `openspec validate refresh-dashboard-experience --strict`, `git diff --check` | Change artifacts and patch formatting valid. |
| Failed then passed | Full server race suite | First run lacked the pinned `skills` checkout and failed to read `skills/contracts/http-v2.json`. Initialized the submodule at the main repository's fixed commit and reran successfully. |
| Failed then passed | HTTPS browser acceptance | Disposable PostgREST had a stale notification/schema cache and reported app data not ready. The focused PostgREST check reproduced it; restarting only that disposable container made the focused and full HTTPS checks pass. No application code was changed for this test service issue. |
| Not executed | Public candidate or production visual and service acceptance | This apply run did not deploy or route traffic. Local HTTPS results do not establish behavior on the public domain. |

No CLI or skills source was changed. The `skills` submodule remains at the main repository's pinned commit. No database migration or table-row browser UI was added.
