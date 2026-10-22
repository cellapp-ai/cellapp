# Spec Delta

## MODIFIED Requirements

### Requirement: Host and browser API isolation
`/auth/device*` 与 `/api/control/v1/*` SHALL 仅在控制域提供；`/api/console/*` SHALL 继续使用浏览器会话，应用域 `/_cellapp/*`（包括 `/_cellapp/db/{table}`）SHALL 继续使用应用访问检查。旧 `/rest/v1/{table}` MUST NOT 提供表操作或落入应用静态回退。未知控制 API 路径 MUST 返回失败，不得落入官网页面或应用静态回退。

#### Scenario: Request reaches wrong host or API family
- **WHEN** 应用域请求控制 API、控制域凭证直接请求应用数据，或 Cookie 与 Bearer 被用于不对应的 API
- **THEN** 请求不能跨越原有身份及数据隔离边界

#### Scenario: Unknown control route
- **WHEN** 客户端请求 `/api/control/v1/unknown` 或与已声明路径相似但不匹配的路径
- **THEN** 返回 API 失败而不是官网 HTML、控制台页面或成功响应

#### Scenario: Former table route
- **WHEN** 应用域请求 `/rest/v1/notes`，即使请求声明接受 HTML 且应用使用 SPA 回退
- **THEN** 返回 404，不读取表，也不返回静态页面
