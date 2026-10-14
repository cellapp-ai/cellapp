## 1. Client release and update check

- [x] 1.1 Bump public package, workspace and lockfile versions to 0.2.1, with a consistency check.
- [x] 1.2 Add daily, bounded npm update check at CLI startup; preserve output, privacy and offline behavior.
- [x] 1.3 Document the behavior and upgrade command for users.

## 2. Verification

- [x] 2.1 Test version comparison, daily caching, newer/equal/malformed responses, failures and CLI output.
- [x] 2.2 Run required client checks, package dry-run and review the release contents.
