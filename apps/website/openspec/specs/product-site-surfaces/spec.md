# Product Site Surfaces Specification

## Purpose
Provides a coherent, navigable product website around the existing Cellapp landing page so visitors can move from product discovery to evaluation and learning without leaving the static site.

## Requirements

### Requirement: Shared site navigation
The site SHALL provide persistent navigation to Home, Pricing, Docs, and Blog, with the current destination visibly indicated and a primary action that leads to the Cellapp getting-started path.

#### Scenario: Visitor changes product surface
- **WHEN** a visitor selects Pricing, Docs, or Blog from the shared navigation
- **THEN** the selected surface opens through a stable URL and the navigation marks it as active

### Requirement: Responsive and accessible shell
The site SHALL remain usable on narrow and wide viewports, provide keyboard-visible focus states, preserve readable contrast, and expose meaningful labels for navigation, buttons, code blocks, and decorative imagery.

#### Scenario: Visitor uses a mobile viewport
- **WHEN** the viewport is narrower than the desktop layout
- **THEN** navigation, cards, tables, code examples, and footer content reflow without horizontal page scrolling or clipped primary actions

### Requirement: Static-hosting compatibility
The site SHALL render all initial content from local/static data and SHALL define a direct-navigation fallback for supported routes so refreshing a deep link does not produce an unhandled application error.

#### Scenario: Visitor refreshes a deep link
- **WHEN** a visitor directly opens or refreshes a supported Pricing, Docs, or Blog URL
- **THEN** the corresponding page renders with its metadata and shared shell intact

### Requirement: Approved homepage slogan
The homepage's first screen SHALL display the exact slogan “your apps, your rules.” as the primary brand statement. The product explanation and usage instruction SHALL remain visible beside or below it so the slogan does not have to explain the service by itself.

#### Scenario: Visitor reads the first screen
- **WHEN** a visitor opens the homepage
- **THEN** the slogan reads “your apps, your rules.” and the visitor can also see what Cellapp provides and how to begin

### Requirement: Four-part product vision
The homepage SHALL present the exact line “Shape, Ship, Share and Sell.” as a four-part vision. Its accompanying explanations SHALL connect Shape, Ship and Share to Agent-assisted local creation, static publication and access-sharing steps, and SHALL label Sell as a future capability with no purchase or sales action available in this flow.

#### Scenario: Visitor evaluates the four parts
- **WHEN** a visitor reads the four-part vision
- **THEN** they can distinguish current actions from the planned Sell stage and are not offered a working sales checkout

### Requirement: One-sentence product explanation
The homepage's first screen SHALL state in one self-contained sentence that Cellapp hosts a built static frontend as a shareable HTTPS application. The sentence SHALL describe a current capability without implying that Cellapp runs the project's build or application server.

#### Scenario: New visitor identifies the offer
- **WHEN** a visitor opens the homepage without prior product knowledge
- **THEN** the visible first screen contains one sentence that answers what Cellapp provides without requiring the visitor to interpret an illustration or another page

### Requirement: One-sentence usage instruction
The homepage's first screen SHALL state in one self-contained sentence how to start: ask the coding Agent working on the project to follow the Cellapp deployment guide, approve authorization in the browser when prompted, and receive the published app URL. It SHALL provide a direct action leading to the copyable deployment prompt and a link to detailed human-readable guidance.

#### Scenario: Visitor starts from the homepage
- **WHEN** a visitor reads the homepage's first screen and activates the primary action
- **THEN** the deployment prompt is available to copy and the next user action is understandable without reading marketing sections

### Requirement: Consistent public entry message
The homepage, deployment page, and their page descriptions SHALL describe the same static-hosting boundary and Agent-first authorization flow. Public messaging SHALL use the `Cellapp` brand spelling and SHALL not present draft pricing or future paid sales as available.

#### Scenario: Visitor follows the deployment path
- **WHEN** a visitor moves from the homepage to the deployment page
- **THEN** the instructions do not contradict the homepage's product and usage sentences
