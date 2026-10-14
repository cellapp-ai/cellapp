# Proposal

## Why

Today every visitor asset passes through Cloud Run, which checks PostgreSQL, reads private object storage, and relays the entire response. A page with many assets multiplies Cloud Run requests and outbound bytes. The current per-file byte quota adds database writes but does not match the desired safeguard against an application spreading rapidly to many people.

## What Changes

- Issue a host-scoped, short-lived grant after one server-side admission check. The Cloudflare Worker validates the grant for subsequent static page and asset requests and serves only the authorized published release from a private R2 bucket. Renew the grant transparently from the existing application session; its validity is at most one minute.
- Keep login, key redemption, control and application data APIs on Cloud Run. Move production artifact delivery to R2 while retaining private S3-compatible storage and local MinIO. Preserve existing GCS objects and a workable rollback path during migration.
- **BREAKING:** Replace the monthly static-response byte limit with a configurable limit of 1,000 distinct, authenticated visitors per application per UTC month. Count each platform user once at first authorized admission, including an application data-only entry; reject new visitors after the cap, while previously admitted visitors and the owner continue. Do not count failed authorization or individual asset requests.
- **BREAKING:** Static-resource revocation, visibility changes, deletion, and operator suspension may take up to one minute to stop an already issued grant. New grants reflect current state. Application data APIs retain their request-time authorization behavior.
- Keep the bucket private and prohibit direct object URLs or unauthenticated edge cache hits. Retain per-source request limiting and operator suspension as separate abuse controls.

## Capabilities

### New Capabilities

- `edge-app-delivery`: Host-bound batched grants, private edge object delivery, renewal, and bounded invalidation.

### Modified Capabilities

- `static-app-lifecycle`: Replace monthly byte exhaustion with monthly distinct-visitor admission and state its behavior at the limit.
- `app-key-access`: Permit a bounded, signed static-resource grant while preserving immediate checks for application data and authentication operations.
- `s3-storage`: Permit authorized edge reads of the private production bucket and require storage migration verification.
- `cloud-run-deployment`: Cover coordinated Cloud Run, Worker, R2, secret, and rollback validation.

## Impact

Go gateway, admission and migration code; PostgreSQL visitor accounting; Cloudflare Worker and private R2 binding; Cloud Run deployment configuration and secrets; public limits response and any client contract that consumes it; storage migration tooling, integration/browser tests, and operations guidance. This change does not introduce anonymous application access or move application data APIs to the edge.
