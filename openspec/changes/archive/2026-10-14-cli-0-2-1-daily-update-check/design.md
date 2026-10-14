# Design

## Context

The CLI currently starts in `packages/cli/src/main.ts`, writes credentials under the per-user Cellapp configuration directory, and prints command results as JSON on stdout. Version `0.2.0` predates the latest source changes.

## Decisions

- Store the last attempted check date in a separate private file in the CLI configuration directory. The date uses the machine's local calendar day. Record the attempt before the request so an offline machine does not retry on every command that day.
- Query only `https://registry.npmjs.org/@cellapp%2Fcli/latest`, with no authentication header, redirect following, or project fields and a short timeout. Accept only a well-formed stable `major.minor.patch` version; compare numeric components.
- Print a concise upgrade command to stderr only when the registry version is newer. Failures in reading or writing update state or contacting npm do not fail the requested CLI command. Keep stdout JSON unchanged.
- Keep a single local version constant and verify in tests that it matches all package versions. Avoid adding a semver or update-check dependency for this narrow comparison.

## Risks / Trade-offs

- The first command of a day can wait briefly for npm. Bound this to two seconds; later commands that day do not wait.
- Simultaneous CLI processes could both check before either records the date. This is harmless duplicate read-only traffic; sequential invocations make one attempt per day.

## Validation

Test new, equal, malformed, offline, same-day, next-day, stdout/stderr, and package-version cases. Run client typecheck, tests, build, public boundary, Skill and package checks, and pack dry-run.
