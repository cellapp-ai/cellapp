# Spec Delta

## ADDED Requirements

### Requirement: Owner recovery of failed data provisioning
The control API SHALL report a stable data provisioning failure code to the application owner and SHALL let that owner safely retry provisioning of a non-deleted application. Failure MUST NOT be reported as ready or expose database internals to the client.

#### Scenario: Retry after transient failure
- **WHEN** the application owner requests reconciliation after a temporary data service failure is resolved
- **THEN** the platform retries idempotently and reports ready only after database setup and API verification succeed

#### Scenario: Persistent provisioning failure
- **WHEN** a retry still fails
- **THEN** the application remains unready, the owner can inspect a stable error code, and server logs contain the underlying failure for operators

#### Scenario: Cross-owner retry
- **WHEN** another owner requests reconciliation for the application
- **THEN** the platform denies the request without changing its data state
