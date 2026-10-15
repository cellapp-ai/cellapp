# Spec Delta

## MODIFIED Requirements

### Requirement: Browser approved device authorization

The system SHALL bind a time-limited device authorization request to the owner who explicitly approves it in the browser on `cellapp.ai`. Only the client holding that request's device secret SHALL claim a deployment credential once. Approval SHALL grant owner-scoped management rather than per-project access.

#### Scenario: First deployment authorization

- **WHEN** a client without a current root-host credential starts authorization and the signed-in owner approves the matching code at `/dashboard/device`
- **THEN** the client polls the existing device protocol, receives an owner-scoped credential, and can continue deployment

#### Scenario: Rejected or expired request

- **WHEN** the owner rejects the code or the request expires
- **THEN** no credential is issued and the client reports that authorization must restart

#### Scenario: Code replay and polling abuse

- **WHEN** a caller has only the browser code, claims an already claimed request, or polls too quickly
- **THEN** the server rejects the claim or returns a slowdown response without issuing another credential

### Requirement: Credential lifecycle and confidentiality

The deployment client SHALL save a deployment credential in the user's private local configuration for reuse across that owner's projects on the same control origin; the credential SHALL NOT be stored in project state, uploaded artifacts, or logs. The system SHALL support expiration and owner revocation, and client logout SHALL revoke and remove the local credential.

#### Scenario: Reuse and revoke

- **WHEN** the same local user deploys another owned project using a valid root-host credential and later revokes it in the browser or signs out locally
- **THEN** the second project needs no new approval, and subsequent management requests with the revoked credential are rejected

#### Scenario: Expired credential

- **WHEN** the client uses an expired credential
- **THEN** the server rejects management and the client starts a fresh device authorization

## ADDED Requirements

### Requirement: Fresh authorization after official domain migration

The client SHALL NOT copy or reuse a token stored for `https://console.cellapp.ai` as its `https://cellapp.ai` credential. A first operation after migration SHALL ask for a new browser approval, while preserving the owner's existing application associations and the ability to revoke old credentials from account management.

#### Scenario: Existing machine changes origin

- **WHEN** a machine has only a credential stored for the retired official origin and opens an existing linked project after migration
- **THEN** the user approves one new root-host request, and that new credential can serve other owned projects on the machine without per-project approval

#### Scenario: Approval uses a different account

- **WHEN** the new browser approval is completed by an account that does not own the linked application
- **THEN** management of that app is rejected and the client does not create another app to mask the mismatch
