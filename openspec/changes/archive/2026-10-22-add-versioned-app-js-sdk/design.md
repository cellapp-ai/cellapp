# Design

## Context

See [proposal.md](proposal.md) for the motivation and the capability deltas for behavior. The Go app gateway currently routes `/_cellapp/me` directly and proxies `/rest/v1/{table}` to PostgREST after per-request admission checks. The public table route will become `/_cellapp/db/{table}`. The notes example calls `/_cellapp/me` directly and uses `supabase-js@2.117.3`. The CLI builds locally, enumerates and hashes output files, then prepares/uploads/publishes a static deployment; `deployments` has no API version field. The public deployment runbook is served as static Markdown from `apps/website/public/deploy/`.

## Goals / Non-Goals

**Goals:**
- Keep the selected API contract inside the static frontend, including when an older release is restored, while keeping a configurable platform default for new builds.
- Separate public contract adapters from common authentication, admission, origin, and database policy enforcement.
- Give Agents a single public version guide and changelog aligned with the server's supported versions.

**Non-Goals:**
- Add a new login provider, expose email/profile information, or implement the unsupported Supabase product APIs.
- Store the chosen API version in `deployments`, derive it from `active_deployment` on each request, or let visitor code manage table schemas.
- Preserve a fixed historical contract for existing unversioned callers; they intentionally follow latest.

## Decisions

### 1. Code-defined contract registry and request selector

Add a Go registry of immutable contracts, each with one canonical `major.minor` identifier and one `yymmdd` alias. Both aliases resolve to the same handler and changelog entry; reject duplicate semantic identifiers or date aliases at startup. The registry designates `latest` and validates the operator-configured `stable` contract against its supported entries. The baseline pair is `1.1 = 261022`; if implementation lands after that date, keep this agreed initial mapping rather than silently renaming it. `major` advances for incompatible semantics, `minor` for additive capability, and the date identifies the contract's declared date. A date alias can identify only one new contract on a given day; compatible fixes continue to serve an existing contract without a new alias. Reject malformed calendar dates, duplicate `v`, unknown aliases, and retired aliases with a stable `unsupported_api_version` error. The error includes the requested alias and a public version-guide URL, not internal configuration. Omitted `v` selects latest; latest and stable are independent pointers.

Parse and remove `v` at the app gateway boundary before forwarding table query parameters to PostgREST. Serve table requests at `/_cellapp/db/{table}` alongside all other active app-host built-in APIs. The SDK's Supabase table builder internally constructs `/rest/v1/{table}` requests; its same-origin fetch adapter rewrites only that table path to `/_cellapp/db/{table}` before network transmission. Keep the present app-host resolution and access checks shared. Version-specific functions adapt the public request/response surface only; they cannot select a different app or bypass permission checks. The first version can delegate to existing behavior, making the selection layer concrete before future forks.

Alternative: clone the entire gateway per version. That would duplicate security-sensitive logic. Alternative: encode the API version in a URL prefix. The chosen query selector matches the requested public form while the table route stays in the `/_cellapp` namespace. Old `/rest/v1` requests are rejected and reserved from static fallback; this unreleased change does not retain a second public table endpoint. Control-domain `/auth/*`, internal `/_hosting/*`, and unsupported Supabase compatibility probes are outside the active app API namespace.

The app-host route audit found `/_cellapp/handoff`, `/_cellapp/me`, `/_cellapp/edge-grant`, `/_cellapp/request`, and `/_cellapp/redeem` already grouped under the platform prefix. Handoff, access request, and redemption are browser flows; edge grant is an internal Worker request. The table route is the only active app-host API moving from another prefix. Control-domain `/auth/*` handles login and CLI credential flows and is not an app-host API. `/_hosting/*` remains an internal edge transport; unsupported Supabase product paths remain explicit failure probes rather than public services.

### 2. Resolve stable version before build; embed it in output HTML

Expose a small authenticated control API read for the current stable contract and supported alias pairs; the CLI uses it before invoking the optional local build command. An explicit `apiVersion` project setting or `--api-version` override may use either alias, but the CLI resolves it to the canonical `major.minor` identifier and verifies support. Keep this explicit preference separate from the resolved build version so a later new build can adopt a new default. If version discovery or validation fails, SDK-enabled publishing fails instead of silently selecting latest.

