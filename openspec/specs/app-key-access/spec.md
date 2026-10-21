# app-key-access Specification

## Purpose

通过每个应用独立的分享密钥保护托管页面及其全部资源，让访客在不同设备验证后使用应用，并使应用所有者能够撤销访问而不暴露部署权限。

## Requirements

### Requirement: Independent protected apps
系统 SHALL 默认让应用要求 Cellapp 登录；新应用默认受限，所有者可将应用设为公开或受限。公开应用允许任何已登录平台用户进入；仅当所有者显式开启匿名页面访问时，未登录访客也可进入该公开应用。受限应用仅允许所有者及持有有效访问权的已登录用户进入，MUST NOT 开启匿名访问。每个受限应用 SHALL 有独立、高熵的访问密钥；密钥明文 MUST NOT 进入 URL、请求日志或可读持久存储，也 MUST NOT 授予管理权限。

#### Scenario: App scoped key
- **WHEN** 已登录用户用应用 A 的访问密钥尝试进入应用 B
- **THEN** 兑换失败，不产生 B 的访问权

#### Scenario: Public and restricted admission
- **WHEN** 已登录用户分别访问公开应用和未获准的受限应用
- **THEN** 公开应用直接开放，受限应用提供申请访问或输入访问密钥的入口

#### Scenario: Anonymous public app admission
- **WHEN** 未登录访客访问已开启匿名页面访问的公开应用页面、深链或静态资源
- **THEN** 页面及其资源可加载，访客不获得 Cellapp 登录身份、应用用户 ID 或所有者权限

#### Scenario: Anonymous admission disabled
- **WHEN** 未登录访客访问未开启匿名页面访问的公开应用或任何受限应用
- **THEN** 系统沿用登录流程，不返回受保护的页面或静态资源

### Requirement: Session based resource protection
系统 SHALL 在签发或续签静态资源授权时验证登录身份与访问权，或验证目标公开应用当前已开启匿名页面访问；随后在每次静态页面和资源请求验证短期授权。应用数据请求 MUST 逐次验证当前登录和访问权，或单独验证当前匿名数据授权。有效应用浏览器会话和短期静态授权仅属于相应用户或匿名访问模式及应用；底层对象、缓存、HEAD 和条件请求 MUST NOT 绕过保护。既有静态授权失效所需时间 MUST 不超过 60 秒。

#### Scenario: First navigation and return
- **WHEN** 无应用会话访客打开仍需登录的应用深层链接并完成平台登录及必要的访问授权
- **THEN** 系统返回原应用内路径；后续有效会话访问无需重复登录或兑换密钥，静态授权可无感续签

#### Scenario: Direct asset request
- **WHEN** 无有效登录、短期静态授权或应用访问权的调用者请求仍受保护的 JS、图片、HEAD、条件请求或底层对象地址
- **THEN** 系统不返回受保护内容，也不通过共享缓存泄露该内容

#### Scenario: Expired session or repeated failures
- **WHEN** 底层会话到期或某来源反复提交无效访问密钥
- **THEN** 系统分别要求重新登录或限制尝试频率，不创建新的有效访问权；需登录应用的短期静态授权不会续签

#### Scenario: Data access after grant revocation
- **WHEN** 用户的应用访问权被撤销后再次调用应用数据接口，但尚持有未到期的静态授权
- **THEN** 数据接口立即拒绝该用户的访问，静态授权最迟在 60 秒内失效；不得把失效的已登录授权降级为匿名授权

#### Scenario: Anonymous page access turned off
- **WHEN** 所有者关闭匿名页面访问或将应用改为受限
- **THEN** 新的匿名静态授权立即被拒绝，已有匿名授权最迟在 60 秒内失效

### Requirement: Key reset revokes future access
系统 SHALL 支持所有者重置应用访问密钥，使旧密钥不能再兑换新的用户访问权；已授予的用户访问权 MUST 保留，直到所有者单独撤销、应用删除或其他明确的授权失效条件发生。普通重新部署 SHALL 保留密钥和访问权。

#### Scenario: Reset while visitor is active
- **WHEN** 所有者重置密钥后，未获准用户提交旧密钥，或已获准用户继续访问
- **THEN** 旧密钥兑换失败，已获准用户继续访问，其他应用不受影响

### Requirement: Browser isolation
系统 SHALL 将各应用及平台控制面隔离于不同来源，避免共享认证与浏览器存储；托管应用 MUST NOT 注册可接管平台验证路径的 Service Worker。

#### Scenario: Cross app isolation
- **WHEN** 应用 A 的脚本尝试读取应用 B 或控制面的会话、存储或管理响应
- **THEN** 平台来源与权限策略阻止未授权读取，不向其暴露所有者凭证

#### Scenario: Offline interception attempt
- **WHEN** 托管应用尝试注册 Service Worker
- **THEN** 平台阻止注册；密钥重置仅保证阻止未来在线读取，不承诺删除已下载副本

### Requirement: User-bound application grants
系统 SHALL 将访问申请批准、访问密钥兑换和所有者直接授权记录为属于特定平台用户及应用的可撤销访问权。公开模式本身 MUST NOT 自动为每位访客创建永久访问权；所有者撤销或修改应用公开状态后，新的静态授权和后续数据请求 MUST 立即反映变化，已签发的静态授权 MUST 最迟在 60 秒内失效。

#### Scenario: Redeem access key
- **WHEN** 已登录用户为受限应用提交有效密钥
- **THEN** 系统原子地授予该用户应用访问权，重复兑换不会产生重复授权

#### Scenario: Switch public app to restricted
- **WHEN** 所有者将公开应用改为受限
- **THEN** 没有明确访问权的已登录访客不能取得新的静态授权或访问数据；已有静态授权最迟 60 秒后停止服务

#### Scenario: Revoke an individual grant
- **WHEN** 所有者撤销某用户对受限应用的访问权
- **THEN** 该用户后续数据请求立即被拒绝、静态授权最迟 60 秒后失效，其他用户的访问权不受影响

### Requirement: Owner-reviewed access requests
系统 SHALL 允许已登录用户为受限应用提交一次有效的待审访问申请，并允许所有者批准或拒绝；同一申请的并发决策 MUST 只产生一个最终结果。拒绝不授予访问权，申请者可获知其状态但不能读取所有者或其他申请者的信息。

#### Scenario: Approve a request
- **WHEN** 所有者批准有效的待审申请
- **THEN** 申请者取得该应用访问权，之后无需再次申请

#### Scenario: Deny or duplicate request
- **WHEN** 所有者拒绝申请，或申请者重复提交仍待审的申请
- **THEN** 拒绝不产生访问权，重复提交不产生多条待审记录

### Requirement: Existing key migration
系统 SHALL 允许现有应用的未轮换分享密钥在新登录流程中作为访问密钥兑换；仅持有旧密钥 Cookie 或旧匿名会话 MUST NOT 获得登录身份或应用访问权。

#### Scenario: Existing visitor after upgrade
- **WHEN** 旧访客携带旧会话进入受限应用
- **THEN** 系统要求平台登录；登录后可输入仍有效的旧密钥获得用户绑定的访问权
