# Spec Delta

## MODIFIED Requirements

### Requirement: Agent-led CLI setup

The public deployment instructions and Skill SHALL make the Agent responsible for obtaining, verifying, and running the published deployment client on demand. The user SHALL NOT need to install a global CLI or learn a CLI command to deploy through the Skill. The Agent MUST report a missing supported runtime, failed package retrieval, or failed verification as a setup blocker and MUST NOT claim publication succeeded.

#### Scenario: CLI is missing

- **WHEN** an Agent follows the guide on a machine with supported Node.js and npm but no installed `cellapp` command
- **THEN** it runs a verified published client on demand and continues without asking the user to install a global command

#### Scenario: Setup cannot complete

- **WHEN** Node.js is unsupported or the published client cannot be fetched or verified
- **THEN** the Agent stops before deployment and reports the specific blocker and recovery step

### Requirement: Public Agent deployment guide

The public website SHALL expose a login-free `https://cellapp.ai/deploy/AGENTS.md` Markdown guide describing the current static deployment flow and its limits. It SHALL direct the Agent to use the published deployment runtime rather than reimplement the upload and authorization protocols, while presenting only the Skill request and browser approval as user steps.

#### Scenario: Agent reads instructions

- **WHEN** an unauthenticated Agent requests the guide
- **THEN** it receives Markdown covering project inspection, on-demand tool execution, browser approval, publication verification, and failure recovery

## ADDED Requirements

### Requirement: Official control origin migration in project state

For an existing `cellapp.json` or supported legacy project association whose origin is exactly `https://console.cellapp.ai`, the deployment client SHALL migrate the association to `https://cellapp.ai` without changing its application ID, create request ID, or pending deployment identity. Other origins MUST remain unchanged; conflicting associations MUST fail without overwriting either file.

#### Scenario: Existing official-origin application

- **WHEN** an Agent deploys a project linked to an existing app at the exact retired official origin
- **THEN** the client selects the new official origin, preserves the linked app and pending operation state, and does not create a replacement app

#### Scenario: Custom origin or conflicting state

- **WHEN** a project uses another explicit origin or the current and legacy project files conflict
- **THEN** the client does not silently redirect or overwrite that state and reports the conflict when it cannot select a safe association

#### Scenario: Interrupted migration

- **WHEN** local state persistence fails before the new origin is recorded
- **THEN** the next run can repeat migration without losing the app association or duplicating a create/publish request