After the local build and before `collect` computes hashes, the CLI inserts or updates one `<meta name="cellapp-api-version" content="1.1">` (using the selected canonical identifier) in every published HTML entrypoint. The CLI validates a pre-existing marker and rejects conflicting or malformed values. This also works for a prebuilt output directory and for plain HTML projects. The published files, not the deployment row, own the selection. The SDK reads the marker once at initialization and keeps that value for the page's lifetime; it fails clearly if absent or malformed. The CLI reports both equivalent aliases and the embedded canonical identifier. Raw apps without the SDK can continue using unversioned endpoints and latest behavior.

Alternative: mutable app settings or request-time lookup through `active_deployment`. Those make an already open page change contract at publish time. Alternative: a separately fetched runtime config file. That can race with a release switch between loading old JS and fetching new config. The HTML marker is part of the page loaded with the app and is easy to inspect when debugging.

### 3. Official browser SDK boundary

Add an ESM browser package in `skills/packages/` with a small Cellapp API: initialize from the page marker, read the current app user and typed access errors, and create a table client for the declared CRUD subset. The table implementation may wrap the pinned `supabase-js` compatibility layer with a custom same-origin fetch that appends `v` while preserving existing table query parameters and credentials. Pin/test the supported Supabase client version rather than implying full compatibility. Keep SDK package SemVer independent of the runtime API contract ID. Include both bundled-app and plain static HTML examples; neither contains secrets.

Alternative: ask each app to compose a raw `fetch` and Supabase client. That leaves version propagation and error mapping duplicated in generated apps.

### 4. One public Agent-facing version record

Place a version guide and chronological changelog under `apps/website/public/deploy/`, reachable without login from the existing English and Chinese deployment runbooks and packaged Skill. The guide defines the selector, default stable versus latest, canonical `major.minor` versus date alias `yymmdd`, the one-to-one constraint, static rollback, supported operations and error handling. Each changelog entry identifies both aliases, status, affected endpoints, response/error differences, migration steps, and support/retirement dates. Update `docs/web-api.md`, `docs/app-data.md`, the runnable notes demo, and the relevant public HTTP contract with links and examples. A repository check compares both aliases in the registry with changelog entries and checks guide links, so a later code-only version cannot silently omit Agent guidance.

Alternative: duplicate full changelogs in root docs, Skill, and website. A single public record with links avoids drift.

## Risks / Trade-offs

- **HTML marker and bundled code differ** → SDK reads the marker once from the page; CLI rewrites before hashing and validates every HTML entrypoint. Browser acceptance covers multi-page and SPA output, and a release switch while a page remains open.
- **Old pinned contract retained longer than planned** → Record support status and retirement date in the changelog; reject retired versions explicitly. Do not remove an old adapter before affected published apps are migrated or deliberately accepted as broken.
- **A new version accidentally weakens access controls** → Centralize admission and data proxy security checks; test old and new versions against revocation, cross-app, anonymous, Origin, and forged-header cases.
- **`v` reaches PostgREST as a filter** → Strip it before proxying and test combinations with select, eq, order, and range.
- **Default stable changes during a build** → Resolve once before build and use that single value for every HTML file and the reported result.
- **Two contract releases on one calendar day need the same date alias** → Allow one new fixed contract per `yymmdd`; apply compatible fixes to its existing contract or schedule a distinct contract for another date.
- **Public documentation deploys separately from Go and npm** → Publish/version docs as part of the rollout sequence and verify the public URLs before marking the version stable.

## Migration Plan

1. Add the version selector and registry, initially mapping the baseline version to current app identity and data behavior. Leave omitted `v` mapped to latest, including the current notes example until updated.
2. Publish the public guide/changelog and control API stable-version read. Deploy compatible server and edge routes before shipping a client that requests `v`.
3. Add the SDK and CLI output embedding. Update the demo and Agent guides, then publish the compatible Skill/CLI package after server verification.
4. Promote the validated contract to default stable. Existing unversioned callers continue to get latest; previously built SDK apps keep their embedded version. Restoring older static files restores their marker and SDK request version without a database migration.
