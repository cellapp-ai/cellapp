# Proposal

## Why

The validation environment pays for a Google HTTPS load balancer even at zero traffic. Cloudflare already manages both DNS zones and can provide the public HTTPS edge, so a direct Cloudflare-to-Cloud Run route can remove that fixed cost after the route is proven safe.

## What Changes

- Add an optional Cloudflare Worker route for the control host and wildcard visitor hosts, forwarding requests to the Cloud Run `run.app` origin.
- Let the Go service accept an original host supplied by that Worker only when authenticated with a dedicated edge secret; retain direct `Host` routing for the current load balancer.
- Add a staged trial, domain cutover, and rollback procedure. Keep the current load balancer until both hosts and browser authorization pass live checks.
- Permit the Cloud Run deployment flow to use the Cloudflare edge instead of requiring the Google load balancer.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cloud-run-deployment`: Accept a trusted third-party HTTPS edge while preserving exact host routing, authorization, and recoverable releases.

## Impact

Go host routing and configuration, Cloud Run deployment configuration and script, Cloudflare Worker and DNS routes, Secret Manager, operations documentation, and browser verification. No change to application URL shape or stored data.
