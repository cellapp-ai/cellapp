## ADDED Requirements

### Requirement: Daily CLI update notice

The CLI SHALL attempt to check the public npm registry for a newer stable `@cellapp/cli` release on its first invocation of each local calendar day. It SHALL notify on stderr only when the latest version is newer, without changing command stdout or exit status. The request MUST NOT include Cellapp credentials or project data. Registry and local state failures MUST NOT prevent the requested command.

#### Scenario: New release available

- **WHEN** the first CLI invocation of the day receives a valid newer stable version
- **THEN** the CLI prints the version and upgrade command to stderr and runs the requested command normally

#### Scenario: Same day or offline

- **WHEN** the CLI runs again on the same local day, or npm cannot be reached within the bounded timeout
- **THEN** it does not repeatedly query npm that day and continues the requested command without an update-check error
