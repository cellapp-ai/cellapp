# Proposal

## Why

An application can be committed in the control database while data provisioning fails. The first CLI deploy then exits before writing project association, so a retry creates a new request and collides with the existing slug. The platform also hides the concrete provisioning failure from the owner and operator.

## What Changes

- Return the created application ID on data provisioning failure and record the underlying error in server logs.
- Preserve the creation request and initial access key in appropriate local storage until publication; reuse the application on retry.
- Let the owner retry data provisioning and inspect a stable failure code; allow linking an older orphaned application to a new project association.
- Make the local PostgreSQL initialization hook executable so a fresh app-data database starts successfully.
- Update the CLI contract, deployment Skill, and operating instructions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `skill-deployment`: Restore an interrupted first deployment without creating another application or losing the initial key.
- `application-data`: Expose an owner-only provisioning retry and failure state.

## Impact

Go control API and application data reconciliation, public CLI and Skill, HTTP contract, and application data operations documentation. No schema migration or new dependency.
