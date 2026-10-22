# Proposal

## Why

Static apps currently combine a direct `/_cellapp/me` fetch with a narrowly compatible `supabase-js` client. There is no single Cellapp browser client or way to pin the runtime API contract, so a later breaking service change could affect already published apps and make failures hard for Agents to diagnose.

## What Changes

- Add an official browser JavaScript SDK for current app user, access/error state, and supported table CRUD, using only the app origin and its visitor session. Keep table schema and policy management in the owner CLI.
- Introduce code-defined application runtime API contracts with equivalent `major.minor` and `yymmdd` aliases (for example, `1.1 = 261022`), selected with `?v=<alias>`. An omitted version uses the latest implementation, including for existing unversioned callers. Unknown or retired aliases fail explicitly.
- Maintain a platform default stable API version. Resolve it before building a new static release, unless the project explicitly selects a supported version; embed the resolved version into the built frontend so reverting to older static files also restores their API selection. Do not persist the version on `deployments` or derive it from the active deployment on each API request.
- Preserve common authentication, admission, origin, and data authorization checks across version handlers; support simultaneous pinned contracts in code.
- Publish an Agent-readable version guide and API changelog covering both equivalent aliases, their one-to-one mapping, selection, supported operations, behavior changes, errors, migration, and support status. Update browser API/data docs, deployment Skill, and runnable example to use the SDK and explicit version selection.
- **BREAKING:** Treat existing requests without `v` as latest rather than promising an immutable legacy contract. This is accepted for the current single-user stage; new SDK-built apps always pin a version.

## Capabilities

### New Capabilities

- `application-api-versioning`: Version selection, stable default, concurrent server contracts, release-bound client selection, and Agent-readable version history.
- `application-js-sdk`: Official browser client for Cellapp identity, access states, and authorized table operations.

### Modified Capabilities

- `application-identity`: Expose current app user behavior under a selected runtime API version while retaining the existing identity and privacy boundary.
- `application-data`: Move the public table endpoint to `/_cellapp/db/{table}` and select its runtime contract alongside application identity. The SDK may continue using Supabase's table query builder internally.
- `skill-deployment`: Guide Agents to select and embed an API version, use the SDK, and consult the version guide and changelog.

## Impact

- Go app gateway and PostgREST proxy routing, runtime version registry, and operator configuration for the default stable version.
- Public `skills` submodule: a browser SDK package, CLI build/deploy integration, example, contracts, and English/Chinese Skill guidance.
- Main repository docs and public Agent documentation; tests for version routing, security boundaries, build-time pinning, static rollback, and unsupported versions.
- No application database migration or new `deployments` column is planned. SDK and HTTP contract versions remain separate from the version of `cellapp.data.json` and the public `/_cellapp/db` route.
