# Spec Delta

## Purpose

Gives prospective users a fast, scannable way to understand CellApp plan differences, estimate the right starting point, and continue toward deployment with a clear conversion path.

## ADDED Requirements

### Requirement: Plan comparison
The pricing surface SHALL present the available plans in a comparable layout with plan name, intended audience, price or pricing label, billing cadence, included capabilities, and a primary action for each plan.

#### Scenario: Visitor compares plans
- **WHEN** a visitor views the pricing surface
- **THEN** they can compare the same core capability categories across all plans without opening separate dialogs or pages

### Requirement: Billing cadence selection
The pricing surface SHALL allow visitors to switch between the supported billing cadences and SHALL update displayed prices and billing context consistently across the plan cards.

#### Scenario: Visitor switches billing cadence
- **WHEN** a visitor selects a different billing cadence
- **THEN** all applicable plan prices, savings context, and billing labels update together while the selected cadence remains visibly selected

### Requirement: Pricing conversion path
Each plan SHALL provide a clear next action, and the recommended plan SHALL be visually distinguished without preventing visitors from choosing another plan or reading the full comparison.

#### Scenario: Visitor chooses a plan
- **WHEN** a visitor activates a plan action
- **THEN** the site routes them to the configured getting-started or contact destination and preserves the selected plan context where the destination supports it

