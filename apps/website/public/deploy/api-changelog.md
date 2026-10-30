# Cellapp application API changelog

[Version guide](./api-versions.md) · [中文版本说明](./api-versions.zh-CN.md)

Each entry names one fixed runtime contract and its two equivalent aliases. The server registry and this list must change together. Status and retirement dates are contractual support information; a missing retirement date means none is scheduled.

## 1.2 = 261027

- **Status:** supported; default stable; latest.
- **Declared:** 2026-10-27. **Retirement:** not scheduled.
- **Added:** same-origin `GET /_cellapp/v1/models` and `POST /_cellapp/v1/chat/completions`. A signed-in, admitted user authorizes provider connections per application. The model list combines authorized providers, and each call checks the selected model's connection grant. Requests have concurrency, time and output limits. Only the documented text, SSE and function-call subset is implemented.
- **Errors:** missing model authorization, unsupported capability, provider failure and interrupted streams have distinct codes. Token usage remains unknown if the provider does not report it.
- **Compatibility:** existing `1.1` pages keep their identity and table behavior and do not receive model access. Rebuild and redeploy the static app with the `1.2` contract and a compatible SDK to opt in.

## 1.1 = 261022

- **Status:** supported.
- **Declared:** 2026-10-22. **Retirement:** not scheduled.
- **Endpoints:** `GET /_cellapp/me` returns `{id,access}` for an admitted app user. Supported `/_cellapp/db/{table}` operations: select, insert, update, delete, equality filter, order, range, and write followed by select. Existing app and table policy checks apply on every request.
- **Errors:** `401 login_required` for missing app login; `403 access_required`, `access_pending`, or `access_denied` for visitor admission; data readiness and policy errors retain their documented codes. An invalid or unsupported version selector returns `400 unsupported_api_version`.
- **Change from direct calls:** SDK requests include `?v=1.1`; omitting `v` uses latest. The selected version is embedded in published HTML and follows static rollback.
- **Migration:** adopt the official Cellapp browser SDK and rebuild/redeploy the static app. Direct callers may append `?v=1.1` or the equivalent `?v=261022` to pin this contract. Table definitions and access settings do not migrate automatically.
