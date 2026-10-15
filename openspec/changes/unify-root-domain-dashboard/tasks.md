# Tasks

## 1. Root-host control pages and authentication

- [x] 1.1 Move `apps/web` routes, navigation, login return handling, and Vite asset base under `/dashboard`; verify direct entry, refresh, back/forward, device code retention, and built asset URLs in Web browser checks.
- [x] 1.2 Update Go management page/asset routes, safe return validation, device verification URL, and logout destinations for the dashboard namespace; verify known deep links load, unknown pages/assets/API paths return 404, and OAuth/device tests preserve state without automatic approval.
- [x] 1.3 Run isolated HTTPS browser acceptance for owner login, Secure cookies, same-origin writes, cross-origin rejection, owner/visitor separation, and session expiry on the root host; record test-double limits and update current Web/API documentation with checked paths.

## 2. Website and edge path boundary

- [x] 2.1 Add a public website management entry and revise the deployment page's user-facing copy to describe Agent-managed setup and browser approval; verify `apps/website` lint, build, static preview tests, public routes, and `/deploy/AGENTS.md` output.
- [x] 2.2 Configure Cloudflare path-specific root-host Worker routing and authenticated Cloud Run forwarding while leaving public Pages paths and `*.ce11.app` visitor handling intact; verify Worker tests for methods, query/body/cookies, unknown paths, no private caching, forged edge assertions, and absence of old API forwarding.
- [x] 2.3 Add an explicit inventory/check of Go control endpoints, Web asset paths, and Worker route patterns, and update edge/Cloud Run configuration and operations guidance; verify every private path is covered and public website paths are excluded.

## 3. Agent-led deployment and official-origin migration

- [x] 3.1 Change the published client's default origin and implement exact-match migration for official old origins in supported project associations, preserving app/pending IDs and rejecting conflicting or custom origins safely; verify client migration, idempotency, interrupted-write, and wrong-owner tests in the standalone `skills` repository.
- [x] 3.2 Keep new-origin credentials separate from old origin-keyed files and trigger one fresh device approval per local environment while reusing the resulting owner credential across owned projects; verify token isolation, revocation/expiry, two-project reuse, and account-mismatch behavior.
- [x] 3.3 Update the Skill, public Agent guide, CLI help/contracts, and packaging checks so the Agent runs a verified published client on demand without a global install or user CLI steps; verify clean-machine package execution, setup-failure reporting, client tests, build, Skill checks, and package contents.

## 4. Coordinated integration and release readiness

- [ ] 4.1 Exercise the combined Pages/Worker/Cloud Run candidate on a controlled host, including website navigation, dashboard OAuth callback, device approval, browser and Bearer APIs, a migrated existing project, and a visitor application; record actual candidate versions, results, and any unavailable external checks.
- [ ] 4.2 Update the deployment/cutover runbook with root-host DNS and OAuth configuration, release order, old API retirement, live acceptance, and coordinated rollback including client/version mismatch; verify documented paths and commands against configuration and the candidate results.
- [x] 4.3 Run strict OpenSpec validation and the affected repository checks, reconcile any failures, and record passed, failed, skipped, and unrun verification before the change is considered ready to archive.

## Workflow follow-up

- Sync main specs and archive only after the required implementation and acceptance work is complete.
- Follow repository Gitflow for review, scoped commits, merge, and default push; Cloud Run, Cloudflare Pages/Worker, and npm publication remain separate release actions.
