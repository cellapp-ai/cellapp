# Spec Delta

## Purpose

Provides a coherent, navigable product website around the existing Cellapp landing page so visitors can move from product discovery to evaluation and learning without leaving the static site.

## ADDED Requirements

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

