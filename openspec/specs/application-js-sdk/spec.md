# application-js-sdk Specification

## Purpose
向 Cellapp 托管的静态前端提供统一的 JavaScript 客户端，让应用使用平台管理的登录身份、访问状态和受管表数据，而不自行处理凭证与接口细节。

## Requirements

### Requirement: Browser client uses app-scoped session and pinned contract
官方 JavaScript SDK SHALL 在当前应用来源使用浏览器已有会话，并在运行时 API 请求上发送本次前端构建固化的 `major.minor` 契约别名。SDK MUST NOT 将控制域凭证、应用密钥或数据库管理凭证嵌入产物，也不得将公开客户端标识当作授权凭证。

#### Scenario: Initialize a published app
- **WHEN** 访客打开使用 SDK 构建并发布的应用页面
- **THEN** SDK 在本应用来源读取该页面固化的 API 版本，并以同源凭证发出带版本的请求

#### Scenario: Missing or invalid build version
- **WHEN** SDK 无法取得有效的固定版本
- **THEN** 它明确失败，不静默改用无版本的最新接口

### Requirement: Current application user and access errors
SDK SHALL 提供读取当前应用用户的方法，返回应用范围内的稳定 ID 和访问状态；未登录、待审核、被拒绝、需要授权及服务故障 SHALL 表达为可区分的错误。SDK MUST NOT 宣称提供尚未授权的额外个人资料或全局平台身份。

#### Scenario: Admitted visitor
- **WHEN** 已登录且获准的访客读取当前用户
- **THEN** SDK 返回本应用用户 ID 与已获准状态

#### Scenario: Visitor cannot enter
- **WHEN** 访客未登录或未获准进入受限应用
- **THEN** SDK 返回与服务端状态对应的可识别错误，不把该状态当作匿名用户身份

### Requirement: Supported table client
SDK SHALL 为应用表提供已声明支持的查询、新增、修改与删除调用，把内部 Supabase 表客户端请求发送到公开的 `/_cellapp/db/{table}` 路径，并保持现有表权限和行策略由服务端逐次执行。它 MUST 明确限制可用查询操作，不能将 Supabase Auth、Storage、Realtime、Functions、RPC 或数据库直连描述为已支持。

#### Scenario: Authorized CRUD
- **WHEN** 已获准访客使用 SDK 对获授权表执行受支持的 CRUD
- **THEN** 返回数据或可识别错误，且请求保持该应用的身份、版本和表权限边界

#### Scenario: Unauthorized operation
- **WHEN** SDK 调用者请求未获授权的表操作或企图伪造其他应用身份
- **THEN** 服务端拒绝，SDK 不通过客户端配置提升权限
