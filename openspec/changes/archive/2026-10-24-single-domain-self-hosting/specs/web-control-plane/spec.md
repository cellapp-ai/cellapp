# Spec Delta

## MODIFIED Requirements

### Requirement: Browser API security and existing client compatibility
浏览器所有者 API SHALL 使用 `/api/console` 路径前缀、独立会话边界、逐资源所有者鉴权和同源写请求检查。所有写请求 MUST 拒绝缺失或不匹配控制域的 Origin；有请求体时限制大小并校验输入。所有者 Cookie MUST 仅属于控制主机，不得发送给同一注册域名的应用主机。既有 CLI Bearer API、OAuth 回调及设备轮询 SHALL 维持兼容；应用网关按新的平台登录和用户访问权契约运行，控制域凭证不得直接用于访客数据操作。

#### Scenario: Uniform browser API prefix
- **WHEN** Web 请求会话、退出、应用管理、数据设置、部署凭证或设备授权决策接口
- **THEN** 这些浏览器 JSON 接口均位于 `/api/console` 下，页面路由与 CLI 接口继续使用各自路径

#### Scenario: Same origin owner mutation
- **WHEN** 已登录所有者从控制域提交合法管理写请求
- **THEN** 请求经会话、来源、输入和资源归属验证后执行，私有响应不可被共享缓存

#### Scenario: Cross origin or missing origin
- **WHEN** 写请求的 Origin 缺失、来自外站或应用访客域
- **THEN** 服务端拒绝且不改变应用、访问权、会话、凭证或设备授权状态

#### Scenario: Browser cookie cannot authenticate the CLI API
- **WHEN** 仅携带所有者 Cookie 调用既有 Bearer 应用 API，或从访客域请求浏览器 API
- **THEN** 不获得额外权限；合法 CLI 与授权轮询保持原有协议，访客请求须通过应用访问权检查

#### Scenario: Same-site application attempts owner actions
- **WHEN** 静态应用运行在控制主机同一注册域名的子域，并向控制主机发送带所有者 Cookie 的跨来源管理读写请求
- **THEN** 应用无法读取私有管理响应，管理写请求因 Origin 不匹配被拒绝，应用主机也不会收到所有者 Cookie
