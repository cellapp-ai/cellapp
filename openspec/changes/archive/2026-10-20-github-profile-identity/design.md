# Design

## Identity and defaults

GitHub's numeric user ID remains the immutable authentication subject. The OAuth user response also supplies a login; validate it before storage. Persist the latest verified login on each GitHub sign-in. An empty user-chosen display name means the current GitHub login is the display default; an explicit display name takes precedence. Existing sessions without a stored login keep a neutral account-ID fallback until their next sign-in.

Use the numeric GitHub subject to construct a fixed `avatars.githubusercontent.com/u/<id>` image URL. Permit that host in the console image CSP only. Do not accept an arbitrary avatar URL from the client or GitHub response. A failed image loads a local initials fallback.

## Namespace

The first app creation continues to accept an explicit namespace from existing clients. If omitted, use a lowercase GitHub login when it satisfies user-slug rules. Prefix an invalid short, numeric-leading, or reserved login with `u-`; if occupied, try a stable `u-<numeric GitHub ID>` fallback, then the existing random namespace. Locking and unique constraints preserve concurrency safety. Once assigned, the namespace never changes on a later GitHub rename; an owner can edit it explicitly in Profile.

## Compatibility and verification

Add a nullable/default-empty owner metadata column in a new migration. Extend the browser session and profile responses additively. Preserve the profile update contract, account IDs, session checks, CLI Bearer behavior, and all current app hosts. Verify OAuth validation and refresh, defaults and conflicts, custom overrides, owner isolation, avatar fallback, responsive layout, and CSP. No production data rows or secrets are introduced.
