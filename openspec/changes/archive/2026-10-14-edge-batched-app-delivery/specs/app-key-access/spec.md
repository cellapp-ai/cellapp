# Spec Delta

## MODIFIED Requirements

### Requirement: Session based resource protection
系统 SHALL 在签发或续签静态资源授权时验证登录身份、目标应用状态及访问权，随后在每次静态页面和资源请求验证短期授权；应用数据请求 MUST 逐次验证当前登录及访问权。有效应用浏览器会话和短期静态授权仅属于该用户与应用，底层对象、缓存、HEAD 和条件请求 MUST NOT 绕过保护。既有静态授权失效所需时间 MUST 不超过 60 秒。

#### Scenario: First navigation and return
- **WHEN** 无应用会话访客打开应用深层链接并完成平台登录及必要的访问授权
- **THEN** 系统返回原应用内路径；后续有效会话访问无需重复登录或兑换密钥，静态授权可无感续签

#### Scenario: Direct asset request
- **WHEN** 无有效登录、短期静态授权或应用访问权的调用者请求 JS、图片、HEAD、条件请求或底层对象地址
- **THEN** 系统不返回受保护内容，也不通过共享缓存泄露该内容

#### Scenario: Expired session or repeated failures
- **WHEN** 底层会话到期或某来源反复提交无效访问密钥
- **THEN** 系统分别要求重新登录或限制尝试频率，不创建新的有效访问权；短期静态授权不会续签

#### Scenario: Data access after grant revocation
- **WHEN** 用户的应用访问权被撤销后再次调用应用数据接口，但尚持有未到期的静态授权
- **THEN** 数据接口立即拒绝访问，静态授权最迟在 60 秒内失效

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
