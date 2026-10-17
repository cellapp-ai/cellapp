# control-api-routing Specification

## Purpose

为 Cellapp 控制域的浏览器认证、控制台操作和程序化管理提供清晰且可兼容迁移的 HTTP 路径边界，同时保持应用访客域的身份与数据接口独立。

## Requirements

### Requirement: Canonical device authorization paths
控制域 SHALL 通过 `POST /auth/device` 创建设备授权请求，通过 `POST /auth/device/poll` 凭设备秘密轮询并领取部署凭证。新路径 SHALL 保持现有设备授权请求、成功响应、错误、有效期、频率限制及一次性领取语义。

#### Scenario: New client authorizes a device
- **WHEN** 未持有部署凭证的客户端调用新建路径、所有者通过现有浏览器页面明确批准，并按规定频率调用新轮询路径
- **THEN** 客户端取得一次绑定该所有者的凭证；浏览器页面不取得凭证明文

#### Scenario: Invalid or repeated polling
- **WHEN** 客户端用错误设备秘密、过期或已领取请求轮询，或超过轮询频率
- **THEN** 新路径返回与旧路径相同的错误码和状态，不额外签发凭证

### Requirement: Canonical current credential revocation path
控制域 SHALL 通过 `DELETE /auth/credentials/current` 撤销调用者当前的 Bearer 部署凭证。浏览器所有者 Cookie、应用访问密钥或应用域会话 MUST NOT 代替该凭证。

#### Scenario: CLI logs out
- **WHEN** 客户端以有效 Bearer 凭证调用新撤销路径
- **THEN** 返回与旧路径相同的成功结果，凭证随后无法再管理应用

#### Scenario: Cookie-only revocation attempt
- **WHEN** 请求仅携带浏览器所有者 Cookie 或没有有效 Bearer 凭证
- **THEN** 撤销被拒绝，其他凭证保持有效

### Requirement: Versioned programmatic control paths
控制域 SHALL 在 `/api/control/v1` 下提供现有 Bearer 管理接口的完整路径树：`/limits`、`/apps` 及其应用、部署和数据子路径。新路径 SHALL 保持原方法、请求与响应、授权、错误码和幂等语义；浏览器 Cookie MUST NOT 授予控制 API 管理权限。

#### Scenario: Owner manages an application
- **WHEN** 有效部署凭证调用 `/api/control/v1/apps`、`/api/control/v1/limits` 或已有应用子路径
- **THEN** 与对应旧根路径具有相同的资源范围、成功响应及失败行为

#### Scenario: Cross-owner or cookie-only management
- **WHEN** 调用者访问其他所有者资源，或只携带浏览器 Cookie 调用新控制路径
- **THEN** 服务端继续按既有 Bearer 所有者边界拒绝请求，不暴露或修改资源

### Requirement: Legacy path compatibility
控制域 SHALL 在迁移期间继续接受现有的 `/device-authorizations`、`/device-authorizations/poll`、`/credentials/current`、`/limits` 和 `/apps` 路径树。旧路径 SHALL 与对应新路径使用相同业务状态和处理语义，不经会丢失方法、请求体或 Authorization 的 HTTP 重定向；移除旧路径需要单独的退役变更。

#### Scenario: Previously installed CLI continues working
- **WHEN** 旧版 CLI 在兼容服务上线后通过根路径登录、发布、管理数据或退出
- **THEN** 它仍能完成原有流程，不要求立即升级

#### Scenario: Retry across old and new paths
- **WHEN** 同一所有者使用相同请求标识和内容分别在旧、新创建应用或部署路径重试
- **THEN** 服务端按原有幂等规则返回同一资源；相同标识但不同内容仍返回冲突

### Requirement: Host and browser API isolation
`/auth/device*` 与 `/api/control/v1/*` SHALL 仅在控制域提供；`/api/console/*` SHALL 继续使用浏览器会话，应用域 `/_cellapp/*` 与 `/rest/v1/*` SHALL 继续使用应用访问检查。未知控制 API 路径 MUST 返回失败，不得落入官网页面或应用静态回退。

#### Scenario: Request reaches wrong host or API family
- **WHEN** 应用域请求控制 API、控制域凭证直接请求应用数据，或 Cookie 与 Bearer 被用于不对应的 API
- **THEN** 请求不能跨越原有身份及数据隔离边界

#### Scenario: Unknown control route
- **WHEN** 客户端请求 `/api/control/v1/unknown` 或与已声明路径相似但不匹配的路径
- **THEN** 返回 API 失败而不是官网 HTML、控制台页面或成功响应
