# Spec Delta

## Purpose

Provides task-oriented documentation that helps a new Cellapp user move from installation to a first deployment, while keeping command reference and troubleshooting discoverable.

## ADDED Requirements

### Requirement: Documentation information architecture
The documentation surface SHALL expose a visible sidebar or equivalent navigation containing Getting Started, Deploy, CLI/API Reference, Configuration, and Troubleshooting sections.

#### Scenario: Visitor finds a task
- **WHEN** a visitor opens the Docs surface
- **THEN** they can identify the current section, see the available documentation groups, and reach the getting-started guide within one interaction

### Requirement: Copyable examples
Documentation SHALL render commands and code examples in clearly labeled code blocks with a copy action and accessible feedback after copying.

#### Scenario: Visitor copies a command
- **WHEN** a visitor activates the copy action for a code block
- **THEN** the exact example text is copied and the interface confirms success without navigating away

### Requirement: Documentation progression
The getting-started path SHALL explain prerequisites, installation, first deployment, and how to verify/share the resulting app, with contextual links to the next relevant step.

#### Scenario: New user follows the quickstart
- **WHEN** a new user follows the getting-started content in order
- **THEN** each step supplies the command or expected result needed to continue, ending in a clear deployment/share action

