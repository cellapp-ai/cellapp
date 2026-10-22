# Design

## Context

See [proposal.md](./proposal.md) for motivation and the two delta specs for acceptance criteria. `infra/compose.yaml` currently starts development PostgreSQL, PostgREST, Caddy, and MinIO with fixed development credentials; Go runs on the host. The production `Dockerfile` already bundles the Go service, matching Web console, PostgREST, and an internal HTTP Caddy entrypoint. Production configuration rejects `AUTH_MODE=dev`, overlapping control/application registrable domains, and an HTTP S3 endpoint. The Cloud Run release path and local development path must stay usable.

## Goals / Non-Goals

**Goals:** A clean checkout plus operator-provided domains, trusted public certificate/key files, and GitHub OAuth credentials can start and verify the full platform on one Linux host. MinIO and both databases are local durable services with independently generated production secrets. README readers can reach hosted usage, self-hosting, and development instructions quickly.

**Non-Goals:** Automatic public wildcard certificate issuance, multi-host high availability, a new login provider, migration of an existing Cloud Run installation, the marketing website, paid sales, and Cloudflare edge/R2 acceleration. The guide will state these boundaries rather than imply they are part of the Compose deployment.

## Decisions

### 1. Keep production Compose separate from development Compose

Add `infra/compose.selfhost.yaml` plus self-host-only initialization/config assets. The current `infra/compose.yaml` remains a localhost development entry with its existing fixed test credentials. This avoids turning a familiar development command into a public service by accident. Compose runs one PostgreSQL instance containing distinct control and application databases, a private MinIO service, the existing production application image, and a separate public TLS ingress. The application image retains its bundled PostgREST and internal Caddy; only ingress binds host ports 80/443. PostgreSQL, MinIO, PostgREST, and the app image's HTTP port stay on private Compose networks.

Alternative considered: extend the development Compose with profiles and production overrides. That makes it easier to accidentally inherit loopback certificates, host ports, or `development` passwords. A separate file is easier to audit.

### 2. Use explicit public certificates for the first release

The operator supplies trusted certificate/key files covering the control hostname and `*.APPS_DOMAIN`, with DNS already directed to the host. The ingress configuration selects the correct certificate and forwards the original Host unchanged to the application image. Preflight checks names, certificate/key readability and coverage, and the required two distinct registrable domains before public startup. GitHub OAuth uses the control origin and its exact `/auth/callback` URL.

Alternative considered: built-in ACME DNS-01. Wildcard issuance requires provider credentials and a provider-specific ingress build, which adds a second public secret and update path. The first release documents certificate acquisition separately; automation can be added without changing Cellapp's public URL contract. A valid wildcard certificate is therefore an explicit prerequisite, not something Compose can infer from a domain alone.

### 3. Keep MinIO private and satisfy the production HTTPS rule

MinIO is the default storage provider, with a persistent object volume and an idempotent private Bucket initializer. Setup generates a private internal CA and a MinIO server certificate for the Compose service hostname, stores them outside Git, and mounts the CA into the application container's trust store. The service uses `S3_ENDPOINT=https://minio:9000` with generated credentials. MinIO's API and console have no host port mapping. An external S3 option requires an explicit HTTPS endpoint, Bucket, region and credentials; it does not alter the bundled default or silently migrate existing objects.

Alternative considered: allowing plain HTTP to MinIO over a Docker network. This conflicts with the existing production S3 security boundary, so the plan keeps that check and adds internal TLS.

### 4. Make bootstrap repeatable and fail closed

A setup command validates operator inputs, creates a protected local config directory with mode 0700, and generates independent high-entropy values only when missing. It writes files with mode 0600 and never prints secret values. The production database initializer creates the app-data database and roles without development passwords; separate control and app-data DSNs and a shared Go/PostgREST JWT secret are supplied from the protected configuration. Setup brings up dependencies, initializes the Bucket, applies control migrations, and checks readiness before starting public ingress. Repeated runs reuse the same secrets, databases and Bucket; a failed step stops with a specific stage and a retry command. The current image may need a small entrypoint override or separate one-shot migration service, but application API behavior remains unchanged.

Alternative considered: relying only on Compose `depends_on`. Container order does not prove Bucket privacy, successful migrations, or application readiness, so setup must make those explicit gates.

### 5. Separate three reader journeys

Root `README.md` becomes a short English product entry, with a matching `README.zh-CN.md` and language links at the top of both files. Each introduces the current capabilities and boundaries, an Agent-first Quick start with a single request to install the deployment Skill and publish, an optional manual CLI path, self-hosting and development links, and the license. Agents fetch the CLI on demand; users approve browser authorization. The self-hosting and development guides remain in Chinese; the English README labels these links. `docs/development.md` receives current checkout, local dependencies, build/test, submodule and Web/website contributor commands. `docs/self-hosting.md` contains the ordered production path and links into existing operations, data, and storage documents for deeper recovery. Existing Cloud Run documentation stays under `docs/operations.md`. Verify all moved links and replace stale `console.cellapp.ai`, old npm package, and SSH submodule claims with current facts.

## Risks / Trade-offs

- [Certificate prerequisite is more than “configure a domain”] → Say plainly in README and self-hosting prerequisites that wildcard DNS and trusted certificates are needed; preflight rejects missing or mismatched files.
- [A single host can lose all durable state] → Use named persistent volumes, document off-host backups of both databases, MinIO objects and secrets, and include a restore drill before calling the guide production-ready.
- [Generated internal CA is lost or rotated] → Persist CA material, do not regenerate on repeated setup, and test service-to-MinIO TLS after restart.
- [Secrets in a host environment file are readable to Docker administrators] → Keep them out of Git and image layers, restrict file permissions, avoid diagnostic commands that print resolved secret values, and state the host-admin trust boundary.
- [Existing image entrypoint assumes Cloud Run conventions] → Exercise the same image under Compose and make only focused entrypoint/config changes; rerun Cloud Run smoke checks if shared behavior changes.
- [No public host available in CI] → Use an isolated Compose integration environment with controlled DNS/cert fixtures for automated gates, then report public-domain OAuth, certificate and wildcard routing checks separately until an operator completes them.

## Migration Plan

1. Add and validate production assets without changing the development Compose or Cloud Run configuration.
2. Verify first boot, restart, repeated setup, migration failure, private Bucket and host routing against isolated resources; exercise a published app and data table with a browser/CLI.
3. Write the self-hosting and development guides from the verified commands, then rewrite the README and check all links and claims.
4. For a new host, the operator backs up config/volumes, points DNS, supplies public certificates and OAuth credentials, runs setup and smoke checks, then admits real users. Updates back up durable state and reuse secrets; rollback restores a compatible application image while retaining volumes, with migration compatibility checked first.
