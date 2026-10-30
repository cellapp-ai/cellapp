# Design

## Context

See proposal.md. PostgreSQL sends `NOTIFY pgrst` after provisioning. The production Cloud Run service co-locates Go and PostgREST and currently has request-based CPU allocation with no minimum instance. The database schema and privileges exist when the API probe fails, and a separately running PostgREST process against the same database receives reload notifications.

## Goals / Non-Goals

**Goals:** Keep the PostgREST listener runnable between requests and preserve the existing readiness gate; use a real new application to verify recovery.

**Non-Goals:** Claim the precise notification-loss mechanism is proven, change data API contracts, or replace PostgREST.

## Decisions

- Set service-level minimum instances to one and disable CPU throttling in the service deploy command. Google Cloud documents that instance-based billing allocates CPU for the container lifecycle, and minimum instances retain a warm instance. Both settings are necessary for an idle listener. Jobs retain their current configuration.
- Keep the production publish path: merge the fix into local `main`, update `deploy/main`, build and deploy a candidate, then switch traffic. A direct console setting would be overwritten by the next script release.
- Verify with an operator-owned new app and data table. If readiness still fails, restore the previous traffic revision and investigate the PostgREST response and notification path before claiming repair.

## Risks / Trade-offs

- [Continuous instance charges] → Document the cost and retain a single minimum instance.
- [The stale cache has another cause] → Treat new-app creation and data API verification as the release acceptance gate; roll back traffic if it fails.
- [Minimum instances are best effort and instances restart] → Keep idempotent provisioning and readiness probes; the setting is a mitigation, not a replacement for retry logic.

## Migration Plan

No database migration. Publish the new Cloud Run service revision, verify a new app, and keep the previous revision available for traffic rollback.
