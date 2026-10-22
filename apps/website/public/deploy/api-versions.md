# Cellapp application API versions

[简体中文](./api-versions.zh-CN.md) · [Version changelog](./api-changelog.md)

Cellapp's application runtime API covers `/_cellapp/me` and the supported table operations under `/_cellapp/db/{table}`. It is separate from the SDK package version, the `/_cellapp/db` route, and the `cellapp.data.json` table-manifest version.

## Selecting a contract

A fixed contract has two equivalent aliases: a semantic `major.minor` identifier and a `yymmdd` date alias. For example, `1.1 = 261022`:

```text
GET /_cellapp/me?v=1.1
GET /_cellapp/me?v=261022
```

Both requests use the same handler, response and error contract. One date alias identifies only one contract. `major` changes for incompatible semantics; `minor` identifies an additive contract. Compatible fixes can continue to serve the same contract without creating a new alias.

Omitting `v` calls the latest server implementation, including for older direct callers. The platform's **default stable** contract can lag behind latest. New SDK builds without an explicit version select the default stable contract once and embed its `major.minor` identifier in every output HTML page. A later change to the default does not alter an existing build. Reverting to older static files restores their embedded selection; a page already open keeps the version it loaded with.

The SDK sends `v` on its identity and table requests. An unsupported, malformed, duplicate, or retired `v` returns `400 unsupported_api_version` with the requested alias and a link to this guide. It never silently falls back to latest. The selected contract controls public API behavior, not login, app admission, table policies, or row security; these are checked on every request.

To diagnose an error, inspect `<meta name="cellapp-api-version">` in the served HTML, then compare that identifier and the error code with the [changelog](./api-changelog.md). Do not infer the runtime API version from an SDK npm version or a static deployment ID. App owners continue to manage tables and policies with the Cellapp CLI; browser code cannot use owner control APIs.
