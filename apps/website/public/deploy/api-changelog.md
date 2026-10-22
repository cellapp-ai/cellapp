# Cellapp application API changelog

[Version guide](./api-versions.md) · [中文版本说明](./api-versions.zh-CN.md)

Each entry names one fixed runtime contract and its two equivalent aliases. The server registry and this list must change together. Status and retirement dates are contractual support information; a missing retirement date means none is scheduled.

## 1.1 = 261022

- **Status:** supported; default stable; latest.
- **Declared:** 2026-10-22. **Retirement:** not scheduled.
- **Endpoints:** `GET /_cellapp/me` returns `{id,access}` for an admitted app user. Supported `/_cellapp/db/{table}` operations: select, insert, update, delete, equality filter, order, range, and write followed by select. Existing app and table policy checks apply on every request.
- **Errors:** `401 login_required` for missing app login; `403 access_required`, `access_pending`, or `access_denied` for visitor admission; data readiness and policy errors retain their documented codes. An invalid or unsupported version selector returns `400 unsupported_api_version`.
- **Change from direct calls:** SDK requests include `?v=1.1`; omitting `v` uses latest. The selected version is embedded in published HTML and follows static rollback.
- **Migration:** adopt the official Cellapp browser SDK and rebuild/redeploy the static app. Direct callers may append `?v=1.1` or the equivalent `?v=261022` to pin this contract. Table definitions and access settings do not migrate automatically.
