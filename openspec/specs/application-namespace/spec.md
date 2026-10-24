# application-namespace Specification

## Purpose
Give applications readable, owner-scoped hosts while keeping the internal app ID and authorization boundary stable.

## Requirements

### Requirement: Owner namespace and application alias

The system SHALL give each owner one public user slug and each application one app slug unique within that user's namespace. Display names MAY repeat. New owners MAY choose both slugs on first creation; when a GitHub owner does not choose a namespace, the system SHALL prefer a valid form of the verified GitHub username and use a unique fallback when unavailable. The owner console SHALL permit editing both slugs after creation.

#### Scenario: Same name or alias in different namespaces
- **WHEN** two owners create applications with the same display name or app slug
- **THEN** each gets a distinct readable host and retains its own immutable app ID

#### Scenario: Alias conflict in one namespace
- **WHEN** a slug is too short, reserved, invalid, too long in combination, or its complete host label is occupied
- **THEN** creation or editing fails with a recognizable error and changes no existing address

#### Scenario: Default to verified GitHub login
- **WHEN** a GitHub owner creates a first application without an explicit namespace
- **THEN** its user slug is derived from the verified GitHub login if valid and available, otherwise a safe unique fallback is used; later GitHub renames do not change assigned application URLs

### Requirement: Slug format and reservation

The user slug MUST contain 3–61 lowercase ASCII letters, digits or hyphens and begin with a letter. Both slug types MUST reject reserved platform names and trailing hyphens. The complete `{app-slug}-{user-slug}` host label MUST fit 63 characters and be globally unique, including historical assignments.

#### Scenario: Invalid or previously assigned host label
- **WHEN** an owner submits invalid or reserved slugs, or a combination previously assigned to another app
- **THEN** no slug changes and the response identifies the invalid or occupied address

### Requirement: Stable share link and identity boundary

The system SHALL use `{app-slug}-{user-slug}.<APPS_DOMAIN>` as the current public app URL, resolving it to the immutable app ID. A control-origin share link SHALL redirect to the current app URL. App authorization, sessions, data and deployments SHALL continue to use the immutable app ID. Legacy ID hosts and previously assigned readable hosts SHALL remain bound to their original apps, and no assigned name SHALL be recycled.

#### Scenario: Open an alias
- **WHEN** a visitor opens a current or prior address of an active app
- **THEN** normal login and app admission apply for the same internal app ID; after deletion the address stops serving content

#### Scenario: Delete an application
- **WHEN** an application is deleted
- **THEN** its aliases and hosts stop serving content and remain occupied so stale links cannot point to another app

#### Scenario: Same-origin write and login return
- **WHEN** a visitor uses a readable or legacy app host
- **THEN** login returns to that validated host, and writes accept only its exact Origin

### Requirement: Existing application migration

The system SHALL backfill readable hosts for existing valid, unambiguous slug pairs without changing their app IDs, aliases, deployment associations or data schemas. Existing pairs that cannot form an available 63-character host label SHALL retain their ID host until edited.

#### Scenario: Upgrade a control database
- **WHEN** an existing database runs the additive host migration
- **THEN** eligible apps receive readable current URLs, ineligible apps keep working ID URLs, and prior links remain assigned to their original apps

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
