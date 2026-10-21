# Spec Delta

## MODIFIED Requirements

### Requirement: Free bounded hosting
系统 SHALL 免费提供本期托管，并执行可配置的应用数、文件数、单文件大小、部署大小、总存储及每应用每 UTC 月已登录独立访客数限制；并发首次访问 MUST NOT 绕过已登录访客上限，超限 MUST NOT 触发自动收费。已登录访客按平台用户 ID 去重，同一用户的页面、资源及数据请求在同一应用和月份内只计一次；默认上限为 1,000 位。所有者访问不占访客名额。匿名访问 SHALL 受单独的请求速率和资源限制，不通过不可靠的 Cookie 或 IP 冒充已登录独立访客计数。

#### Scenario: Upload or creation quota exceeded
- **WHEN** 创建或上传超过可用额度，包括并发预留用量
- **THEN** 系统拒绝操作并返回具体限制与恢复办法，已发布应用保持不变

#### Scenario: Traffic exhausted
- **WHEN** 应用当月已接纳上限数量的独立访客，另一已登录且有访问权的用户首次访问页面、资源或应用数据接口
- **THEN** 系统拒绝该新访客的访问，返回 429 和下个 UTC 月份的恢复时间；所有者仍能访问控制面

#### Scenario: Previously counted visitor and repeated requests
- **WHEN** 应用达到当月访客上限，而当月已计入的用户再次打开页面、加载资源或调用应用数据接口
- **THEN** 该用户继续访问且不重复计数；单个文件的大小和请求次数不作为月访客数

#### Scenario: Unauthorized or concurrent first visits
- **WHEN** 未登录且匿名访问未开启、未获准的已登录用户访问，或同一已登录用户同时发起多个首次请求
- **THEN** 失败的授权不计入名额，成功的同一已登录用户至多计一次，独立已登录用户不会越过设定上限

#### Scenario: Anonymous request limits
- **WHEN** 匿名访客访问已开放的页面或数据接口，或匿名请求达到相应限制
- **THEN** 成功请求不消耗已登录独立访客名额；系统仍按应用及来源限制请求，超限返回 429，不产生自动收费

#### Scenario: New UTC month
- **WHEN** UTC 月份切换，先前访问过的已登录用户再次进入应用
- **THEN** 新月份独立计数并适用该月上限，旧月份的计数仍可供运营查看
