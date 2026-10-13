# Tasks

## 1. Authenticated origin host

- [x] 1.1 Add optional edge-origin configuration and authenticated host selection in the Go server; verify control, visitor, forged-header, unknown-host, and existing direct-Host cases with focused tests.
- [x] 1.2 Add a Cloudflare Worker that validates incoming hosts and forwards requests to the configured HTTPS `run.app` origin with a dedicated secret; verify methods, paths, bodies, redirects, cookies, and client header stripping with Worker tests.

## 2. Repeatable deployment

- [x] 2.1 Extend Cloud Run configuration and release script with an explicit Cloudflare edge mode and Secret Manager binding; verify preflight, ingress selection, and domain-check failure paths with script tests.
- [x] 2.2 Document Cloudflare setup, dedicated secret handling, staged cutover, pricing limits, and rollback; validate example commands and links.

## 3. Live trial

- [x] 3.1 Deploy an isolated Cloudflare Worker probe and Cloud Run service while preserving the Google load balancer; verify authenticated origin success and direct-origin denial over public HTTPS.
- [x] 3.2 Route the control and wildcard domains through Cloudflare; verify HTTPS, original-host routing, unknown-host denial, and origin response headers.
- [x] 3.3 Record browser GitHub login through the Cloudflare trial, production origin and asset checks, the tested DNS rollback before load-balancer deletion, resource removal, and remaining visitor-app limitation (no published application available).
