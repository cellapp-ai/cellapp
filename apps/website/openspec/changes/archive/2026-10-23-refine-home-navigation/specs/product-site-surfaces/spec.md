# Spec Delta

## MODIFIED Requirements

### Requirement: Shared site navigation
The site SHALL provide persistent navigation to the homepage through the Cellapp wordmark and to Deploy, Pricing, Docs, and Blog through text links. The active destination SHALL be visibly indicated. On the homepage, the two slashes in the wordmark SHALL use the active accent; on other routes, the wordmark SHALL use its standard appearance.

#### Scenario: Visitor opens the homepage
- **WHEN** a visitor opens `/`
- **THEN** the wordmark links to `/`, its two slashes are highlighted, and no separate Home text link appears

#### Scenario: Visitor changes product surface
- **WHEN** a visitor selects Deploy, Pricing, Docs, or Blog from the shared navigation
- **THEN** the selected surface opens through a stable URL, its text link is marked active, and the wordmark uses its standard appearance

## ADDED Requirements

### Requirement: Session-aware navigation action
The navigation action SHALL display “Deploy for free” for a visitor without a confirmed signed-in session and “Dashboard” for a visitor with a confirmed signed-in session. Both states SHALL lead to the existing dashboard entry, where an unauthenticated visitor can sign in.

#### Scenario: Signed-out visitor opens the site
- **WHEN** a visitor has no active Cellapp session
- **THEN** the navigation action reads “Deploy for free” and opens the dashboard sign-in flow

#### Scenario: Signed-in visitor opens the site
- **WHEN** the current browser session is confirmed as signed in
- **THEN** the navigation action reads “Dashboard” and opens the dashboard

#### Scenario: Session status is unavailable
- **WHEN** the site is opened as a local file or the session check fails or returns an unexpected response
- **THEN** the initial static page remains usable and the navigation action continues to read “Deploy for free” without presenting the visitor as signed in
