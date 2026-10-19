# Local verification record

Date: 2026-10-19. Worktree: `codex/mcp-agent-connection`. Target MCP protocol: `2025-06-18`; Go SDK: `github.com/modelcontextprotocol/go-sdk v1.3.1`.

The database checks used a disposable PostgreSQL container on loopback port 55432 with separate control and application-data databases. The application-data checks used a disposable PostgREST instance on loopback port 55433. Browser checks used disposable private MinIO on loopback port 9000. These are local test services, not a public candidate deployment.

| Result | Check | Evidence and limits |
| --- | --- | --- |
| Passed | `go test -race -p 1 -count=1 ./apps/server/...` | Repeated after the final local implementation changes with isolated PostgreSQL, application-data PostgreSQL and PostgREST configured. Covers migration, MCP OAuth, SDK tools, table plan/apply/verify, shared REST behavior and concurrency. |
| Passed | `RUN_BROWSER_TESTS=1 ... go test -race -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting` | Repeated with all isolated test DSNs, PostgREST and MinIO credentials configured. Real local Chrome consent, authorization-code exchange, connection listing and revocation. |
| Passed | `node --test infra/cloudflare-worker.test.mjs` | 18 Worker route tests, including MCP and OAuth control paths. |
| Passed | `npm test`, `npm run build`, `go vet ./apps/server/...`, `git diff --check`, strict OpenSpec validation | Repeated after the final local implementation changes. Existing web, CLI, Go and build regressions. `npm test` includes the CLI contract and Web typecheck suites. |
| Passed | Focused `go test -race -count=1` MCP tool, upload and publish checks | Against isolated PostgreSQL after the full suite: cross-instance resume, same-offset concurrency keeps object bytes aligned with committed digest, single-winner concurrent publish, structured error and no access-key exposure. |
| Failed then passed | Isolated PostgREST checks during initial full-suite attempts and a later focused rerun | The disposable PostgREST process stopped receiving schema reload notifications; restarting that test container restored it. The target checks and full race suite passed without changing server probe behavior. |
| Setup failure then passed | Final Chrome acceptance rerun | The first command omitted required isolated application-data and PostgREST environment variables; it stopped before running the browser scenario. The rerun with those variables passed. |
| Skipped | Public S3 integration test requiring `RUN_S3_TESTS=1` | The focused MCP tests used private in-memory storage; the browser suite used disposable MinIO. This does not establish production object-store behavior. |
| Not executed | Real Codex and 豆包 connections to an HTTPS candidate Cellapp service | No candidate service address or test accounts were available. Tool import, user-delegated OAuth, interrupted upload, data table creation and revoke must be checked with both client products before marking tasks 5.2 and 6.2 complete. |
| Completed | Cloud Run service publication | User explicitly chose a direct formal-traffic switch. The service release result is recorded below. |
| Not executed | Cloudflare Worker route and npm publication | The public MCP paths still reach Pages rather than the service. Skills and npm package content did not change. |

The change remains active pending real client checks and public MCP routing.

## Release direction

On 2026-10-19 the user requested a Cloud Run release and explicitly chose a direct switch to formal traffic. Tasks 5.2 and 6.2 remain incomplete because a separate HTTPS candidate and test accounts are unavailable. This choice does not turn those checks into passed acceptance; the release must record the actual revision, route results and any remaining Cloudflare work separately.

## Cloud Run service release — 2026-10-19

- Source: `deploy/main` commit `4d9692e93b7b6c87b3971c15ba13376759b43d01`; fixed skills commit `c4d761025a32068854a937e08d40f1812c12c900`. No skills or npm package content changed; npm remains `@cellapp/cli@0.3.0`.
- Previous serving revision: `cellapp-00018-n8f`. Migration job execution: `cellapp-migrate-9fft7`, reported successful by the release script. New serving revision: `cellapp-00019-56g`, verified at 100% traffic. Image digest: `sha256:afa6b800920d383f7ff72621c1fcaea3b24a9a98065b1a28c49598a1ea68b053`.
- Release script passed candidate revision health, readiness, control-domain login and invalid application-host checks. Independent read-only checks found `/dashboard/login` HTTP 200, unauthenticated `/api/control/v1/apps` HTTP 401, and no `ERROR` severity log entries for the new revision in the first 15 minutes.
- Public `GET /mcp`, `/.well-known/oauth-protected-resource/mcp`, and `/.well-known/oauth-authorization-server` still returned Pages HTML rather than MCP/OAuth responses. The Cloudflare Worker route changes were not published as part of this Cloud Run release. Therefore public Agent connection and tasks 5.2/6.2 remain unverified and unavailable at the documented root-domain URL.
- Rollback target for a service regression: `cellapp-00018-n8f`. The 009 migration and published objects must remain in place if service traffic is rolled back.
