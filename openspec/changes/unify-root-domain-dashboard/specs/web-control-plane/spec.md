# Spec Delta

## MODIFIED Requirements

### Requirement: Owner session and management entry

Web SHALL use the existing browser owner session and provide login, logout, and a safe return after login on `cellapp.ai`. The canonical owner entry SHALL be `/dashboard`, with the real application list as its default content. Sessions and private data MUST remain unavailable to visitor application hosts.

#### Scenario: Open the control domain

- **WHEN** a user opens `https://cellapp.ai/dashboard`
- **THEN** a signed-in owner sees the real application list, a signed-out user sees the login entry, and private information is hidden while session state is loading

#### Scenario: Return from login

- **WHEN** a user completes login with a return target under `/dashboard`, including a device approval URL with a code
- **THEN** the browser returns to that safe management URL with the device code intact; an external or non-management return target is rejected

#### Scenario: Logout or expired session

- **WHEN** the user logs out or a browser API detects a missing or expired session
- **THEN** the current server session is invalidated on logout, private state and newly revealed keys are cleared, the interface enters dashboard login, and APIs return 401 rather than login HTML

### Requirement: Browser API security and existing client compatibility

Browser owner APIs SHALL remain under `/api/console` on `https://cellapp.ai`, with separate session and Bearer boundaries, per-resource owner checks, and same-origin write checks. Every browser write MUST reject a missing or nonmatching Origin. Existing CLI Bearer API paths, OAuth and device polling request/response contracts SHALL remain intact on the new host; owner cookies MUST NOT authorize CLI APIs or visitor application data.

#### Scenario: Uniform browser API prefix

- **WHEN** the Web UI requests session, logout, app management, data settings, credential management, or device decisions
- **THEN** those JSON endpoints remain under `/api/console`, separate from `/dashboard` pages and CLI API paths

#### Scenario: Same origin owner mutation

- **WHEN** a signed-in owner submits a valid management write from `https://cellapp.ai`
- **THEN** the request passes session, Origin, input, and ownership checks and the private response is not shared-cacheable

#### Scenario: Cross origin or missing origin

- **WHEN** a browser write has no Origin or one from another host, including a visitor application host
- **THEN** the server rejects it without changing apps, grants, sessions, credentials, or device decisions

#### Scenario: Browser cookie cannot authenticate the CLI API

- **WHEN** a request carries only an owner Cookie to a Bearer application API, or a visitor host requests a browser API
- **THEN** it gains no management privilege; valid Bearer and device polling calls keep their existing protocol

### Requirement: Reproducible same origin Web delivery

The Go service SHALL serve built management HTML and assets on the root control host only for `/dashboard` and known management descendants and asset paths. Valid management deep links SHALL refresh; missing artifacts SHALL fail service startup. The management shell MUST NOT replace public website pages, unknown APIs, missing assets, or visitor gateway responses. Private page responses SHALL remain uncacheable and protected from framing.

#### Scenario: Build and deploy matching artifacts

- **WHEN** a complete build is started with the configured Web artifact directory
- **THEN** `/dashboard`, its deep links, scripts, styles, and fonts load on `cellapp.ai`; missing required artifacts fail startup and independent maintenance commands do not require Web assets

#### Scenario: Refresh and unknown routes

- **WHEN** a user refreshes a valid `/dashboard` deep link or requests an unknown management/API/asset path
- **THEN** a valid deep link loads the management page and an unknown path returns 404 rather than a successful HTML fallback

#### Scenario: Verify secure browser integration

- **WHEN** cross-service browser acceptance runs
- **THEN** it verifies Secure cookies, login, management, and origin rejection on the root host against an isolated database and actual Web artifacts, distinguishing test doubles from real external services

### Requirement: Personal profile settings

Web SHALL show the current account ID and login mode under `/dashboard/profile` and allow the current owner to set or clear a display name. The session, not a client-supplied account ID, SHALL select the account; changing the name MUST NOT change account identity, app addresses, or access rights.

#### Scenario: Open profile and save a name

- **WHEN** a signed-in owner opens `/dashboard/profile` and submits a valid display name
- **THEN** only that owner's name changes and remains visible after refresh

#### Scenario: Clear or reject a name

- **WHEN** the owner submits an empty name or an invalid name or JSON body
- **THEN** empty clears the name; an invalid value returns an explicit error without changing the stored name

#### Scenario: Authentication and origin boundaries

- **WHEN** a request has no owner session or attempts a cross-origin profile update
- **THEN** the update is rejected; a CLI Bearer credential cannot substitute for the browser session

#### Scenario: Refresh and uncertain result

- **WHEN** the owner refreshes `/dashboard/profile` or an update has an uncertain result
- **THEN** the deep link loads correctly and the UI offers a fresh read without claiming success or automatically retrying a write

## ADDED Requirements

### Requirement: Canonical management URLs and legacy browser links

Management navigation SHALL use `/dashboard` and descendants for login, applications, credentials, device approval, and profile. Legacy `console.cellapp.ai` browser GET/HEAD links MAY redirect to equivalent root-host management URLs, preserving safe query context; that host MUST NOT serve the retired CLI/API endpoints.

#### Scenario: Deep link and device approval

- **WHEN** a user opens `/dashboard/applications/<owned-app-id>` or `/dashboard/device?code=<code>` directly
- **THEN** the corresponding management page loads, and the code survives sign-in without approving it automatically

#### Scenario: Retired API host

- **WHEN** a client sends a deployment or browser API request to `console.cellapp.ai`
- **THEN** the request is not processed as a management operation and is not forwarded with credentials to the new host
