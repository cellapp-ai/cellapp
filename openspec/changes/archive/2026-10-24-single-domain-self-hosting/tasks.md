# Tasks

## 1. Production host boundary

- [x] 1.1 Update Go configuration to accept apex/direct, apex/nested, sibling and separate-domain layouts while rejecting a control hostname inside the application wildcard; add table tests for the valid and invalid layouts and run the targeted Go tests.
- [x] 1.2 Verify exact control-host routing, unknown-host denial, host-only cookies, and rejection of app-origin or missing-Origin owner writes under a shared registrable domain; preserve Bearer client behavior and run the targeted integration tests.

## 2. Single-host setup and instructions

- [x] 2.1 Update self-host setup to accept both single-domain wildcard layouts without altering existing secrets or certificates on repeat setup; verify generated Caddy host rules, certificate checks, invalid nesting and old two-domain setup with isolated fixtures.
- [x] 2.2 Update the dedicated self-hosting guide with one-domain DNS, TLS and OAuth examples plus both wildcard forms; check all commands and links against the current Compose/startup assets.

## 3. Coordinated acceptance

- [x] 3.1 Revise the still-active Compose self-hosting delta and related design notes so later sync cannot restore the two-domain requirement; strictly validate both changes.
- [x] 3.2 Run relevant Go race, browser/security and self-host checks, record passed/failed/unrun work and limitations, and confirm no existing two-domain behavior regressed.

## Workflow follow-up

- After acceptance, sync main specs and archive this change; commit, merge to local main and push according to repository Gitflow.
