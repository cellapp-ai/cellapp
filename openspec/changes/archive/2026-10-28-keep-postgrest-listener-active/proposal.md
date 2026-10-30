# Proposal

## Why

Production application database provisioning succeeds, but the co-located PostgREST process can continue serving a stale schema list after PostgreSQL sends reload notifications. The first data API probe then fails and blocks deployment of a newly created app.

## What Changes

- Keep CPU allocated to the Cloud Run service instance between requests and retain one warm instance so PostgREST can process its database notification listener.
- Preserve the existing provisioning and API probe gate. Validate the mitigation with a newly created operator-owned app before considering the incident resolved.
- Document the continuous instance cost and release behavior.

## Capabilities

No application contract changes. Existing application-data readiness requirements remain in force; this is a Cloud Run runtime configuration repair.

## Impact

Cloud Run service settings and operations guidance. Migration and cleanup jobs remain unchanged. The existing diagnostic proposal remains separate.
