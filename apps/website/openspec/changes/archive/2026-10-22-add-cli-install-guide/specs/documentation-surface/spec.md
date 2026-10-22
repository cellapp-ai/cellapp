## ADDED Requirements

### Requirement: Verified CLI installation choices

Both Docs languages SHALL show an Installation section for users who choose to operate the CLI themselves. It SHALL state the supported Node.js prerequisite, provide the published npm package's on-demand and global install commands, and show how to verify the `cellapp` command. The guide SHALL explain that Agent-led deployment does not require manual installation. It SHALL not present an unverified shell installer or package-manager distribution as available.

#### Scenario: Reader runs the CLI manually
- **WHEN** a reader opens the Installation section in either language
- **THEN** they can copy an exact on-demand command or global install command and identify the matching help check

#### Scenario: Reader uses an Agent
- **WHEN** a reader follows the Agent deployment path
- **THEN** the installation section makes clear that the Agent obtains the client without a manual setup step
