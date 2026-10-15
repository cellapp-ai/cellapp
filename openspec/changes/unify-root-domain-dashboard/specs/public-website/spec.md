# Spec Delta

## MODIFIED Requirements

### Requirement: Public deployment entry

The public website SHALL expose `https://cellapp.ai/deploy` without login, with a copyable one-request Agent prompt and a link to `https://cellapp.ai/deploy/AGENTS.md`. The page SHALL describe supported static deployment and shall not present future selling features as available. The user-facing flow SHALL ask the user only to request deployment and approve browser authorization when needed; the Agent SHALL prepare and run the deployment tool.

#### Scenario: Signed-out visitor starts deployment

- **WHEN** a visitor opens `/deploy` without a console session
- **THEN** the page offers the prompt, explains that the Agent handles tool setup and publication, and asks the user to approve browser authorization when prompted

## ADDED Requirements

### Requirement: Public website and dashboard share the root host

The public website SHALL remain available at `https://cellapp.ai/` and its published public paths without owner login, while `/dashboard` and its management descendants SHALL be served by the owner control plane on the same host. A public-page request MUST NOT receive owner-private data or a management HTML fallback.

#### Scenario: Public and management navigation

- **WHEN** a visitor opens `/`, `/deploy/`, or `/deploy/AGENTS.md` and then follows the management entry
- **THEN** the public resources load without a session and the management entry opens `https://cellapp.ai/dashboard`

#### Scenario: Unknown public route

- **WHEN** a visitor opens an unknown public path
- **THEN** it does not return the authenticated dashboard shell as a successful public page
