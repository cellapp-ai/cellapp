# Spec Delta

## MODIFIED Requirements

### Requirement: Confirmed application access operations
Web SHALL 提供访问密钥重置、应用公开/受限设置、访问申请审批、用户访问权撤销和应用删除。改变访问权或密钥的操作 SHALL 说明后果、要求所有者确认并阻止重复提交；服务端 MUST 逐次验证所有者及应用状态，且界面只以服务端响应报告成功。

#### Scenario: Reset a share key
- **WHEN** 所有者确认重置自有应用的访问密钥且操作成功
- **THEN** 服务端生成高熵新密钥并只保存摘要，旧密钥不能再兑换；已有用户访问权仍有效，界面仅在本次结果中展示/复制新密钥

#### Scenario: Sensitive result lifecycle
- **WHEN** 用户离开密钥结果页面、退出或会话失效
- **THEN** 新密钥从界面内存清除，不进入 URL、持久存储或日志；后续详情读取不能重新获得密钥明文

#### Scenario: Delete an application
- **WHEN** 所有者确认删除自有应用且服务端成功
- **THEN** 应用从可管理列表移除，原页面、数据访问和后续部署不能继续使用该应用；数据与对象进入可重试清理

#### Scenario: Cancel or uncertain write result
- **WHEN** 用户取消确认，或写请求失败且无法确定服务端是否已执行
- **THEN** 取消不会提交；结果不确定时不自动重试且不声称操作未生效，界面提供重新查询或主动再次操作的指引

#### Scenario: Unauthorized or concurrent mutations
- **WHEN** 请求针对非自有应用，或审批、撤销、密钥重置与删除并发发生
- **THEN** 非自有资源返回 404 且无变化，并发结果不复活已删除应用或已撤销访问权

### Requirement: Browser API security and existing client compatibility
浏览器所有者 API SHALL 使用 `/api/console` 路径前缀、独立会话边界、逐资源所有者鉴权和同源写请求检查。所有写请求 MUST 拒绝缺失或不匹配控制域的 Origin；有请求体时限制大小并校验输入。既有 CLI Bearer API、OAuth 回调及设备轮询 SHALL 维持兼容；应用网关按新的平台登录和用户访问权契约运行，控制域凭证不得直接用于访客数据操作。

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

## ADDED Requirements

### Requirement: Owner view of application access
Web SHALL 为所有者展示自有应用真实的公开/受限状态、待审申请及已授权用户，并允许按申请和用户执行批准、拒绝与撤销；状态变更后 SHALL 给出可辨别的结果，不展示其他应用的用户或申请。

#### Scenario: Review and approve
- **WHEN** 所有者打开自有受限应用的访问管理并批准待审申请
- **THEN** 页面展示真实申请状态及最终授权结果，申请者随后可进入应用

#### Scenario: Revoke or change visibility
- **WHEN** 所有者撤销用户访问权或将公开应用改为受限
- **THEN** 页面反映服务端状态，受影响用户的后续页面和数据请求按新状态判断
