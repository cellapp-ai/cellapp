# Spec Delta

## Purpose

The public website helps a visitor understand Cellapp's current static hosting offer and start the supported Agent-assisted deployment flow from the first screen.

## ADDED Requirements

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
