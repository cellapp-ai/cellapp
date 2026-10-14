# 应用数据数据库运维

Cellapp 的平台控制数据继续由 `DATABASE_URL` 指向的库保存。应用表数据使用 `APP_DATA_DATABASE_URL` 指向的**另一个 PostgreSQL 数据库**，可以与控制库共享实例，但不能是同一个数据库。服务启动时核对连接目标；数据角色不能读取控制表，控制运行角色不取得应用表授权。生产请分别管理两套凭证，且不要把任何数据库 DSN 放进应用静态产物或 Skill 项目状态。

## 本地开发

新建本地 Compose 数据卷时，PostgreSQL 初始化钩子自动创建 `cellapp_data` 数据库，并执行 [`bootstrap.sql`](../apps/server/appdata/bootstrap.sql)。本地 provisioner 和 PostgREST authenticator 的密码均为公开的开发值 `development`；生产不得沿用。`.env.example` 的 `APP_DATA_DATABASE_URL` 使用 provisioner 角色。应用数据内部账本位于 `cellapp_internal.app_versions`，普通应用 API 角色没有访问权。

Compose 中的 PostgREST 16.3 只映射到本机回环地址 `127.0.0.1:3001`，Go 服务通过 `POSTGREST_URL` 访问。PostgREST 使用 authenticator 账号，`PGRST_DB_PRE_CONFIG=cellapp_internal.pre_config` 动态读取应用 schema；新应用创建后，数据库通知 PostgREST 重新载入配置和结构，真实探针通过后才把应用标为 ready。`APP_DATA_JWT_SECRET` 与 PostgREST 的 `PGRST_JWT_SECRET` 必须相同，并且与平台 `SECRET` 不同。它们均不得进入静态应用。

已存在的本地数据库卷不会再次运行初始化钩子。先确认当前实例是本项目的开发库、已备份并且不存在同名 `cellapp_data` 库，再执行下列可重试命令；它不会删除已有库或控制表：

```sh
docker compose -f infra/compose.yaml exec -T postgres sh /docker-entrypoint-initdb.d/10-appdata.sh
```

如果已有卷仍使用旧数据库管理员名称，先按 [命名升级说明](operations.md#命名升级)核对并调整实际管理员账号，不要为使脚本成功而重建数据卷。初始化后检查控制库与应用库是两个数据库，并用应用角色验证无法读取控制表。开发容器仅绑定 loopback PostgreSQL 端口。

## 生产准备

以数据库管理员创建独立应用数据库，连接到**应用数据库**执行 `apps/server/appdata/bootstrap.sql`。脚本幂等创建 `cellapp_appdata_provisioner`、`cellapp_appdata_authenticator` 和内部账本，不包含密码。通过受保护的管理员会话分别使用 `\password` 设置高熵密码，再把 provisioner DSN 作为 `APP_DATA_DATABASE_URL` 注入 Go 服务；PostgREST 使用独立 authenticator DSN，且其 HTTP 端口只允许 Go 服务访问。不要让这两个角色拥有控制库表权限，也不要把 provisioner、authenticator 或数据库管理员密码交给应用前端。

PostgREST 需设置 `PGRST_DB_SCHEMAS=cellapp_api`、`PGRST_DB_PRE_CONFIG=cellapp_internal.pre_config`、`PGRST_DB_ANON_ROLE=cellapp_appdata_anon`、`PGRST_OPENAPI_MODE=disabled` 和专用的 `PGRST_JWT_SECRET`。Go 进程配置相同的 `APP_DATA_JWT_SECRET` 与内部 `POSTGREST_URL`。应用 schema 由数据库预配置函数动态列出，不能靠公开客户端提交的 profile 扩大可见范围；所有者还需要执行真实 API 探针。生产升级 `bootstrap.sql` 后，应重新执行该幂等脚本，并确认 PostgREST 配置和 schema 缓存已重载。

上线前核对：两个 DSN 连接的数据库名称不同；应用角色只能访问本应用 schema；新建表在授权前不可读写；PostgREST 的内部接口无法从公网或应用脚本直接连通。正常服务的 `/health` 会探测控制库、应用数据连接和对象存储；表结构/API 缓存未就绪的应用保持 pending/error，不能报告为可发布。应用创建和删除跨两个数据库，失败由后台幂等重试，不通过删除数据库卷恢复。

若首次部署返回 `app_data_not_ready`，先保留生成的 `cellapp.json` 和 CLI 私有凭证目录，不要换 slug 或删除应用。执行 `cellapp data status --project <项目目录>`，查看 `state` 和 `errorCode`；确认应用数据库、PostgREST 与 JWT 配置恢复后，执行 `cellapp data reconcile --project <项目目录>`，再重试原部署命令。服务端的 `app_data_reconcile_failed` 日志按 `app_id` 记录底层错误；API 只向所有者返回稳定错误码，不返回 SQL 或数据库凭证。旧版 CLI 若在失败后没有写入 `cellapp.json`，可从 `cellapp apps` 确认所属应用 ID，再执行 `cellapp link <app-id> --project <项目目录> --output <静态产物目录>`。若初始访问密钥已经遗失，所有者可显式执行 `cellapp reset-key <app-id>`；这不会撤销既有访客授权。

## 备份、恢复与回滚

分别备份控制库、应用数据数据库及私有对象存储，并保留三者同一可恢复时间范围的快照。恢复后核对控制库中未删除的应用与数据 schema、角色、版本账本、发布对象相互匹配；缺失时先拒绝该应用访问并执行安全的重新协调。删除应用先在控制库阻断访问，再异步清理对应 schema，数据库暂时不可用时保留清理状态。回滚服务版本时保留两个数据库及其数据，不执行 `docker compose down -v`，也不把新登录与用户授权流程退回匿名密钥会话。

应用命名空间在控制库增量迁移 `004_app_namespaces.sql` 中回填。每个既有所有者获得独立随机空间名，既有应用得到 `app-<原应用 ID>` 别名；原 ID 域名、发布和应用数据库 schema 不变。别名记录包含已删除应用，恢复数据时须与应用表保持同一控制库快照，以免旧分享链接被重新分配。
