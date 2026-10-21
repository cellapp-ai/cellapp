# Spec Delta

## MODIFIED Requirements

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
