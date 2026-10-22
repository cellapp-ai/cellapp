# Add CLI installation guide

## Why

The Docs quickstart currently tells users that an Agent fetches the client but does not show a direct installation path. Readers who want to run `cellapp` themselves need verified, copyable commands like the installation section in OpenCode's documentation.

## What Changes

- Add an Installation section to both Docs languages with the supported Node.js prerequisite, on-demand command, optional global npm installation, and a help check.
- Keep the Agent-first deployment path and clarify that it requires no manual installation.
- Use the published npm package and existing command names. Do not advertise an install script or package managers without a verified Cellapp distribution.

## Capabilities

### Modified Capabilities

- `documentation-surface`: Add a manual CLI installation path alongside the existing Agent workflow.

## Impact

Website content, Docs rendering and browser acceptance checks. No CLI, service, or deployment protocol changes.
