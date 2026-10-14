# Spec Delta

## ADDED Requirements

### Requirement: Recoverable initial application creation
The CLI SHALL preserve a stable creation request and its initial access key outside project files until the first release succeeds. A provisioning failure after application creation MUST leave a reusable application association and MUST NOT cause a retry to create another application.

#### Scenario: Provisioning fails after creation
- **WHEN** application creation commits but data provisioning reports failure
- **THEN** the CLI records the returned application ID, retains the key in private credential storage, reports failure, and a retry targets the same application

#### Scenario: Creation response is lost
- **WHEN** the application creation response is lost before the CLI receives it
- **THEN** the next attempt uses the original request ID and key, allowing the server to return the same application

#### Scenario: Older project lacks association
- **WHEN** the owner selects an application they own and supplies an output directory to link an unassociated project
- **THEN** the CLI writes a non-secret project association without creating another application
