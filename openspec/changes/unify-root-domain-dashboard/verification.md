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

## Release preflight — 2026-10-15

- The reviewed implementation is on local main at `e733b6f` and skills main at `5031002`; the release branch merge is `84b3b74`. The release worktree matches main and its skills checkout matches the gitlink. The local release configuration validates with `control_origin=https://cellapp.ai`.
- The public npm registry lists `@cellapp/cli` through `0.2.1`; the checked package version is `0.3.0`, with both CLI output and `skills/cellapp-deploy/SKILL.md` in the package. The client typecheck, 21 tests, build, Skill, boundary, and package checks passed again. Sixteen Worker tests, 22 release-script tests, and strict OpenSpec validation passed again.
- The current Cloud Run ready revision and 100% traffic target are `cellapp-00016-c7k`. Cloudflare Worker `cellapp-edge` serves version `8617764f` and currently has only `console.cellapp.ai/*` and `*.ce11.app/*` routes. A request to `https://cellapp.ai/dashboard/login` returns the public website title and lacks dashboard assets; HTTP 200 here is a Pages fallback, not a working dashboard.
- The checked Google Cloud project lists only the production `cellapp` service and production/cloud-build buckets matching this application. No isolated candidate service or test bucket was found. The combined candidate, real OAuth callback, Pages/Worker path precedence, and migrated-project acceptance remain unrun. No production deployment, traffic switch, Cloudflare route change, Git push, Skill update, or npm publish occurred in this preflight. Publication is stopped before the service stage pending a controlled candidate environment and its acceptance.

## Direct production cutover — 2026-10-15 (in progress)

The user explicitly chose to proceed directly after the missing controlled candidate was reported. This is a production acceptance record, not a substitute for the unrun isolated candidate scenarios.

- GitHub's Cellapp OAuth application now accepts the exact `https://cellapp.ai/auth/callback` URI; the old callback remains registered for rollback. The updated root-host OAuth login reached the signed-in Dashboard, and an existing application detail deep link loaded its release and table status.
- Cloudflare Worker `cellapp-edge` version `b443a8b6-5fcd-4da8-95d2-cb1e23788933` serves 100% traffic. Nine routes cover the listed root control paths and `*.ce11.app/*`; the old `console.cellapp.ai/*` route is absent after a dashboard reload. Existing R2, rate limiting, grant public key, and origin secret bindings were preserved.
- Cloud Run built `sha256:c6513c82469c56d8a91504933a8d2aedab6352d9f71e0fd6e6711bb91d990032`, completed migration `cellapp-migrate-k62qp`, and serves revision `cellapp-00017-p9s` at 100%. The prior revision is `cellapp-00016-c7k`. The release script exited unsuccessfully after traffic switch because the R2 Worker strips `X-Request-ID` from a denied unknown-visitor response; the root dashboard returned 200 with a request ID and dashboard assets, while the unknown visitor domain returned 404 without that header. The check was corrected and passed against the real domains; 23 release-script tests passed. No service rollback was needed.
- Cloudflare Pages production deployment `861ad39c-85cc-4652-a3a3-cc55ff0022aa` serves the new website. The live website contains the Dashboard link and on-demand Agent guide; the dashboard still reaches Cloud Run. The existing `lean-trainer-wyx.ce11.app` application loaded and reported data ready in a signed-in browser; an anonymous request returned 401 with `private, no-store`. Anonymous browser and Bearer API reads returned 401; direct `run.app` access and a forged original-host header returned 404.
- A production cross-origin POST probe was rejected by automatic approval review because it could write production data; it was not performed. The live device authorization request and a temporary local client credential test are awaiting the user's browser confirmation. npm login reached email OTP and is awaiting the user; `npm whoami` returned 401 before that flow. GitHub Skill, npm package, and repository branch pushes have not happened. The change is not archived or fully released.
