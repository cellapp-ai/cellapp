# Proposal

## Why

The published CLI version does not include recent deployment recovery and default-origin changes. Users also have no prompt when a newer CLI is available.

## What Changes

- Bump the public `@cellapp/cli` package and its workspace packages to `0.2.1`.
- On the first CLI invocation of each local calendar day, query the public npm registry for the latest stable version and print an upgrade hint to stderr when it is newer.
- Keep commands usable when the registry or local update state is unavailable; do not send credentials or project data to the registry.

## Capabilities

### Modified Capabilities

- `skill-deployment`: Provide a bounded, daily CLI update notice without changing command results.

## Impact

Public CLI startup, package metadata, tests, and user documentation. No server or HTTP contract changes and no new runtime dependency.
