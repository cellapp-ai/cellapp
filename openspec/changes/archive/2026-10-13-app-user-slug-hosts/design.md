## Context

The service currently stores one `app_namespaces.handle` per owner and one `apps.slug` per application, while serving uploads on `{app-id}.<APPS_DOMAIN>`. App sessions, database schemas and deployments all use the internal ID. A control-origin alias redirects to that host.

## Decisions

1. Keep internal IDs unchanged. Store the current readable host label on each app and all assigned labels in `app_hostnames`. The latter is never recycled, including after rename or deletion. Use a unique key on the complete label because hyphens make its two components ambiguous when parsed.
2. Enforce user slug length 3–61, lowercase ASCII label syntax, reserved names, and a 63-character total label on new or edited values. A static reserved set covers common platform routes. App slugs remain unique in an owner's namespace, and complete host labels are globally unique.
3. Add a nullable current label for compatibility. Migration backfills only valid, non-conflicting existing pairs. Ineligible apps retain the ID host until their slugs are edited; migration changes no existing slug, app ID or deployment record.
4. Keep old ID hosts and historical readable hosts routable to the original app. Retain old control aliases in `app_alias_history`. Deleted apps resolve nowhere and names are not reassigned.
5. Only browser-session console APIs can edit slugs. Transactions lock the owner and update the namespace plus all eligible owned app labels atomically. A collision or a too-long app slug that already has a readable host rolls back the whole edit. Legacy apps without a readable host may keep their ID URL until their own slug is shortened, so an old pair of 63-character slugs is not trapped.
6. Validate every request Host before routing. App write Origin must exactly match that validated request host. The control-to-app login handoff carries a validated host label so the visitor returns to the host they opened; each host keeps its own host-only cookie.

## Risks and Recovery

- A new host has a separate cookie jar from an older host, so an existing visitor may sign in once on the new address. Legacy hosts remain usable.
- Production needs wildcard DNS and TLS for `*.ce11.app`. Roll back the service binary without rolling back migration 007; older binaries continue serving ID hosts, while new readable hosts are temporarily unavailable.
- If a slug write response is lost, the console offers a refresh instead of retrying automatically. Unique database constraints resolve concurrent claims.
