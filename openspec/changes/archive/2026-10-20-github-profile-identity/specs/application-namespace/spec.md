# Spec Delta

## MODIFIED Requirements

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
