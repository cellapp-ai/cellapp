# application-data Specification

## Purpose
为静态应用提供由 Cellapp 托管的持久化 PostgreSQL 表数据，并让应用前端通过受控的数据接口读写本应用的数据，同时隔离平台控制数据和其他应用的数据。

## Requirements

### Requirement: Separate application data database
系统 SHALL 将应用数据放在独立于平台控制数据库的 PostgreSQL 数据库中，并为每个应用隔离其表、数据和数据库权限。应用请求 MUST NOT 获得控制数据库或其他应用的数据访问权。

#### Scenario: Cross-application access attempt
- **WHEN** 应用 A 的请求指定应用 B 的表、schema、用户身份或数据接口地址
- **THEN** 系统拒绝请求，数据库权限也不能读取或修改 B 的对象

#### Scenario: Control data isolation
- **WHEN** 应用数据凭证或请求尝试读取用户、应用、部署或资源等平台控制表
- **THEN** 数据库拒绝访问，不返回控制数据

### Requirement: Owner-managed tables and policies
系统 SHALL 允许已授权所有者通过版本化变更创建和修改本应用表及其读写规则，并提供可重试的执行结果。新表在明确授权前 MUST 拒绝前端读写；浏览器访客 MUST NOT 执行 DDL 或取得数据库管理凭证。

#### Scenario: Create an application table
- **WHEN** 所有者为自有应用提交合法的新版本表定义及访问规则
- **THEN** 系统只在该应用的数据范围内执行，记录已应用版本，并使已授权操作可用

#### Scenario: Application table name overlaps platform or another app
- **WHEN** 两个应用都创建名为 `users` 的表，或应用表名与平台控制表同名
- **THEN** 各应用只能访问自身 schema 中的表，同名不会关联或暴露平台用户数据

#### Scenario: Conflicting or reserved application name
- **WHEN** 表名与本应用其他表或系统生成的索引名冲突，或使用数据接口保留的 `rpc` 路由名及 PostgreSQL 系统列名
- **THEN** 系统在计划或应用前拒绝清单，给出可识别的命名错误，不留下部分建表

#### Scenario: Unauthorized or failed migration
- **WHEN** 非所有者提交变更，或变更失败、版本冲突、越过应用数据边界
- **THEN** 系统拒绝或回滚本次变更，不报告成功，不改变其他应用或平台控制数据

#### Scenario: New table without policy
- **WHEN** 所有者创建表但未赋予某项前端操作权限
- **THEN** 该项操作被拒绝，表不会因创建而自动向应用用户开放

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

### Requirement: Supabase JavaScript table CRUD compatibility
系统 SHALL 提供应用专属 HTTPS 数据入口和可嵌入前端的公开配置，使受支持版本的 `supabase-js` 可对授权表执行 `select`、`insert`、`update`、`delete`，以及常用列选择、等值过滤、排序、分页和写后返回。接口 SHALL 明确记录支持矩阵和错误语义；公开配置 MUST NOT 是管理凭证。

#### Scenario: Static frontend CRUD
- **WHEN** 已登录且获准进入应用的用户使用公开配置创建 `supabase-js` 客户端并调用受支持的表操作
- **THEN** 请求在本应用权限下执行，返回与所声明兼容范围一致的数据或可识别错误

#### Scenario: Unsupported Supabase feature
- **WHEN** 应用调用本期未承诺的 Auth、Realtime、Storage、RPC、数据库直连或其他接口
- **THEN** 系统明确报告不支持，不假装操作成功或授予额外权限

#### Scenario: Forged client key or profile
- **WHEN** 请求伪造公开密钥、授权头、schema profile 或另一应用的 URL
- **THEN** 系统仍按已验证的平台会话及目标应用权限判断，不访问其他应用数据

### Requirement: Bounded data operations
系统 SHALL 对应用数据请求及所有者表变更实施明确的大小、耗时和资源限制，并使超限或数据库故障产生可识别的失败；失败 MUST NOT 泄露 SQL、凭证或其他应用的数据。

#### Scenario: Expensive query or migration
- **WHEN** 请求超过配置的行数、响应大小、执行时间或资源限额
- **THEN** 系统终止或拒绝该操作并返回可恢复的限制信息，不拖垮平台控制请求

#### Scenario: Application data database unavailable
- **WHEN** 独立应用数据库不可用
- **THEN** 数据请求明确失败，平台控制数据不被误认为空数据或成功结果

### Requirement: Owner recovery of failed data provisioning
The control API SHALL report a stable data provisioning failure code to the application owner and SHALL let that owner safely retry provisioning of a non-deleted application. Failure MUST NOT be reported as ready or expose database internals to the client.

#### Scenario: Retry after transient failure
- **WHEN** the application owner requests reconciliation after a temporary data service failure is resolved
- **THEN** the platform retries idempotently and reports ready only after database setup and API verification succeed

#### Scenario: Persistent provisioning failure
- **WHEN** a retry still fails
- **THEN** the application remains unready, the owner can inspect a stable error code, and server logs contain the underlying failure for operators

#### Scenario: Cross-owner retry
- **WHEN** another owner requests reconciliation for the application
- **THEN** the platform denies the request without changing its data state

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

### Requirement: Versioned application table contract
应用域 `/_cellapp/db/{table}` SHALL 接受与应用身份接口一致的可选运行时 API 版本选择。SDK SHALL 将其内部 Supabase 表客户端生成的请求转换到该公开路径；浏览器不需要知道内部的 `/rest/v1` 路径。显式版本固定对外支持的表操作及错误语义。任何版本 MUST 继续逐次执行当前应用、用户、匿名开关、表和行权限检查。旧 `/rest/v1/{table}` SHALL 不再作为公开表接口，也不得回退为静态页面。

#### Scenario: Pinned table request
- **WHEN** 已获准用户通过 SDK 请求 `/_cellapp/db/notes?v=1.1` 或通过等价日期别名请求 `/_cellapp/db/notes?v=261022`，并执行该契约支持的表操作
- **THEN** 请求按该契约处理，并仍受当前表策略和行策略限制

#### Scenario: Anonymous or revoked request under an older contract
- **WHEN** 匿名访问开关被关闭或用户访问权被撤销，而客户端仍使用受支持的旧 API 版本
- **THEN** 后续数据请求按当前权限拒绝，不恢复历史授权
