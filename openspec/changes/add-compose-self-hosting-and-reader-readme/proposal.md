# Proposal

## Why

The root README is dominated by contributor commands and mixes local development, Cloud Run operations, and end-user usage. The existing Docker Compose file starts development dependencies but does not provide a reproducible way to run the complete Cellapp platform on an operator's server. Readers need a short project entry point and a separate, truthful self-hosting path.

## What Changes

- Rewrite the root README around what Cellapp does, its supported boundaries, the hosted Agent/CLI usage path, and links to self-hosting and contributor guides. Remove stale URLs, package examples, and development transcripts from the root page.
- Move local checkout, build, test, and contributor setup into `docs/development.md`, preserving necessary commands and linking to existing detailed standards and operations material.
- Add `docs/self-hosting.md` for a complete single-host Docker Compose deployment: Cellapp service, control and application databases, PostgREST, private MinIO by default, HTTPS ingress, initialization, GitHub OAuth, verification, updates, backup, and recovery.
- Add a separate production Compose configuration and safe setup path. Operators supply a control hostname and a wildcard application domain on different registrable domains, GitHub OAuth credentials, and trusted certificate/key files for both domains. Generate and persist independent platform, database, JWT, and MinIO secrets rather than using development defaults.
- Keep existing Cloud Run and local-development flows available. An external HTTPS S3-compatible store remains an optional advanced configuration; no new application API is proposed.

## Capabilities

### New Capabilities

- `compose-self-hosting`: Reproducible, isolated, recoverable production deployment of the full Cellapp platform on one host with private MinIO by default.

### Modified Capabilities

- `repository-workflow`: Give readers a product-focused root README and a separate contributor setup guide while preserving discoverable development and deployment instructions.

## Impact

Root README; new development and self-hosting guides; production Compose, ingress, initialization, secret and validation assets; possibly the existing production image entrypoint for Compose reuse. The current dev Compose and Cloud Run release path must remain compatible. No client submodule or public protocol change is expected.
