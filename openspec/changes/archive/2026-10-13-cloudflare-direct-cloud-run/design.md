# Design

## Context

The Go service already routes control and application traffic by exact `r.Host` and intentionally ignores forwarding headers. Caddy passes Host through. Cloud Run currently accepts only Google load balancer ingress, and both public DNS names resolve to its global forwarding rule. Cloudflare Free cannot override the origin Host with Origin Rules; a Worker can fetch the `run.app` URL instead.

## Goals / Non-Goals

**Goals:** Keep control and visitor URLs unchanged, authenticate the browser hostname at the Go boundary, test the new path independently first, and retain a direct rollback to the current load balancer.

**Non-Goals:** Change application identity, cookies, OAuth callback, database, storage, or DNS for unrelated domains. Cloudflare Origin CA certificates are not needed when the Worker fetches a publicly trusted `run.app` HTTPS origin.

## Decisions

1. **Worker as the only host assertion source.** The Worker validates the incoming hostname, replaces any client-supplied edge headers, forwards the original method, path, query, body, and ordinary headers to the exact configured `run.app` URL, and sets a dedicated secret header plus original-host header. The Go service accepts the asserted host only when the request Host is the configured `run.app` origin and the secret matches in constant time. Unknown hosts stay rejected. A separate secret is used rather than the platform signing key so Cloudflare never receives application cryptographic material. A bare wildcard CNAME cannot supply this authenticated host assertion; Cloudflare Free Origin Rules cannot override Host.
2. **Keep the current path during migration.** The normal `r.Host` path remains unchanged for Google load balancer traffic. Cloud Run ingress is switched to `all` only after an image supporting the edge assertion is deployed and its secret is present. Direct `run.app` requests without the edge proof remain unable to reach control or visitor routes. The health endpoint may be reachable on the origin; it exposes no session or application data.
3. **Separate probe before DNS cutover.** Deploy the Worker without production routes and test through its `workers.dev` hostname with a fixed allowed probe host. The user confirmed there is no published traffic and authorized service downtime, so the control and wildcard DNS can then be orange-clouded for live domain testing without a temporary public subdomain. Keep the Google load balancer until the Cloudflare path works, then use proxied placeholder DNS records and remove the load balancer. Before deletion, rollback can restore gray-cloud DNS to `34.120.142.70`. After deletion, rollback uses an earlier Cloud Run revision that supports the edge proof, while the Worker and DNS remain in place. The separate probe service and Worker are removed after the production path is configured.
4. **Explicit edge mode in deployment config.** The deployment script keeps Google load balancer as its existing default and opts into Cloudflare mode only with an origin hostname and Secret Manager reference. This prevents a future release from silently restoring the old ingress or omitting the edge secret. The release probe checks the configured public routes before reporting success.

## Risks / Trade-offs

- [Cloud Run becomes reachable at `run.app`] → Require the dedicated secret for host routing; test forged/missing edge headers and keep existing application authorization checks. Rotate the secret if exposed.
- [Cloudflare Worker or DNS misroutes traffic] → Probe on a separate hostname first and retain the old load balancer during cutover. After removing it, retain the Worker, DNS, secret, and a known good Cloud Run revision as the rollback path.
- [Browser behavior differs at the proxy] → Verify redirects, `__Host-` cookies, OAuth callback, uploads, and an authorized visitor route in a real browser before removing the load balancer.
- [Traffic cost shifts to Cloudflare/Cloud Run] → Measure request volume and Worker free-tier limits before cutover; do not claim a zero-cost public edge.

## Migration Plan

1. Implement and locally test authenticated host selection and Worker request forwarding.
2. Create a dedicated edge secret in Secret Manager and a matching Cloudflare Worker secret without printing either value; deploy a Cloud Run revision with both old Host path and the new proof path.
3. Test a `workers.dev` probe; run negative host and direct-origin checks.
4. Switch the control and wildcard DNS records and attach Worker routes. Complete live control, OAuth, and visitor checks.
5. After live checks and user authorization, remove the Google forwarding rule, proxy, URL map, backend, NEG, public IP, and certificate map. Keep managed certificate objects for possible future use. Roll back by switching to a known good edge-compatible Cloud Run revision if a later release fails.
