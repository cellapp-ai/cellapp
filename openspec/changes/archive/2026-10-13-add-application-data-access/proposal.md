# Proposal

## Why

Cellapp currently hosts static frontends but has no application data service. Developers must supply their own database and user system to build persistent applications. This change makes Cellapp responsible for application data and platform identity while preserving static-only deployment and strong separation from platform control data.

## What Changes

- Add a PostgreSQL application-data database separate from the control database, with one isolated schema and restricted database privileges per application. Provide owner-managed, versioned table definitions and access policies through the Cellapp CLI and Skill.
- Add an authenticated, application-scoped data API compatible with the `supabase-js` table CRUD path (`from(...).select/insert/update/delete`), with explicit supported behavior and denial for unsupported Supabase products. Applications receive no direct database credentials or custom server runtime.
- Add owner-scoped public application references `@namespace/alias` and a share-link redirect to stable app-ID hosts; keep display names free to repeat and old links valid.
- Let users sign in through Cellapp before entering any application. A public application admits any signed-in user; a restricted application requires an existing grant, an approved access request, or redemption of its access key. Expose a stable per-application user ID without disclosing a cross-application identifier.
- **BREAKING** Replace anonymous share-key browser sessions with user-bound grants. Existing application keys remain redeemable after sign-in; existing key-only sessions no longer authorize access. Rotating a key stops future redemption but preserves existing grants, which owners revoke separately.
- Add owner controls for visibility, access requests, grants, key rotation, table management, and data policy management. Update the published Skill, CLI contracts, and documentation to operate and explain these capabilities.
- Keep payment and automatic purchase grants out of this stage; model grants so a later purchase flow can issue them without changing application authorization.

## Capabilities

### New Capabilities

- `application-data`: Application database provisioning, schema and role isolation, owner-managed table migrations and policies, and Supabase-compatible table CRUD.
- `application-identity`: Platform login for application visitors, application-scoped identity, and current-user access for static frontends.
- `application-namespace`: Owner-scoped app aliases and share links that resolve to stable app-ID hosts.

### Modified Capabilities

- `app-key-access`: Replace key-only access with signed-in public/restricted application admission and durable user grants.
- `web-control-plane`: Let owners manage application visibility, requests, grants, keys, and data settings through authorized browser controls.
- `skill-deployment`: Teach the Skill and CLI to create and manage data tables and policies while continuing to deploy static frontends only.
- `static-app-lifecycle`: Include application data access shutdown and eventual data cleanup when an application is deleted.

## Impact

The Go control service, gateway, identity and migration paths, a separate PostgreSQL database and restricted roles, the web console, the public CLI/shared packages and HTTP contracts in the `skills` submodule, the deployment Skill, local and production configuration, tests, and operations documentation are affected. The existing control database, private object storage, and static build execution boundary remain separate. Another in-progress worktree plans a Supabase-backed control environment; this change requires application data to remain a distinct logical database regardless of the control database provider.
