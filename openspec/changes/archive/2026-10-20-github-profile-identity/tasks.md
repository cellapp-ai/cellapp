# Tasks

## Identity

- [x] Parse and validate GitHub login alongside immutable ID; persist the latest verified login with an additive migration without replacing custom display names.
- [x] Extend owner-scoped session/profile reads with effective name and stable avatar metadata; keep unauthorized and dev-mode behavior safe.

## Namespace

- [x] Prefer a valid GitHub login for the first implicit user slug; preserve explicit namespaces, existing slugs and historical URLs, and handle conflicts safely.

## Console

- [x] Replace the sidebar ID label with avatar and visible name, add image fallback and responsive styling, and reflect display-name edits promptly.
- [x] Update documentation and CSP for the fixed GitHub avatar host.

## Verification

- [x] Test OAuth parsing, username changes, default/fallback namespace, custom override, authorization, and browser account display.
- [x] Run relevant Go integration/race, frontend typecheck/lint/build/browser checks, and strict OpenSpec validation; record results.
