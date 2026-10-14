# Cloud Run Deployment Specification Delta

## MODIFIED Requirements

### Requirement: Authenticated third-party edge origin

Cellapp SHALL support an optional third-party HTTPS edge in front of Cloud Run. It MUST recover the browser hostname only from an authenticated edge assertion and use that hostname consistently for control and visitor routing, application login handoff, and visitor write origin checks. It MUST reject unauthenticated origin requests.

#### Scenario: Valid control and visitor host
- **WHEN** an authenticated edge request carries the configured control hostname or a single application hostname under the configured visitor domain
- **THEN** the request reaches the matching control or application route with its original path, method, body, and host-scoped cookies intact

#### Scenario: Forged or malformed original host
- **WHEN** a direct origin request omits the edge proof, presents an invalid proof, or names a host outside the configured control and visitor domains
- **THEN** Cellapp does not serve control or application content for that asserted host

#### Scenario: Direct Host compatibility
- **WHEN** a deployment without the third-party edge receives the browser Host directly through a trusted HTTPS ingress
- **THEN** the existing routing and authorization behavior remains available

#### Scenario: Published visitor application through the edge
- **WHEN** a visitor opens a published application through an authenticated edge request, signs in, and performs a same origin write
- **THEN** the authorization handoff returns to the requested application hostname, the page loads, and the same origin write passes its origin check

#### Scenario: Forged edge host or cross origin write
- **WHEN** the edge proof is invalid or a visitor write Origin differs from the authenticated application hostname
- **THEN** the server rejects the forged route or write without issuing access to another host
