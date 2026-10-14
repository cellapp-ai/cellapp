## ADDED Requirements

### Requirement: Agent-led CLI setup

The public deployment instructions and Skill SHALL guide an Agent to check the local CLI, install the published `@cellapp/cli` package when absent and verify it before deployment. Installation failure SHALL be reported with the actionable blocker rather than treated as deployment success.

#### Scenario: CLI is missing

- **WHEN** an Agent follows the guide in a project with supported Node.js and no `cellapp` command
- **THEN** it installs the CLI, verifies the command and continues deployment without asking the user to perform setup manually

#### Scenario: Setup cannot complete

- **WHEN** Node.js is unsupported or npm installation fails
- **THEN** the Agent stops before deploying and reports the specific environment issue

### Requirement: Public Agent deployment guide

The public website SHALL expose a login-free `https://cellapp.ai/deploy/AGENTS.md` Markdown guide that describes the current static deployment flow and its limits. It SHALL direct the Agent to use the CLI rather than implement the upload protocol itself.

#### Scenario: Agent reads instructions

- **WHEN** an unauthenticated Agent requests the guide
- **THEN** the service returns Markdown covering inspection, setup, browser authorization, publishing, verification and failure recovery
