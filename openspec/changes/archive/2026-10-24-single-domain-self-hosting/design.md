# Design

## Context

See [proposal.md](proposal.md). The dashboard already uses `/dashboard`, but production configuration and self-host setup reject a shared registrable domain. The Go gateway distinguishes the exact control Host from application hosts by wildcard suffix. Browser owner sessions use `__Host-` cookies; browser writes already require an exact control Origin. The still-active Compose self-hosting delta describes the old two-domain rule.

## Goals / Non-Goals

**Goals:** Allow a self-hosted operator to purchase one domain and select either `example.com` + `*.example.com` or `example.com` + `*.apps.example.com`, while preserving existing separate-domain and sibling-host deployments.

**Non-Goals:** Serve uploaded applications under a path of the control origin, provide custom application domains, alter public production edge routing, or automate DNS, certificates, and GitHub OAuth in this change.

## Decisions

1. **Use exact control Host before wildcard app routing.** Keep the existing gateway order. Permit the control hostname to equal the application base or be a parent/sibling, but reject a control hostname that is a child of the app base. Retain production public-suffix validation; remove only the equality rejection of the two registrable domains. This avoids rewriting application addresses or relaxing unknown-host handling. A path-based app mount was rejected because uploaded JavaScript would share the control origin.
2. **Match setup preflight to server rules.** Let self-host setup accept an equal base, a nested app base, or non-overlapping siblings. Keep certificate checks for the exact control host and a sample wildcard app host. The Go server remains authoritative for public-suffix validation. Existing secrets, certificates and volume identities remain stable when setup is repeated.
3. **Protect the same-site cross-origin boundary.** Preserve host-only `__Host-owner` and `__Host-cellapp` cookies, exact-Origin checks for all browser-authorized mutations, no cross-origin read permission on private control responses, and Bearer-only CLI APIs. The two hosts are same-site for browser cookie policy, so tests must exercise app-origin writes with owner cookies and missing Origin rather than relying on SameSite alone. App-side writes still require the exact app Origin.
4. **Coordinate active planning.** Revise the unarchived Compose self-hosting delta's production-domain requirement and examples so later sync/archive cannot restore the obsolete two-domain restriction. Existing official root-dashboard cutover work remains separate; this change concerns configurable self-hosting and the generic Go host boundary.

## Risks / Trade-offs

- [Same-site subdomain sends owner cookies to control requests] → Require exact control Origin for cookie-authorized writes, retain host-only cookies, and verify no private response is made readable across origins.
- [A wildcard accidentally claims the control host] → Route the exact control host first; reject a control hostname below the app base and test unknown hosts.
- [Wildcard certificate or DNS does not cover the chosen layout] → Document and preflight the exact host and `probe.<apps-domain>` certificate names for both layouts.
- [Old planning reintroduces two domains] → Update and strictly validate the active Compose delta alongside this change.

## Migration Plan

Existing two-domain installations keep their URLs and configuration. New single-domain installations use the documented setup arguments and matching wildcard DNS/certificate. Existing installations can change their configured application host base only after arranging DNS/certificates and preserving old links or a separate redirect plan; this change does not rewrite published hostnames or database records.
