# Proposal

## Why

Cellapp currently requires a platform login even for public applications, so a public form cannot be opened and submitted by an anonymous visitor. Owners need explicit control over anonymous reading and writing, while operators need a platform-wide switch that can stop anonymous submissions immediately.

## What Changes

- Add owner-managed, per-application anonymous page access and separate anonymous data read/write settings. All three default to off for existing and new applications; a restricted application cannot enable them.
- Add an operator-controlled, platform-wide anonymous write switch, default off. Anonymous writes require both this switch and the application's write setting; turning either off blocks subsequent writes without changing stored rows.
- Extend versioned table policies with explicit anonymous read and insert permissions. Application-level access does not by itself grant access to a table. Anonymous update and delete remain unavailable.
- Let anonymous visitors open enabled public applications and their static assets, including through the production edge, while keeping private storage, host isolation, short grant lifetime and immediate data authorization checks.
- Apply bounded anonymous request limits and document the limits of identity-free visitor accounting. Expose the effective settings and consequences in the owner console, CLI/Skill and operations guidance.
- **BREAKING** Amend the existing rule that all applications require a Cellapp login, but only for applications whose owner explicitly enables anonymous page access.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `app-key-access`: Allow explicitly enabled public applications to serve pages and assets without login, while keeping restricted applications and signed-in grants protected.
- `application-data`: Define the operator gate, application and table-level anonymous read/insert authorization, and request limits.
- `edge-app-delivery`: Bind short static grants to the anonymous admission mode without exposing private objects or weakening signed-in grants.
- `static-app-lifecycle`: Define anonymous traffic accounting and rate limits alongside the existing signed-in monthly visitor quota.
- `web-control-plane`: Let owners view and manage per-application anonymous settings with clear effective status.
- `skill-deployment`: Document and support the new policy modes and anonymous client behavior in the CLI and Agent guidance.

## Impact

The Go configuration, app gateway, data proxy and admission logic; control database migration and browser API; PostgREST role/policy provisioning; edge Worker grant flow; owner dashboard; public CLI/shared contracts and deployment Skill; environment and operations documentation; and isolated PostgreSQL, browser and edge acceptance tests. Existing applications and table manifests remain private until their owners opt in. No public database endpoint or storage bucket is introduced.
