## ADDED Requirements

### Requirement: Public deployment entry

The public website SHALL expose `https://cellapp.ai/deploy` without login, with a copyable one-request Agent prompt and a link to `https://cellapp.ai/deploy/AGENTS.md`. The page SHALL describe supported static deployment and shall not present future selling features as available.

#### Scenario: Signed-out visitor starts deployment

- **WHEN** a visitor opens `/deploy` without a console session
- **THEN** the page offers the prompt and explains that the Agent handles CLI setup and the user approves browser authorization

### Requirement: Independent website delivery

The website SHALL build complete static pages and the deployment Markdown in `apps/website/dist/`. Its pnpm dependencies and Cloudflare Pages upload SHALL remain separate from the root npm workspace and the Go console release.

#### Scenario: Site release is prepared

- **WHEN** the website build succeeds
- **THEN** `dist/deploy/index.html` and `dist/deploy/AGENTS.md` are present alongside the existing static pages, ready for a separate Cloudflare Pages upload
