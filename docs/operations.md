# 运维与上线验收

## 生产配置

从 `.env.example` 创建自己的配置，设置 `NODE_ENV=production` 和 `AUTH_MODE=github`，替换 GitHub OAuth、S3、数据库、SECRET 和域名。GitHub OAuth App 的 callback URL 必须精确设置为 `${CONTROL_ORIGIN}/auth/callback`。`AUTH_MODE=dev` 只供 localhost 非生产开发，生产或非本地控制域会拒绝启动。服务启动时检查必填配置和正数额度；SECRET 必须稳定保存，变更会使所有摘要凭证不可用。

控制域和应用域必须使用不同注册域，例如 `control.example.com` 与 `*.example-apps.net`；精确 Host 路由隔离应用。反向代理终止 TLS，保留原始 Host，不将应用内容与控制面共享缓存。后端端口仅暴露给代理，不能开放给公网。网关响应为 `private, no-store`。

对象 bucket 必须私有。S3 使用通用配置变量，生产显式注入 `S3_ACCESS_KEY_ID` 和 `S3_SECRET_ACCESS_KEY`，Endpoint 必须为 HTTPS；服务只探测配置的 Bucket，不自动创建生产 Bucket。本地 MinIO、凭证兼容和已有后端升级见 [S3 存储配置](s3-storage.md)。运行匿名 GET 验收时应返回拒绝，不得返回对象内容；仅服务端具备读写权限。不要在客户端发放对象读取签名 URL。对象上传使用有界临时文件，配置足够的临时磁盘并限制并发入口，上传失败不会发布版本。

额度涵盖应用数、文件数、单文件/部署大小、全部暂存和保留版本占用、每应用 UTC 月流量。`UPLOAD_TTL_SECONDS` 和 `RETENTION_SECONDS` 控制回收期限。免费额度耗尽返回错误，不收费。具体数值必须在上线前结合预算配置。

## 命名升级

当前构建产物为配套的 `dist/cellapp-server` 和 `dist/web/`，新环境的数据库和角色均使用 `cellapp`。旧 CLI 别名、项目配置与凭证仍兼容；历史验收记录保留当时名称。

已有 PostgreSQL 数据卷不会因修改 `POSTGRES_USER`、`POSTGRES_DB` 自动改名。可以暂时保留原来的 `DATABASE_URL`，并在 Compose 中同时保留旧角色、库名及健康检查参数；或在停机并备份数据库及角色后原位升级。以下命令仅适用于 README 的本地开发实例；先确认不存在同名 `cellapp` 角色、数据库或 `cellapp_rename_admin` 临时角色，并停止服务端及其他旧数据库连接。角色不能由它自身的会话改名，以下步骤通过本地 Unix socket 使用临时管理员；将 `/secure/backup` 替换为已准备好的私有备份目录：

```sh
docker compose -f infra/compose.yaml exec -T postgres pg_dump -U ohmyapp -d ohmyapp -Fc > /secure/backup/cellapp-before-rename.dump
docker compose -f infra/compose.yaml exec -T postgres pg_dumpall -U ohmyapp --roles-only > /secure/backup/cellapp-before-rename-roles.sql
docker compose -f infra/compose.yaml exec -T postgres psql -U ohmyapp -d postgres -v ON_ERROR_STOP=1 -c 'CREATE ROLE cellapp_rename_admin LOGIN SUPERUSER;'
docker compose -f infra/compose.yaml exec -T postgres psql -U cellapp_rename_admin -d postgres -v ON_ERROR_STOP=1 -c 'ALTER DATABASE ohmyapp RENAME TO cellapp;'
docker compose -f infra/compose.yaml exec -T postgres psql -U cellapp_rename_admin -d postgres -v ON_ERROR_STOP=1 -c 'ALTER ROLE ohmyapp RENAME TO cellapp;'
```

角色 OID 和数据库内容保持不变。旧实例使用 README 中的开发密码；若实际使用 MD5 密码，角色改名会清空密码，需通过 `psql -U cellapp -d postgres` 的 `\password cellapp` 交互重设。随后更新自己的 `DATABASE_URL`、Compose 初始化变量和健康检查配置，启动新服务并验证登录与应用访问，最后使用 `psql -U cellapp -d postgres -c 'DROP ROLE cellapp_rename_admin;'` 删除临时管理员。不要删除数据卷或重新初始化。若中途失败，按数据库实际状态完成后续步骤或用备份恢复，不要直接重复全部命令。回滚时先停服务，通过另一个管理员会话在 `postgres` 库中反向改名数据库和角色，并恢复连接配置及必要密码；完成后删除临时管理员。

