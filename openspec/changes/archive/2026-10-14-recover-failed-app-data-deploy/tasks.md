# Tasks

## 1. Control API recovery

- [x] 1.1 Return the committed app ID on provisioning failure and log the underlying error; verify Go hosting tests pass.
- [x] 1.2 Add owner-authorized data reconciliation and status error code; verify Go hosting tests pass and the HTTP contract documents both responses.
- [x] 1.3 Make the fresh-volume PostgreSQL initialization hook executable; verify the isolated Compose database and PostgREST start healthy.

## 2. CLI and Skill recovery

- [x] 2.1 Preserve creation request and initial key across failure, associate a committed app, and retry its provisioning; verify the interrupted first-deploy contract test and private key permissions.
- [x] 2.2 Support linking an older orphaned app and an explicit data reconcile command; verify TypeScript typecheck, CLI tests, and Skill validation.
- [x] 2.3 Document recovery commands and expected failure behavior in the Skill and operations guide; verify referenced commands exist.

## 3. Integration verification

- [x] 3.1 Run isolated PostgreSQL and PostgREST acceptance for a failed-then-recovered first deployment and the full race suite serially; record results before archive.

## Workflow follow-up

- Sync the delta specifications and archive after integration acceptance.
- Review and commit the skills repository first, then update and commit the parent gitlink when its commit is remotely available.
