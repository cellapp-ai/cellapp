# Spec Delta

## ADDED Requirements

### Requirement: Root-host website and control path routing

The production edge SHALL keep public website paths on `cellapp.ai` available without owner authentication and forward only declared control pages, browser APIs, OAuth/device endpoints, deployment APIs, and control assets to the authenticated Cloud Run origin. The edge MUST preserve method, path, query, body, host assertion, and host-scoped cookies, and MUST NOT cache private control responses as public website content.

#### Scenario: Public page and owner API on one host

- **WHEN** requests for `/`, `/deploy/`, `/dashboard`, `/api/console/session`, and a deployment API reach `cellapp.ai`
- **THEN** the public pages come from the website and the declared management/API paths reach the control service with their original request semantics

#### Scenario: Undeclared path and visitor host

- **WHEN** an unknown root-host path or a published visitor application host is requested
- **THEN** the unknown path does not expose a control fallback, and visitor content remains subject to the existing isolated-host access checks

#### Scenario: Forged origin assertion

- **WHEN** a direct caller attempts to present `cellapp.ai` as an original host without valid edge proof
- **THEN** Cloud Run rejects it as a control request

### Requirement: Coordinated canonical-host cutover

The release procedure SHALL verify website, dashboard, OAuth callback, device approval, Browser API, deployment API, and visitor-host flows before declaring the root-host migration complete. It SHALL record separate website, edge, service, and client versions and an actionable rollback path. The retired `console.cellapp.ai` host MUST NOT keep a credential-bearing API path operational.

#### Scenario: Candidate release verification

- **WHEN** compatible website, edge, service, and client candidates are prepared
- **THEN** operators can validate public and private paths, same-origin writes, a real approval flow, and an existing-project deployment before the retired API host is disabled

#### Scenario: Cutover fails

- **WHEN** any required route, OAuth callback, or deployment operation fails during cutover
- **THEN** operators stop the transition or restore the prior coordinated version without losing database or release objects, and do not claim the migration complete
