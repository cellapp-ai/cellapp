# Implementation verification

All database tests used separate local control and app-data PostgreSQL databases. The real API test used a standalone PostgREST 16.3 connected only to the isolated app-data database. Browser acceptance used real Chrome, HTTPS, PostgreSQL, PostgREST and a private local MinIO bucket.

| Check | Result |
| --- | --- |
| `openspec validate add-anonymous-app-data-access --strict` | Passed |
| `go test -race -count=1 ./apps/server/...` with isolated PostgreSQL and PostgREST DSNs | Passed, including migration, per-app role isolation, anonymous insert/read/denial, rate and concurrency limits, signed-in regressions |
| `RUN_BROWSER_TESTS=1 RUN_EDGE_BROWSER_TESTS=1 go test -race -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting` with isolated services | Passed, including anonymous form submission in a no-session browser, deep links, edge grant renewal and revocation |
| `node --test infra/cloudflare-worker.test.mjs` | Passed: 19 tests |
| Root `npm run typecheck`, `npm run build`, `go vet ./apps/server/...` | Passed |
| Web lint and browser fixture test | Passed: 9 browser groups, including anonymous owner controls and narrow screen |
| Skills submodule typecheck, unit tests, build, Skill, boundary and package checks | Passed |
| Real Cloudflare R2 and production Worker deployment | Not run; local edge browser used the production Worker code with a private R2 binding test double. No production service or public route was modified. |

The isolated browser fixture enables the operator switch only within the test server. The production operator switch remains default false. Anonymous traffic is not counted as signed-in monthly visitors; rate-limit counters are separate and keyed by application and source IP.
