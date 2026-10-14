# Design

## Context

See proposal.md. The control transaction commits before a separate data database and PostgREST are reconciled. The existing request fingerprint includes the initial key, so a client must reuse both the request ID and key after an uncertain response. Project metadata cannot contain secrets.

## Goals / Non-Goals

**Goals:** Preserve first-deploy identity across failures, provide owner-only recovery, and expose actionable diagnostics without leaking SQL or credentials.

**Non-Goals:** Reassign tombstoned slugs, change data isolation, or diagnose an unavailable production database from local tests.

## Decisions

- Save non-secret project metadata before POST /apps. Save the generated key in a mode-0600 file beside the CLI's origin-specific credential. Reuse both on retry and remove the key file after confirmed publication. This retains the existing server idempotency fingerprint without putting secrets in the project.
- On provisioning failure, return a 503 with the already committed app ID in details. The CLI associates it immediately. Keep failure detail in server logs; expose only a stable error code through owner data status.
- Add an owner-authorized reconcile operation that uses the existing idempotent appdata.Reconcile. CLI deploy invokes it for an unready linked app; the data command also exposes it directly.
- Allow link to create a project association only after listing owner-visible apps, requiring an explicit output directory for a fresh project. Never silently replace a different existing app association.
- Preserve executable mode on the Compose PostgreSQL initialization hook. PostgreSQL's entrypoint executes `.sh` files directly when they have this mode; the existing non-executable mount exited with `bad interpreter: Permission denied` on a fresh volume.

## Risks / Trade-offs

- Private pending key survives until publication and may linger if the user abandons the project. Restrict permissions to 0600 and keep it out of project files; document cleanup with normal credential management.
- A persistent database or PostgREST fault still prevents publication. Return the same unready error and use the stable code plus server logs to diagnose configuration or service health.

## Migration Plan

Deploy server API first, then CLI and Skill. Existing applications and project associations remain valid. Older orphaned apps can be linked manually. Rollback preserves app records and private key files; the old CLI may still retry with the same saved project request metadata.
