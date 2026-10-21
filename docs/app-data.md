# 应用数据与表策略

Cellapp 为每个应用在独立的应用数据 PostgreSQL 数据库中建立独立 schema 与角色。控制库保存平台用户、应用、发布及授权关系，应用数据库保存应用自己的表。应用只上传前端代码；本阶段不能部署自定义服务端，也不能从前端直连 PostgreSQL。应用域 `/rest/v1/{table}` 使用真实 PostgREST 执行受限的 Supabase 表级操作。

## 创建与变更表

在应用项目根目录放置 `cellapp.data.json`，参考 [notes 示例](../skills/skills/cellapp-deploy/examples/notes/cellapp.data.json)。首次版本为 `1`，每次变更加一。`cellapp data plan --project .` 预览，`cellapp data apply --project .` 提交，`cellapp data status --project .` 查看账本，`cellapp data verify --project .` 检查真实数据 API 缓存。表版本与静态发布分别提交；超时后先检查 status/verify，再以完全相同的 manifest 重试。同一版本内容改变会拒绝。

表名、列名必须是小写 ASCII 字母起始，后接小写字母、数字或下划线，最长 63 字符。平台控制库与应用数据库分离，每个应用又有独立 schema，因此不同应用可以同时使用 `users`、`applications` 等表名；应用的 `users` 只是应用表，不是平台用户表，也不会自动获得平台用户资料。同一应用内的表名必须唯一；`rpc` 占用 PostgREST 路由，不能用作表名。PostgreSQL 系统列名和与自动生成的索引名冲突的名称在 plan 阶段拒绝。每应用最多 20 张表，每表最多 30 列、10 个索引。类型只支持 `text`、`uuid`、`bigint`、`boolean`、`timestamptz`、`jsonb`；主键不能为空。新增非空列到已有数据表可能失败并回滚，先使用可空列。主键、已有列类型和可空性不能变更。删除表或列属于破坏性操作，manifest 需设置 `allowDestructive: true`，提交时还需 `--allow-destructive`；先审查 plan 与备份。

每张表的 `policies` 分别指定 `select`、`insert`、`update`、`delete`，默认 `none`；可用 `all-admitted-users` 或 `own-rows`。`own-rows` 需要不可空的 text `ownerColumn`，其值来自同源 `/_cellapp/me` 的应用内用户 ID。Cellapp 同时使用 PostgreSQL 表授权与行级安全策略，对 `update` 的旧行和新行分别约束；未授权操作不能通过另一个 Supabase 客户端或伪造 profile 绕过。不要把平台用户 ID、访问密钥或数据库凭证写入应用表 manifest。

## 前端调用与限制

[完整静态示例](../skills/skills/cellapp-deploy/examples/notes/README.md)使用 `@supabase/supabase-js@2.117.3`、当前应用 origin 和公开标识 `cellapp-public`。此标识不是密钥。当前支持 `.from(table).select()`、`.insert()`、`.update()`、`.delete()`、`.eq()`、`.order()`、`.range()` 和写操作后 `.select()` 返回行；其他 Supabase Auth、Realtime、Storage、RPC、函数、关联查询和任意过滤器不在本期范围内。应用调用 `/_cellapp/me` 获取应用内用户 ID；未登录为 401，受限应用未获准为 403，数据服务未就绪为 503。

单请求体最多 1 MiB、单响应最多 2 MiB，PostgREST 单次最多 1000 行，应用数据请求 10 秒超时，数据库语句 5 秒超时，锁等待 1 秒；所有者表变更总计 30 秒、单语句 10 秒、锁等待 2 秒。应用存储默认上限 50,000,000 字节，由数据库事务账本约束。达到限制时客户端必须显示错误并缩小查询或联系管理员；不要在浏览器中无限重试。生产部署、凭证、备份和删除清理见 [应用数据运维](app-data-operations.md)。

匿名访问需要应用公开并分别开启页面、读取或提交；表清单还须将 `policies.anonymousSelect` 或 `policies.anonymousInsert` 明确设为 `all`。匿名提交另受运维开关控制，缺省关闭。未登录仅支持 `GET`、`HEAD` 和 `POST` 新增，`PATCH`、`DELETE` 要求登录。仅允许新增的表会强制最小响应，不返回新增行。匿名读取每来源每应用每分钟限 60 次，写入限 20 次，同时最多 8 个匿名数据请求；超限返回 `429`，请求体过大返回 `413`，来源不符返回 `403`。未登录请求不写入按用户计数的月访客表；IP 仅用于限流，不作为访客身份或独立访客数。
