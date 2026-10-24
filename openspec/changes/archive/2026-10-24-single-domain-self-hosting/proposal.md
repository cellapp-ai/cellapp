# Proposal

## Why

Personal self-hosting currently requires two purchased registrable domains even though the dashboard already lives under `/dashboard` and application hosts are isolated by hostname. A single-domain installation should support both `example.com` with `*.example.com` and `example.com` with `*.apps.example.com`, without weakening owner-session or application boundaries.

## What Changes

- Accept a control host and application wildcard base on the same registrable domain, including an identical apex/base pair; keep separate-domain installations valid.
- Preserve exact control-host routing, host-only owner cookies, application-scoped sessions, and exact-Origin checks for browser mutations; reject a control hostname that falls inside the application wildcard.
- Update single-host setup, TLS/DNS guidance, and an isolated browser/security acceptance path for both wildcard layouts.
- Coordinate the still-active Compose self-hosting delta so a later archive does not restore the two-domain requirement.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `application-namespace`: permit application host bases on the control host's registrable domain while maintaining unambiguous host routing and application isolation.
- `web-control-plane`: require owner browser actions to remain protected from scripts on a same-site application subdomain.

## Impact

Go configuration and gateway validation, single-host setup and guide, TLS/host tests, and the active Compose self-hosting OpenSpec change. No existing application URLs or credentials are rewritten.
