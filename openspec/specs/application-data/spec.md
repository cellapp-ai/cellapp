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
系统 SHALL 根据当前已认证且获准进入应用的用户执行逐表、逐操作和必要的逐行权限检查；表权限 MUST 区分应用所有者管理权与访客数据权，前端不得通过伪造用户 ID、请求头或字段提高权限。

#### Scenario: User-owned rows
- **WHEN** 表规则仅允许用户操作自己的记录，用户尝试查询或修改其他用户的记录
- **THEN** 其他用户的记录不可读取或修改

#### Scenario: Public application with private table
- **WHEN** 已登录用户可以进入公开应用，但表没有授予其读取权限
- **THEN** 数据接口仍拒绝读取，不以应用公开状态覆盖表权限

#### Scenario: Access revoked between requests
- **WHEN** 用户的应用访问权被撤销后再次调用数据接口
- **THEN** 后续请求不能继续读写，即使浏览器仍持有旧页面或会话

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
