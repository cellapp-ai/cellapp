## 1. Host and data model

- [x] 1.1 Add additive migration for current and historical host labels, preserving ineligible legacy aliases.
- [x] 1.2 Allocate complete host labels atomically and retain old aliases and hosts on rename or deletion.
- [x] 1.3 Route readable and legacy hosts to immutable app IDs and preserve requested host through login handoff and Origin checks.

## 2. Owner settings and clients

- [x] 2.1 Add browser-session APIs and console forms for user and application slug edits.
- [x] 2.2 Update public CLI/Skill guidance and HTTP contract documentation.

## 3. Verification and delivery

- [x] 3.1 Cover reserved names, short slugs, collisions, old links, host routing and handoff with isolated database tests.
- [x] 3.2 Run console browser, server race, migration and client checks; record any external environment gap.
- [x] 3.3 Sync and archive the OpenSpec change, commit child then parent repository without pushing.