首次开发登录会将 `ohmyapp:development` 原位改为 `cellapp:development`，保留原账号 ID、应用、凭证与会话。若两种身份均已存在，返回 `development_identity_conflict`，需人工核对账号和资源，不能自动合并。回滚旧服务前，停止所有新旧服务，并在备份和确认没有冲突后将该固定账号的 issuer 原位改回旧值；不要同时运行使用不同开发 issuer 的服务。GitHub 所有者身份不受影响。

## 迁移、清理和停用

```sh
./dist/cellapp-server -migrate
./dist/cellapp-server -cleanup
./dist/cellapp-server -metrics
./dist/cellapp-server -suspend APP_ID
```

维护命令使用服务端数据库权限，应仅向运营人员开放。服务运行时每分钟清理一次；失败记录 `cleanup_failed`，修复依赖后可重试。`-metrics` 输出应用总数和已预留存储；数据库 `usage` 表提供按应用/月份流量明细。停用即时阻止新分发，并提升密钥代数，使已有会话失效。

请求日志只记录请求 ID、方法和耗时，不记录 URL 参数、请求正文、Authorization、Cookie 或分享密钥。用返回的 `X-Request-ID` 关联问题；需要详细诊断时应增加安全的错误码，而不能记录原始请求。

## 备份和回滚

定期使用 PostgreSQL 的备份工具导出控制数据库，并备份私有对象 bucket；两者必须覆盖同一个可恢复时间范围。秘密配置单独备份，不放仓库。恢复演练确认 active_deployment 指向的对象完整、密钥摘要仍能用原 SECRET 验证。

升级前先备份并应用向后兼容迁移，再替换服务镜像/二进制。服务回滚使用上一版本，保留数据库和对象。不要用 `docker compose down -v` 作为回滚，这会删除持久数据。本期没有用户可操作的版本回滚界面。

## 上线验收

- 使用真实 GitHub OAuth 在浏览器授权 CLI，确认账号绑定使用 GitHub 数字用户 ID，拒绝和过期请求不能领取 token。
- 发布 HTML 和 SPA，分别用两个浏览器输入密钥；深层页面可访问，缺失资源为 404。
- 未授权访问 HTML、JS、图片、HEAD、条件请求和对象地址均不能读取内容。
- 更新失败仍返回旧版本；重置后旧密钥/旧 cookie 不可读取；删除后立即拒绝分发。
- 检查跨账号管理拒绝、跨应用存储隔离、Service Worker 注册被拒绝。
- 在小额度配置下测试并发创建、上传及流量超限，并验证清理回收。
- 验证备份恢复、运营停用、Secret 脱敏及代理端口隔离。

以上真实环境验收未完成前，不应把单元测试通过解释为已可公网运营。

## Web 控制面交付

`npm run build` 生成服务端和完整 Web 静态产物。将两者放在同一版本的发布目录，正常服务显式设置 `WEB_ROOT=/absolute/release/web`；默认目录仅适用于仓库开发。启动时检查 index 和 assets，缺失则失败，不回退到旧首页。数据库迁移、清理、暂停和统计维护命令独立于静态目录。

控制域提供 `/login`、`/applications`、`/applications/<id>`、`/credentials` 和 `/device`，有效深链可刷新。`/api/console` 使用浏览器会话和严格 Origin，CLI `/apps` 等接口继续要求 Bearer。未知资源/API 返回 404，API 与 HTML 不缓存，带指纹资源可长期缓存。CSP 限制资源来自本站并禁止嵌入；不放宽访客网关边界。

部署后检查登录、应用列表、深链与资源加载，并回归 CLI 与访客域。回滚恢复配套的前一版可执行文件和 Web 目录，不恢复已重置的密钥、已撤销凭证或已删除应用。浏览器接口字段及失败恢复见 [Web API](web-api.md)，本轮验收见 [Web 控制面验收](web-control-plane-verification.md)。
