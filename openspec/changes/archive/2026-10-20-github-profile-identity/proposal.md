# GitHub profile identity in the console

## Why

The signed-in console currently identifies a GitHub user by an internal account ID, and a new owner's default namespace is random. This makes the account area and first application address feel disconnected from the account used to sign in.

## What Changes

- Show the verified GitHub username and avatar in the signed-in console.
- Use the GitHub username as the default visible name until the owner chooses a display name.
- Prefer a valid form of the GitHub username for a new owner's first user slug, with a safe unique fallback.
- Preserve existing slugs, custom display names, account IDs, and application addresses.

## Capabilities

### Modified Capabilities

- `web-control-plane`: signed-in identity and profile defaults.
- `application-namespace`: initial GitHub-based user slug.

## Impact

GitHub OAuth user parsing, owner metadata and an additive control migration, owner-session/profile JSON, the dashboard account area, CSP for GitHub avatars, documentation and browser/server acceptance checks. No CLI contract or existing address migration.
