# edge-app-delivery Specification

## Purpose

让已登录且获准进入应用的访客以短期、应用域名专属的授权从边缘读取私有静态产物，同时避免每个文件都经过控制服务，并保持未授权资源与不同应用来源的隔离。

## Requirements

### Requirement: Batched static-resource grant
系统 SHALL 在验证平台登录、应用状态与访问权后，或验证目标公开应用已开启匿名页面访问后，为当前应用域名、发布版本和相应访问模式签发有效期不超过 60 秒的静态资源授权。有效应用会话或仍被允许的匿名页面访问在授权到期时 SHALL 可无感续签；授权本身 MUST NOT 延长底层应用会话或授予数据权限。

#### Scenario: Page with multiple assets
- **WHEN** 已获准访客打开应用页面并随后请求同一发布版本的脚本、样式和图片
- **THEN** 一次有效授权覆盖这些静态请求，无需逐文件向控制服务重新验证

#### Scenario: Expired grant with valid session
- **WHEN** 授权到期，但访客的应用会话和访问权仍有效
- **THEN** 下一次静态请求自动续签并继续加载，无需重新登录或刷新页面

#### Scenario: Renewal denied or unavailable
- **WHEN** 授权已到期且应用会话失效、访问权不再有效、匿名页面访问已关闭或授权服务不可用
- **THEN** 边缘不得提供受保护的文件；需登录应用沿用现有登录恢复流程，服务故障返回可识别失败

#### Scenario: Anonymous page grant
- **WHEN** 未登录访客打开已开启匿名页面访问的公开应用并加载多个静态资源
- **THEN** 边缘使用只适用于该应用域名、发布版本和匿名静态访问模式的短期授权提供资源，不创建登录会话

### Requirement: Private and isolated edge delivery
边缘 SHALL 先验证授权的完整性、期限、精确访客域名、应用及发布版本，再读取私有产物或内部缓存。它 MUST NOT 向浏览器暴露对象存储凭证、公开对象地址或可跨应用复用的授权；未知路径、未发布版本和未授权的 GET、HEAD、条件请求 MUST NOT 返回文件内容。

#### Scenario: Authorized immutable file
- **WHEN** 访客带有效授权请求其应用当前发布版本的文件
- **THEN** 边缘按该版本返回正确内容类型和文件内容，并可复用经授权后读取的版本化文件缓存

#### Scenario: Host or version replay
- **WHEN** 某应用或旧发布版本的授权被用于另一域名、应用或发布版本
- **THEN** 边缘拒绝该授权，且不读取目标对象

#### Scenario: Unauthenticated cache and object access
- **WHEN** 请求没有有效授权，或直接请求对象存储、猜测对象键、发送 HEAD 或条件请求
- **THEN** 私有文件不会因缓存命中或底层对象地址而泄露

#### Scenario: SPA fallback and missing asset
- **WHEN** 已授权访客请求启用 SPA 的应用页面路径，或请求不存在的脚本/图片
- **THEN** 页面路径可回退至该版本入口文件，缺失的脚本/图片返回 404

### Requirement: Route separation and bounded state delay
登录、授权变更、控制 API 与应用数据 API SHALL 继续进入控制服务；边缘只处理已授权的静态 GET/HEAD。已签发静态授权在权限撤销、公开或匿名访问状态改变、应用删除、停用或新版本发布后 MUST 最迟于 60 秒内失效；新授权 MUST 使用最新状态。边缘 SHALL 对访客请求保留独立限速，并在签名、存储或授权依赖故障时拒绝受保护内容。

#### Scenario: Data API and mutation
- **WHEN** 访客调用应用数据接口或提交登录、访问申请、密钥兑换等变更
- **THEN** 请求继续由控制服务执行逐请求鉴权，边缘不把它当作静态对象读取

#### Scenario: Revocation during active page
- **WHEN** 所有者撤销访问权、关闭匿名页面访问或运营人员停用应用，而访客仍持有先前签发的授权
- **THEN** 新授权被拒绝，既有静态授权最多继续有效 60 秒，之后所有静态文件请求被拒绝

#### Scenario: Edge dependency failure
- **WHEN** 边缘无法验证授权或访问私有对象存储
- **THEN** 它返回失败，不回退到公开读取、其他应用对象或未经授权的缓存内容
