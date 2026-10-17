# Spec Delta

## Purpose

Provide public, reusable design guidance for Agents developing user-owned Cellapp static applications when the user has not supplied a UI design, without making the user choose a frontend framework or styling tool.

## ADDED Requirements

### Requirement: Public user-app design guide
The public website SHALL serve a login-free Agent-readable design guide at a stable path and include it in the complete static website output. The guide SHALL state that it applies to user-owned applications, not the Cellapp console or website design system.

#### Scenario: Agent retrieves the guide
- **WHEN** an unauthenticated Agent requests the published design-guide path
- **THEN** it receives the complete Markdown guidance without a console session

#### Scenario: Website build prepares publication
- **WHEN** the website production build completes
- **THEN** the design-guide file is present in the static output alongside the public deployment instructions

### Requirement: Bilingual public design guidance
The public user-app design guide SHALL be available in English and Simplified Chinese as separate Markdown files at stable paths. Each edition SHALL link to the other, and both SHALL provide equivalent instructions and Cellapp capability boundaries.

#### Scenario: Agent or reader selects a language
- **WHEN** an unauthenticated reader opens either design-guide edition
- **THEN** that edition is complete and provides a working link to the other language

### Requirement: Design defaults without user opt-in
The guide SHALL tell Agents to use its design recommendations when an app-development request lacks UI direction, without asking the user whether to enable them. It SHALL defer to explicit user design instructions and established project conventions and SHALL NOT prescribe a frontend framework, styling library, or Cellapp-branded appearance.

#### Scenario: No UI design supplied
- **WHEN** a user asks an Agent to develop an app without specifying its visual design
- **THEN** the Agent can infer an appropriate visual direction from the app's purpose and audience and proceed without a design-guidance opt-in question

#### Scenario: Existing design direction
- **WHEN** a user supplies a design direction or the project has an established visual system
- **THEN** the guidance directs the Agent to respect that direction instead of imposing a default look or tool

### Requirement: Complete and reviewable interface guidance
The guide SHALL cover visual hierarchy and consistency, responsive layouts, interaction and data states, accessible operation, and review of the actual built interface on narrow and wide screens. It SHALL distinguish recommendations from Cellapp hosting constraints and SHALL NOT claim that a styling tool alone ensures visual quality.

#### Scenario: Agent builds a data-entry app
- **WHEN** an Agent follows the guide to develop a data-entry app
- **THEN** it has guidance for forms, navigation, empty/loading/error/success feedback, keyboard and touch operation, and visual review of the static build
