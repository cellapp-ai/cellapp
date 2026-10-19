# Proposal

## Why

Codex、豆包等支持 MCP 的 Agent 目前只能阅读 Cellapp 指引并自行调用本机 CLI，无法通过标准远程工具连接直接查询、发布应用和管理应用数据表。提供受所有者授权的远程 MCP 入口，可让不同 Agent 使用同一套明确、可审计的能力，同时保留静态产物、应用数据和私有存储边界。

## What Changes

- 在 Cellapp 控制域提供独立的 HTTPS MCP 端点与工具发现；首版覆盖所有者登录授权、应用列表、创建应用、部署准备与上传、部署状态和发布确认。
- 提供面向远程 MCP 客户端的 OAuth 授权发现、浏览器同意、短期凭证和撤销流程；MCP 凭证与 CLI 设备凭证分开，不允许浏览器 Cookie 或访客密钥直接管理应用。
- 通过受限的 MCP 工具接收 Agent 在本地生成的静态产物，复用现有清单校验、额度、幂等和原子发布语义；服务端不构建或运行上传的程序。
- 提供应用数据状态、故障协调、版本化表变更预览/应用和数据接口验证工具，复用现有受限清单与所有者权限；MCP 不创建任意 PostgreSQL 数据库、不接受任意 SQL 或数据库管理员凭证。
- 提供 Codex 和豆包的连接说明与真实客户端验收；保留现有 CLI、Skill 与 REST 控制接口。

## Capabilities

### New Capabilities

- `mcp-agent-access`: 远程 MCP 连接、所有者 OAuth 授权、应用与发布工具、静态产物传输、应用数据表管理及结果语义。

### Modified Capabilities

- `control-api-routing`: 为控制域新增独立 MCP 与 OAuth 发现/授权路径，并明确与现有 REST、浏览器和应用域的隔离。

## Impact

- 服务端 `apps/server/internal/hosting` 的路由、授权、部署与应用数据适配、存储与清理；必要时增加增量控制数据库迁移及配置。应用数据继续使用现有独立 PostgreSQL 数据库与隔离 schema。
- 控制域的 Cloudflare 路由、Cloud Run 配置、运维与协议说明；公开连接文档及跨客户端验收。
- 评估并引入维护中的 MCP Go SDK；使用现有业务处理与存储接口，避免复制部署规则。现有 CLI/Skill 仍可工作，不改变公开 REST 契约。
