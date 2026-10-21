# static-app-lifecycle Specification

## Purpose

管理免费静态应用从创建、完整发布、持续更新到删除的生命周期，保持访问地址稳定，并以明确的额度和错误行为控制存储、上传和分发资源使用。

## Requirements

### Requirement: Atomic releases at a stable address
系统 SHALL 为应用提供稳定 HTTPS 地址，仅将完整且验证通过的部署发布为当前版本；失败和过期的并发发布 MUST NOT 替换已生效版本。

#### Scenario: Failed update
- **WHEN** 新版本上传中断、校验失败或发布失败
- **THEN** 原版本继续服务，未发布对象不对访客开放

#### Scenario: Concurrent updates
- **WHEN** 两个更新基于相同旧版本发布且其中一个先成功
- **THEN** 后一个返回版本冲突，不静默覆盖先发布成功的版本

### Requirement: Static routing
系统 SHALL 按正确内容类型提供文件，并支持显式启用的 SPA 页面回退；缺失静态资源 MUST NOT 被伪装为成功 HTML 响应。

#### Scenario: SPA deep link
- **WHEN** 已授权访客以 HTML 导航请求启用 SPA 的应用内不存在文件的页面路径
- **THEN** 系统返回当前版本 index.html；未启用 SPA 的同类请求返回 404

#### Scenario: Missing asset
- **WHEN** 已授权访客请求不存在的脚本或图片资源
- **THEN** 系统返回 404，不返回 SPA 入口

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

### Requirement: Delete and reclaim
系统 SHALL 允许所有者删除应用，立即阻止新的授权、应用数据访问和发布，并在最多 60 秒内阻止既有短期授权读取静态内容；随后最终回收其对象、应用数据及用量。跨数据库与对象存储的清理 MUST 可重试，失败不得重新开放已删除应用。被删除应用的地址标识 MUST NOT 分配给其他应用。

#### Scenario: Delete live app
- **WHEN** 所有者删除已有应用
- **THEN** 新授权、数据访问与部署立即被拒绝，已签发的静态授权最迟 60 秒后失效，后台清理可重试并释放实际占用额度

#### Scenario: Abandoned deployment cleanup
- **WHEN** 未完成上传或未引用版本超过配置保留期限
- **THEN** 系统清理对应对象和预留用量，不删除当前有效版本或仍在使用的应用数据

#### Scenario: Data cleanup failure
- **WHEN** 删除应用后独立应用数据库暂时不可用
- **THEN** 应用仍保持不可访问，清理任务保留可重试状态，恢复后回收该应用表和数据且不影响其他应用
