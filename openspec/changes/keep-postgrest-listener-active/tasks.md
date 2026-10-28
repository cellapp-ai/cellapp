# Tasks

## 1. Keep the listener active

- [x] 1.1 Set Cloud Run service CPU to always allocated and service-level minimum instances to one; verify the deploy command uses both flags and leaves jobs unchanged.
- [x] 1.2 Document the runtime behavior and cost; verify the operations guide matches the deploy script.

## 2. Publish and verify

- [ ] 2.1 Merge the focused change into local main and deploy/main, deploy a candidate through the standard release script, and verify the new revision receives all traffic with the intended CPU/minimum settings.
- [ ] 2.2 Create an operator-owned app and data table, verify the first data API probe and schema operation succeed, then clean up the test app; restore previous traffic if acceptance fails.
