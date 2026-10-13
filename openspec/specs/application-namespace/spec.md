# application-namespace Specification

## Purpose
Give applications a human-readable, owner-scoped reference while preserving immutable app-ID hosts and their authorization boundary.

## Requirements

### Requirement: Owner namespace and application alias
The system SHALL give each owner one public namespace handle and each application one alias unique within that namespace. Display names MAY repeat. New owners MAY choose a valid namespace on their first application and MAY choose an application alias; otherwise the system assigns stable values. It SHALL return the reference and share URL in owner APIs and show them in the console and CLI guidance.

#### Scenario: Same name or alias in different namespaces
- **WHEN** two owners create applications with the same display name or alias
- **THEN** each application receives a distinct `@namespace/alias` reference and retains its own immutable app ID

#### Scenario: Alias conflict in one namespace
- **WHEN** an owner requests an alias already assigned to an application in that namespace
- **THEN** creation fails with a recognizable conflict; an omitted alias gets a unique suffix

### Requirement: Stable share link and identity boundary
The system SHALL resolve a share URL on the control origin to the existing ID-based app host. Authorization, app sessions, table access and app database schemas SHALL continue to use the immutable app ID.

#### Scenario: Open an alias
- **WHEN** a visitor opens an active application's share URL
- **THEN** the control service redirects to its ID host, where normal login and admission apply

#### Scenario: Delete an application
- **WHEN** an application is deleted
- **THEN** its alias stops resolving and remains occupied so a stale link cannot point to another application

### Requirement: Existing application migration
The system SHALL backfill a namespace and unique alias for existing owners and applications without changing their app IDs, hosts, deployment associations or data schemas.

#### Scenario: Upgrade a control database
- **WHEN** an existing database with owners and apps runs the additive namespace migration
- **THEN** each existing app receives a unique reference, and its prior app URL continues to work
