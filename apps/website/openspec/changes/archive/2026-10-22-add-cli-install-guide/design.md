# Design

## Context

The public `@cellapp/cli` package exposes the `cellapp` binary and requires Node.js 22.12 or later. The public client README documents `npm install -g @cellapp/cli` and an on-demand `npm exec` command. The Agent runbook deliberately uses on-demand execution, so manual installation is optional.

## Decision

Add a top-level Installation section after Getting Started in `docGuides`, preserving matching anchors across English and Chinese. Present on-demand execution first and global npm installation second. Each executable command gets its own labeled, copyable block so copy never includes commentary or an unrelated command. Reuse `CodeBlock` and supply a localized label in the content data. Mention that Agent deployments continue to obtain the client themselves.

## Validation

Check the package metadata, lint and build, then run the static preview browser suite with exact clipboard checks, both languages, no-JavaScript HTML, and narrow-screen overflow checks.
