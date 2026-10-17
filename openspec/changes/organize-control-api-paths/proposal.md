# Proposal

## Why

Cellapp 的 CLI/Agent 控制接口目前散落在控制域根路径（`/apps`、`/limits`、`/device-authorizations`、`/credentials/current`），难以从 URL 辨认认证与管理职责。为后续稳定开放 HTTP 接口并预留独立的 MCP 接入点，需要先明确路径边界，同时保护已安装客户端。

## What Changes

- 在控制域新增 `POST /auth/device`、`POST /auth/device/poll` 和 `DELETE /auth/credentials/current`，分别承接设备授权创建、轮询和当前 Bearer 凭证撤销；浏览器登录与控制台设备审批路径保持现状。
- 在控制域新增 `/api/control/v1` 下的所有现有 Bearer 管理操作，包括额度、应用、部署和应用表管理；方法、请求与响应、授权、错误码及幂等语义保持一致。
- 保留旧根路径作为同处理逻辑的兼容入口，让旧版 CLI 在迁移期间继续工作；新版 CLI/Agent 使用新路径，正式删除旧路径需另行设计版本与退役计划。
- 同步控制域边缘路由、公开客户端契约、调用代码、测试和文档，验证新旧路径及不同认证方式的隔离。
- 明确应用域的 `/_cellapp/*` 和 `/rest/v1/*` 不属于控制 API；本次不实现 MCP 协议，只为未来独立入口保留命名空间。

## Capabilities

### New Capabilities

- `control-api-routing`: 规定控制域认证、浏览器控制台与程序化控制接口的路径、认证边界和旧路径兼容行为。

### Modified Capabilities

None. Existing deployment authorization and management behavior stays the same; the new capability defines its HTTP entry points.

## Impact

服务端路由 `apps/server/internal/hosting`、Cloudflare Worker 和根域名路径清单、公开 `skills` 子仓库的 CLI 与 HTTP 契约、相关测试及部署文档需要更新。应用域、数据库结构、OAuth 回调地址、浏览器控制台 API 的请求/响应均不变。`unify-root-domain-dashboard` 仍在进行，实施时要与它的根域路由清单协调，并先发布兼容服务与边缘路由，再让新版 CLI 使用新路径；npm 与实际服务发布是独立交付步骤。
