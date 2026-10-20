# Spec Delta

## ADDED Requirements

### Requirement: Coherent owner dashboard presentation

The authenticated dashboard SHALL present the same navigation, page hierarchy, action styles, and feedback patterns across applications, deployment credentials, device authorization, and profile settings. Application lists and details SHALL make the real application identity, address, publication state, and available actions distinguishable without implying unavailable capabilities.

#### Scenario: Scan owned applications
- **WHEN** an owner opens the dashboard with one or more applications
- **THEN** each application has a readable name, address, publication state, and identifiable paths to manage or open it; suspended and unpublished applications are not presented as live publications

#### Scenario: Move among management tasks
- **WHEN** an owner moves among the application list, application detail, credentials, device authorization, and profile
- **THEN** the navigation and page structure remain recognizable, the current destination and primary available action are clear, and existing URL, focus, and session behavior remains intact

#### Scenario: Narrow or long content
- **WHEN** an owner uses a narrow or wide viewport, 200% zoom, keyboard navigation, or long application names and addresses
- **THEN** primary information and actions remain readable and operable without page-wide horizontal overflow or hover-only disclosure

#### Scenario: No data or failed request
- **WHEN** an application list or management section is loading, empty, or unavailable
- **THEN** it shows the corresponding state and an appropriate next step without fabricated metrics, records, or a false success result

### Requirement: Read-only owner inspection of application table structure

The dashboard SHALL let an authenticated owner inspect the latest applied table definition of each owned application through an owner-scoped browser read. It SHALL show table names, columns and types, nullability, primary keys, indexes, access policies, and applied version. It MUST NOT expose table rows, database credentials, or table mutations through this view.

#### Scenario: Inspect applied structure
- **WHEN** an owner opens the tables area of an application with an applied table version
- **THEN** the dashboard lists that version's tables and reveals the selected table's columns, keys, indexes, and select/insert/update/delete policy modes from the applied definition

#### Scenario: No applied tables
- **WHEN** an owned application has no applied table version, or the latest applied definition contains no tables
- **THEN** the dashboard shows a truthful empty structure state and does not imply that rows or tables exist

#### Scenario: Pending or unavailable data service
- **WHEN** the application data service is pending, failed, or unavailable while the owner requests structure
- **THEN** the dashboard distinguishes a failed read from an empty structure, reports the available readiness state without internal database details, and offers a safe retry of the read

#### Scenario: Other owner or visitor requests structure
- **WHEN** a signed-out user, visitor host, or different owner requests an application's table structure
- **THEN** the owner-session boundary applies, a different owner's application is not disclosed, and no table names or policy metadata leak

#### Scenario: Read-only boundary
- **WHEN** the owner inspects or changes the selected table in the dashboard
- **THEN** no data rows are fetched and no schema, policy, or row mutation is submitted; existing CLI or Agent table management remains the write path
