# Spec Delta

## ADDED Requirements

### Requirement: Dedicated MCP control routes
控制域 SHALL 在 `/mcp` 提供远程 MCP 协议入口，并在控制域提供该资源所需的 OAuth 发现与授权端点。MCP 端点 MUST NOT 被当作现有 REST 控制接口、控制台页面或静态站点回退处理；应用域 MUST NOT 承载这些路径。

#### Scenario: Agent reaches control domain
- **WHEN** Agent 在控制域请求 `/mcp` 及其授权发现地址
- **THEN** 收到 MCP 协议或 OAuth 元数据响应，并可按声明流程连接

#### Scenario: Wrong host or unknown MCP route
- **WHEN** 请求在应用域访问 MCP 路径，或在控制域访问未知 MCP 子路径
- **THEN** 请求失败，不暴露应用管理能力，也不返回官网或应用页面

#### Scenario: Existing client after MCP rollout
- **WHEN** 现有 CLI 继续访问 `/auth/device*` 或 `/api/control/v1/*`
- **THEN** 原有方法、凭证、响应及错误语义保持可用
