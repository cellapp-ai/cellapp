# Fix application access through the authenticated edge

## Why

An application published at a readable visitor hostname redirects to `/auth/app` with the Cloud Run origin hostname in the `host` parameter. The authorization endpoint correctly rejects that hostname because it was never assigned to the application, leaving the published application inaccessible. The same origin checks for visitor writes also compare the browser Origin against the internal Cloud Run Host.

## What Changes

- Use the already authenticated browser hostname for visitor handoff redirects and visitor write origin checks.
- Keep the existing direct Host and edge proof rules; do not infer identity from unauthenticated forwarding headers.
- Cover the full browser handoff and same origin behavior in an edge routed regression test.

## Impact

Only server visitor routing and authorization behavior changes. The CLI, deployment protocol, database schema, and stored application host assignments stay the same.
