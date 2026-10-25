# Spec Delta

## ADDED Requirements

### Requirement: Reader-focused root documentation

The root README SHALL introduce Cellapp's current purpose, supported capabilities and limits, and a short path to use the hosted service. It SHALL link to separate self-hosting and contributor guides instead of embedding full environment, test, Cloud Run, or migration runbooks. Current command, domain, package, and licensing claims MUST match the maintained source and guides.

#### Scenario: New user opens the repository
- **WHEN** a reader opens the root README without prior Cellapp knowledge
- **THEN** they can identify what Cellapp hosts, how to start publishing an application, and where to find full self-hosting and development instructions

#### Scenario: Contributor seeks local setup
- **WHEN** a contributor follows the README development link
- **THEN** a separate guide provides the current recursive checkout, prerequisites, local dependencies, service startup, build, test, and relevant standards and operations links

#### Scenario: Operator seeks production setup
- **WHEN** an operator follows the README self-hosting link
- **THEN** they reach the full production Compose guide and can distinguish it from local development and the existing Cloud Run operations path

### Requirement: Bilingual project entry

The default root `README.md` SHALL be in English. A matching `README.zh-CN.md` SHALL provide a Chinese version, with language-switch links at the top of both files. Links from the English README to Chinese-only guides SHALL indicate their language.

#### Scenario: Reader switches language
- **WHEN** a reader chooses the Chinese link from the default English README, or the English link from the Chinese README
- **THEN** they reach the equivalent project entry in the selected language

### Requirement: Agent-first Quick start

Both README languages SHALL present a single Agent request that installs the Cellapp deployment Skill and publishes the current project as the primary path. They SHALL offer manual global CLI installation as a secondary path and identify browser authorization as the user's required first-use action.

#### Scenario: Reader delegates publication
- **WHEN** a reader follows the Agent path in either README
- **THEN** they can give their coding Agent one request to install the Skill, build locally, deploy, and return the app URL without manually installing the CLI
