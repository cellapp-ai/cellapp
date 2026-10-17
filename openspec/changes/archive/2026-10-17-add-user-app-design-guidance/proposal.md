# Proposal

## Why

Cellapp's public Agent instructions explain how to build and deploy a static app, but give little help when a user asks the Agent to develop an app without a UI design. A lightweight design guide can improve the resulting interface without making the user choose a framework, style library, or design process.

## What Changes

- Publish a standalone, Agent-readable guide for developing user-owned Cellapp apps when no UI design is supplied. It offers adaptable defaults for visual direction, content hierarchy, interaction states, responsive layout, accessibility, and visual review.
- Provide English and Simplified Chinese editions of the public app-design guide, Agent deployment runbook, and deployment Skill as separate, mutually linked Markdown files with equivalent guidance.
- Make the existing public deployment instructions and deployment Skill point to the guide for app-development requests. Agents use it without asking whether to enable it, while honoring user-specified design and established project conventions.
- Keep the guide framework and styling-tool agnostic. Do not prescribe StyleX, Tailwind, React, or a Cellapp-branded visual treatment; the Agent chooses suitable tools for the project and still delivers a buildable static artifact.
- Verify that the guide is present in the website's production static output and that the Agent entry points reference the published path.

## Capabilities

### New Capabilities

- `user-app-design-guidance`: Public, optional design guidance for Agents developing user-owned static apps without a supplied UI design, including its discoverability and verification.

### Modified Capabilities

- `skill-deployment`: The Agent deployment flow gains a pointer to the optional user-app design guide when the request includes development and lacks UI direction.

## Impact

Planning affects the public website's Agent-facing Markdown and static export, the public deployment Skill in the `skills` submodule, and their checks and documentation. There is no intended change to Cellapp hosting, CLI commands, HTTP contracts, application data, or the control console's design standards. The website and Skill have separate release paths and must be reported separately during implementation.
