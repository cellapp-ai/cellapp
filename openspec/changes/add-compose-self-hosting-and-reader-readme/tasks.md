# Tasks

## 1. Production Compose and configuration

- [x] 1.1 Add an isolated production Compose file for the existing application image, PostgreSQL, MinIO and public TLS ingress; verify `docker compose config` resolves the intended services and exposes only ingress ports.
- [x] 1.2 Add a repeatable self-host setup/preflight command that validates the two domain boundaries, certificate coverage, GitHub OAuth inputs and non-development mode; verify invalid and missing inputs fail without printing secrets.
- [x] 1.3 Generate and reuse independent protected platform, database, JWT and MinIO secrets outside Git; verify a second setup preserves byte-for-byte values and development defaults are rejected.
- [x] 1.4 Configure ingress with operator-supplied control and wildcard certificates and original-Host forwarding; verify valid control/app hosts route correctly and unknown hosts are rejected using isolated TLS fixtures.

## 2. Durable dependencies and startup gates

- [ ] 2.1 Configure private MinIO with persistent data, generated credentials and verified internal HTTPS; verify signed object round trip, anonymous denial, no public storage ports, and trust after container restart.
- [x] 2.2 Add production-safe application database/role initialization without fixed development passwords, and run control migrations before public startup; verify fresh bootstrap, repeat bootstrap and distinct database/role boundaries.
- [x] 2.3 Gate public readiness on database, MinIO Bucket privacy, migrations and application health; verify a failed dependency or migration reports its stage and retains existing volumes for retry.
- [x] 2.4 Exercise the existing production image under Compose, adjusting its entrypoint only where required; verify the paired Web console and PostgREST start, and rerun the Cloud Run entrypoint smoke check if shared behavior changes.

## 3. Reader and operator documentation

- [x] 3.1 Write `docs/self-hosting.md` from the verified Compose commands, covering domain/DNS/certificates, OAuth, first start, health, publish checks, updates, backup, restore and failure recovery; verify every local command against the new assets and label public-domain checks not run locally.
- [x] 3.2 Move contributor setup and checks from the root README into `docs/development.md`, including recursive submodules, local Compose, server/Web/website builds and relevant standards links; verify commands, paths and links against current repository files.
- [x] 3.3 Rewrite root `README.md` as a concise English project/use entry with current capabilities, boundaries, an Agent-first one-request Skill installation and publication path, a secondary manual CLI path, self-hosting/development links and licenses; add an equivalent `README.zh-CN.md` with top language links in both files. Verify links and current names/domains/packages, with no stale installation artifact or development runbook left inline.

## 4. End-to-end acceptance

- [x] 4.1 In an isolated Compose environment, verify first boot, GitHub-mode authorization with a test identity provider, static publication, app-host access, application data operation and unknown-host rejection; record actual pass/fail/skip results.
- [x] 4.2 Verify restart, repeated setup, migration/dependency failure recovery, private Bucket and backup/restore of both databases, MinIO objects and secrets; document any public DNS, trusted certificate or real GitHub OAuth checks that remain unexecuted.

## Workflow follow-up

- After implementation and required acceptance, sync the delta specs, archive this change, review the resulting diff, and commit only its files as required by the repository workflow.
