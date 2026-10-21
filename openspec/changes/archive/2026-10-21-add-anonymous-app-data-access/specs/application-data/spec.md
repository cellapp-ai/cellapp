# Spec Delta

## MODIFIED Requirements

### Requirement: Application user row authorization
系统 SHALL 根据当前已认证且获准进入应用的用户执行逐表、逐操作和必要的逐行权限检查。对未登录访客，系统 SHALL 仅在公开应用的匿名页面及相应数据操作均已开启、且目标表明确授予该匿名操作时允许读取或新增记录；匿名访客 MUST NOT 修改或删除记录。表权限 MUST 区分应用所有者管理权、已登录访客数据权与匿名访客数据权；前端不得通过伪造用户 ID、请求头或字段提高权限。

#### Scenario: User-owned rows
- **WHEN** 表规则仅允许用户操作自己的记录，用户尝试查询或修改其他用户的记录
- **THEN** 其他用户的记录不可读取或修改

#### Scenario: Public application with private table
- **WHEN** 已登录用户可以进入公开应用，但表没有授予其读取权限
- **THEN** 数据接口仍拒绝读取，不以应用公开状态覆盖表权限

#### Scenario: Access revoked between requests
- **WHEN** 用户的应用访问权被撤销后再次调用数据接口
- **THEN** 后续请求不能继续读写，即使浏览器仍持有旧页面或会话；不得降级到匿名权限

#### Scenario: Anonymous table isolation
- **WHEN** 匿名访客尝试访问未明确授权的表、其他应用的表、用户专属记录或管理数据
- **THEN** 请求被拒绝，匿名身份无法取得用户身份或跨越应用及数据库边界

## ADDED Requirements

### Requirement: Operator and owner gates for anonymous data
系统 SHALL 提供默认关闭的运维配置开关，控制平台是否接受匿名新增记录。所有者 SHALL 能分别设置自有公开应用的匿名页面访问、匿名读取和匿名新增记录，三项对既有和新应用均默认关闭；匿名读取和新增记录只有在匿名页面访问开启时才可开启。匿名新增记录 MUST 同时满足运维与应用开关，匿名读取不受写入总开关影响。

#### Scenario: All gates permit anonymous insert
- **WHEN** 运维允许匿名新增记录、公开应用开启匿名页面与写入、目标表允许匿名新增，且请求符合其他限制
- **THEN** 未登录访客可向该表新增记录，不创建平台用户或应用访问权

#### Scenario: Operator disables anonymous submissions
- **WHEN** 运维开关关闭，而应用和表仍允许匿名新增记录
- **THEN** 后续匿名新增请求立即得到稳定、可识别的拒绝；现有记录和已登录用户的权限不变

#### Scenario: Owner disables one operation
- **WHEN** 所有者关闭匿名读取或匿名写入，或将应用改为受限、停用或删除
- **THEN** 后续对应匿名数据请求立即被拒绝，不因旧页面或静态授权继续执行

#### Scenario: Invalid settings and ownership
- **WHEN** 非所有者修改设置，或所有者试图为受限应用开启匿名能力、在未开启匿名页面时开启匿名数据能力
- **THEN** 系统拒绝修改且保持先前设置不变

### Requirement: Explicit anonymous table policies
版本化表定义 SHALL 为匿名查询和新增记录提供独立的显式授权，默认均为拒绝。匿名权限 MUST 不继承已登录用户的 `all-admitted-users` 或 `own-rows` 规则；匿名查询只支持明确公开的行，匿名新增不得依靠客户端声明的用户身份。匿名 `PATCH`、`DELETE` 及未支持的接口 SHALL 被拒绝。

#### Scenario: App setting without table permission
- **WHEN** 应用允许匿名读取或新增，但目标表没有相应匿名策略
- **THEN** 数据接口拒绝该表的操作，不因应用级开关而开放全部表

#### Scenario: Insert response without read permission
- **WHEN** 匿名访客向仅授予匿名新增的表请求写后返回记录
- **THEN** 新增操作可按授权执行，但响应不返回未获匿名读取授权的行或字段

#### Scenario: Existing manifest
- **WHEN** 旧版表定义不包含匿名策略
- **THEN** 它仍可加载和更新，匿名查询及新增均保持拒绝

### Requirement: Bounded anonymous data requests
匿名数据请求 SHALL 经过应用专属 HTTPS 入口，并受逐请求应用状态、来源、大小、耗时和按应用及来源的速率限制约束。客户端公开配置、Authorization、profile、用户 ID 或匿名 Cookie MUST NOT 成为绕过运维、应用或表权限的凭证；故障和超限 SHALL 返回可识别错误，不泄露内部凭证或其他应用的数据。

#### Scenario: Cross-origin or forged identity
- **WHEN** 匿名写请求缺失或提供不匹配的 Origin，或试图伪造用户身份、schema 或另一应用地址
- **THEN** 请求被拒绝或仍以目标应用的匿名权限处理，不能取得已登录权限

#### Scenario: Burst or oversized submission
- **WHEN** 同一来源对应用集中提交或请求体、响应、执行时间超过限制
- **THEN** 系统限速或拒绝并给出稳定错误，不继续执行超限写入
