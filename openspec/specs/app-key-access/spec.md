# app-key-access Specification

## Purpose

通过每个应用独立的分享密钥保护托管页面及其全部资源，让访客在不同设备验证后使用应用，并使应用所有者能够撤销访问而不暴露部署权限。

## Requirements

### Requirement: Independent protected apps
系统 SHALL 默认保护每个应用并分配独立、高熵的分享密钥；本期 MUST NOT 提供公开免密模式，且 MUST NOT 在 URL、请求日志或可读持久存储中保留密钥明文。

#### Scenario: App scoped key
- **WHEN** 访客用应用 A 的密钥尝试访问应用 B
- **THEN** 验证失败，不建立应用 B 的访问会话

### Requirement: Session based resource protection
系统 SHALL 在密钥验证成功后建立有期限且仅属于该应用的浏览器会话，并在每次资源请求中验证权限；底层对象、缓存、HEAD 和条件请求 MUST NOT 绕过保护。

#### Scenario: First navigation and return
- **WHEN** 无会话访客打开应用深层链接并提交有效密钥
- **THEN** 系统显示验证页，验证成功后返回原应用内路径；未过期会话访问无需重复输入

#### Scenario: Direct asset request
- **WHEN** 无有效会话的调用者请求 JS、图片、HEAD、条件请求或底层对象地址
- **THEN** 系统不返回受保护内容，也不通过共享缓存泄露该内容

#### Scenario: Expired session or repeated failures
- **WHEN** 会话到期或某来源反复提交无效密钥
- **THEN** 系统分别要求重新验证或限制尝试频率，不创建有效访问会话

### Requirement: Key reset revokes future access
系统 SHALL 支持所有者重置应用密钥，使旧密钥及此前建立的会话在后续请求中失效；普通重新部署 SHALL 保留密钥及有效会话。

#### Scenario: Reset while visitor is active
- **WHEN** 所有者重置密钥后，访客提交旧密钥或使用旧会话请求资源
- **THEN** 系统拒绝访问，要求使用新密钥验证；其他应用不受影响

### Requirement: Browser isolation
系统 SHALL 将各应用及平台控制面隔离于不同来源，避免共享认证与浏览器存储；托管应用 MUST NOT 注册可接管平台验证路径的 Service Worker。

#### Scenario: Cross app isolation
- **WHEN** 应用 A 的脚本尝试读取应用 B 或控制面的会话、存储或管理响应
- **THEN** 平台来源与权限策略阻止未授权读取，不向其暴露所有者凭证

#### Scenario: Offline interception attempt
- **WHEN** 托管应用尝试注册 Service Worker
- **THEN** 平台阻止注册；密钥重置仅保证阻止未来在线读取，不承诺删除已下载副本
