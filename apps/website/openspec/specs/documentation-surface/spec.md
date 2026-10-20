# Documentation Surface Specification

## Purpose
Provides task-oriented documentation that helps a new Cellapp user move from installation to a first deployment, while keeping command reference and troubleshooting discoverable.

## Requirements

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

### Requirement: Complete bilingual guides and language choice
The Docs surface SHALL provide equivalent, complete English and Simplified Chinese guides at `/docs/` and `/docs/zh-CN/` respectively. Both routes SHALL show a visible language switch linking to the corresponding section in the other version, localize the guide text and navigation, and declare the corresponding document language and page metadata. Neither version SHALL be a summary of the other.

#### Scenario: Reader chooses a language
- **WHEN** a visitor opens either Docs route and selects the other language
- **THEN** they reach the complete guide in that language through a normal link, including when JavaScript is unavailable

#### Scenario: Reader opens the Chinese guide directly
- **WHEN** a visitor opens or refreshes `/docs/zh-CN/`
- **THEN** the Chinese guide, navigation, and language switch are available in the returned static HTML with Chinese language and metadata declarations

### Requirement: Readable guide structure
The Docs surface SHALL show a clear page summary, a navigable outline with the current section identified, descriptive headings, short paragraphs, ordered steps, and contextual links between related topics. Its main text and examples SHALL remain readable on narrow and wide screens without horizontal page scrolling.

#### Scenario: Reader locates a task
- **WHEN** a visitor opens either Docs language on desktop or mobile
- **THEN** they can locate first deployment, verification, configuration and troubleshooting guidance from the document outline and move to a section with one interaction

### Requirement: Complete first-deployment path
Each language's getting-started guide SHALL state the static-output prerequisite, the Agent prompt entry point, browser authorization, what a successful publication returns, and how to verify the resulting HTTPS app before sharing it. Each step SHALL tell the reader what action or result comes next.

#### Scenario: New user follows the guide
- **WHEN** a reader starts with a local frontend project and follows the first-deployment guide
- **THEN** they can reach the copyable prompt, know when to approve browser access, and know which returned URL and page behavior to verify

### Requirement: Accurate boundaries and references
Both guides SHALL distinguish the user's local build and Agent-run client from Cellapp's static hosting, explain that embedded frontend values are public, and link to the public Agent guide and verified client/contracts for exact command or API details. They SHALL not present unsupported server execution, account, payment, or API behavior as available.

#### Scenario: Reader needs exact technical details
- **WHEN** a reader reaches a command or API reference topic
- **THEN** any shown syntax is verified against the published client or linked authoritative material, and a route to the fuller reference is available

### Requirement: Actionable recovery guidance
Troubleshooting in each language SHALL cover at least failed static output, broken assets or deep links, interrupted publication or authorization, and access-related viewing problems. Each topic SHALL describe a check, a corrective action, and when to return to the deployment flow or seek help.

#### Scenario: Deployment result is unusable
- **WHEN** a reader's published page is blank, a deep link fails, or the Agent reports a publishing error
- **THEN** the Docs surface provides a relevant diagnostic step and a next action rather than only a generic error description

### Requirement: Documentation works as static content
Each guide's essential text, outline, language switch, and links SHALL be available when its route is opened directly or refreshed, including when client-side JavaScript is unavailable. Copy controls, when offered, SHALL copy the displayed text and provide accessible success or failure feedback.

#### Scenario: Direct navigation without JavaScript
- **WHEN** a reader opens either Docs language directly with JavaScript disabled
- **THEN** the guide, section links and external references remain available for reading
