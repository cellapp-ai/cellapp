## ADDED Requirements

### Requirement: Owner slug settings

The authenticated console SHALL show the current user slug in Profile settings and the current app slug and URL in each owned application's details. It SHALL let the owner edit either slug, show validation and conflict errors, and refresh the server state after an uncertain write. A user-slug change SHALL update current addresses of all owned apps atomically.

#### Scenario: Edit an application slug
- **WHEN** an owner saves a valid available app slug
- **THEN** the details show the new readable URL and prior addresses remain assigned to the same app

#### Scenario: Edit a user slug
- **WHEN** an owner saves a valid available user slug
- **THEN** all owned apps get current URLs ending with that slug, or the whole change fails without partial updates

#### Scenario: Unauthorized or uncertain write
- **WHEN** a non-owner, cross-origin request or disconnected browser attempts a slug edit
- **THEN** the server rejects unauthorized writes, and the console does not report uncertain writes as successful
