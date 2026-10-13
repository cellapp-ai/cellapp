## Why

Application links currently expose a random internal ID even though owners can choose readable namespace and application aliases. Owners need a direct readable host and a way to change both slugs from the console.

## What Changes

- New applications use `{app-slug}-{user-slug}.<APPS_DOMAIN>` as their public URL. The app ID stays immutable for authorization, data and deployment identity.
- A user slug is at least three characters. Both slugs reject platform names such as `accounts`, `cloud`, `login` and `app`; the full host label must fit 63 characters and be globally unique.
- The console lets an owner edit their user slug and each application slug. Old ID hosts, previously assigned readable hosts and control-domain aliases remain bound to the original application.
- Eligible existing aliases get readable hosts during migration. Long, reserved or ambiguous historical aliases keep their ID hosts until edited.

## Capabilities

### Modified

- `application-namespace`: slug validation, owner editing, host routing and migration.
- `web-control-plane`: profile and application address settings.

## Impact

Control database migration 007, HTTP hostname router, app login handoff and same-origin checks, console API/UI, public CLI guidance and contract. Production DNS and TLS must include `*.ce11.app` with `APPS_DOMAIN=ce11.app`.
