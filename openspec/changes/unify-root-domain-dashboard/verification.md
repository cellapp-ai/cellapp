# Verification record — 2026-10-15

This record is for the local worktree `codex/unify-domain-dashboard`. It does not claim a production release or a combined remote candidate.

## Passed

- Main repository: `npm run typecheck`, `npm test` (including eight Web browser groups), `GOCACHE=/private/tmp/cellapp-go-cache npm run build`, `GOCACHE=/private/tmp/cellapp-go-cache go vet ./apps/server/...`, and `python3 -m unittest tests/deploy_cloud_run_test.py` (22 tests).
- Go server: `go test -race -count=1 ./apps/server/...` against a separate PostgreSQL and PostgREST stack passed. `RUN_BROWSER_TESTS=1 RUN_EDGE_BROWSER_TESTS=1 go test -race -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting` passed on `https://cellapp.localhost:<port>` using real local PostgreSQL, PostgREST and private MinIO. Browser acceptance covered login, device approval, Secure cookies, owner/visitor access, app data, revocation and visitor edge behavior; OAuth identity and R2 binding were controlled test doubles. Isolated test containers and volumes were removed after the run.
- Edge: `node --test infra/cloudflare-worker.test.mjs` (16 tests), including apex path inventory, request forwarding and retired-host rejection.
- Website: `corepack pnpm run lint` (exit 0 with existing warnings), full `build`, and `CHROME_PATH=/Applications/Google Chrome.app/Contents/MacOS/Google Chrome corepack pnpm run test` against its static preview. All eight public routes, `/deploy/AGENTS.md`, mobile sizes, accessibility, direct refresh and file bundle passed with no runtime errors.
- Public skills repository: `npm run typecheck`, `npm test` (21 tests), `npm run build`, `npm run check:skill`, `npm run check:boundary`, `npm run check:package`, `npm pack --dry-run --json`; packed `0.3.0` tarball ran `cellapp --help` via `npm exec` from an empty temporary directory.
- `openspec validate unify-root-domain-dashboard --strict` passed. `git diff --check` passed in main and skills repositories.

## Failed then resolved

- Initial website build could not open Parcel's LMDB cache under the sandbox. Repeated with permitted local cache access; full build passed.
- Initial website browser run could not find its pinned Playwright headless shell. Repeated with the installed Google Chrome executable; browser acceptance passed.
- Initial Go and npm package checks could not write their default caches. Repeated with writable temporary caches; checks passed. No ownership or permission changes were made to user caches.
- The client update-hint fixture still advertised `0.2.3` after the planned package version became `0.3.0`; fixture updated to `0.3.1`, and the full client suite passed.

## Skipped or unrun

- A real GitHub OAuth provider and Cloudflare R2 were not part of the isolated browser run; the identity provider and Worker R2 binding were test doubles, while PostgreSQL, PostgREST and private S3/MinIO were real local services.
- No combined real Pages custom domain / Worker route / Cloud Run candidate was deployed or verified. Route precedence, real OAuth callback, old-host API retirement, new-origin owner approval, existing-project publish and visitor access remain to be checked on a controlled host. No candidate version IDs exist yet.
- No production DNS, OAuth application, Cloud Run, Pages, Worker, GitHub Skill, or npm publication was performed. This change is not ready to archive or declare released.
