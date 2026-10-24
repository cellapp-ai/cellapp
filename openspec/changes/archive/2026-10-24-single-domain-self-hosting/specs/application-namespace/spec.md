# Spec Delta

## ADDED Requirements

### Requirement: Single registrable domain hosting

The platform SHALL allow a production control host and application wildcard base to share one registrable domain. The control host MAY equal the application base, so `example.com/dashboard` and `<app>.example.com` work together; it MAY also be the parent or a sibling of a dedicated base such as `apps.example.com`. Existing separate-domain configurations SHALL remain valid. Control routing MUST use the exact configured host, and a control host beneath the application wildcard MUST be rejected.

#### Scenario: Apex and direct wildcard
- **WHEN** an operator configures `https://example.com` as the control origin and `example.com` as the application base
- **THEN** `/dashboard` is served only on `example.com`, and a published application is served on its assigned `*.example.com` host

#### Scenario: Apex and nested wildcard
- **WHEN** an operator configures `https://example.com` as the control origin and `apps.example.com` as the application base
- **THEN** the control route stays on `example.com` and the application uses its assigned `*.apps.example.com` host

#### Scenario: Existing separate or sibling hosts
- **WHEN** an operator uses control and application bases on separate registrable domains or non-overlapping sibling subdomains
- **THEN** valid existing hosts continue to route to the same control and application surfaces

#### Scenario: Control host within application wildcard
- **WHEN** the control host is a child of the configured application base, or a host does not map to a published application
- **THEN** unsafe configuration is rejected before serving, and an unknown host never receives another application's content
