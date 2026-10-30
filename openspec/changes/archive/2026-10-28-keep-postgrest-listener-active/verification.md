# Release verification

Date: 2026-10-28 UTC.

- Source: main `ab08dadd2cc727a80a73bf10200bcd81eff0e97e`; deploy/main merge `f04f3a278c1d56d783d9955c761c6d2fe3d9093d`.
- Cloud Build succeeded; image digest `sha256:21a026296daa24c22c6e09f5b601b5194b6b7523604fc3c73dd218af788d2b71`.
- Migration execution `cellapp-migrate-lvvgw` completed successfully. Revision `cellapp-00028-kbl` serves 100% traffic. Service reports `run.googleapis.com/cpu-throttling=false` and service-level `run.googleapis.com/minScale=1`.
- Official `@cellapp/cli@latest` created operator-owned app `84073b5d5e0ade75ecc623a3021cb352` and published its first static release. Data status was `ready`, version 0, with no error. `data plan`, `data apply`, and `data verify` created the `notes` table and reported ready at version 1. The app was deleted afterward; it no longer appears in the owner app list.
- Revision logs for the acceptance app showed `Received a schema cache reload message on the "pgrst" channel`, `Config reloaded`, and a newly loaded schema cache. No `app_data_reconcile_failed` appeared in the queried revision logs.
- Public route checks: `https://cellapp.ai/dashboard/login` returned 200 with request ID; unknown app host returned 404; unauthenticated VisOKR host returned 401.
- The original VisOKR app was already `data_state=ready` before this release and remains without an active deployment. This release does not publish the owner's project files.
- Formal local test suites and interactive browser OAuth/visitor sessions were not run in this release. The production CLI flow, data API, Cloud Run revision, and route checks above were the acceptance evidence.
