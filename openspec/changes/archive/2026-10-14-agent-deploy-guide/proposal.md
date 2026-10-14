# Proposal

## Why

Users currently need to discover a separate Skill and CLI installation path before deploying. The public website gives little help for a first deployment. A public guide should let users give one request to an Agent and let the Agent handle setup, authorization and publication.

## What Changes

- Merge the existing `cellapp-website` source into `apps/website` without modifying the source checkout.
- Add a public `https://cellapp.ai/deploy` page with a copyable Agent request and a clear static deployment path.
- Publish a machine-readable Markdown guide at `https://cellapp.ai/deploy/AGENTS.md` without login.
- Update the public Skill and README so an Agent checks, installs and verifies the CLI when needed, then builds and deploys the local project.
- Keep future selling as product direction; do not suggest checkout or paid distribution exists yet.

## Capabilities

### Added Capabilities

- `public-website`: Public deployment entry in the Cloudflare Pages website, separate from the authenticated control domain.

### Modified Capabilities

- `skill-deployment`: Agent-led setup and public deployment instructions.

## Impact

Website static build, public Skill and documentation. The Go control service and deployment protocol are unchanged. Cloudflare Pages upload remains a separate release step.
