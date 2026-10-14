# 运维与上线验收

## 生产配置

从 `.env.example` 创建自己的配置，设置 `NODE_ENV=production` 和 `AUTH_MODE=github`，替换 GitHub OAuth、S3、数据库、SECRET 和域名。GitHub OAuth App 的 callback URL 必须精确设置为 `${CONTROL_ORIGIN}/auth/callback`。`AUTH_MODE=dev` 只供 localhost 非生产开发，生产或非本地控制域会拒绝启动。服务启动时检查必填配置和正数额度；SECRET 必须稳定保存，变更会使所有摘要凭证不可用。

控制域和应用域必须使用不同注册域，例如 `console.cellapp.ai` 与 `*.ce11.app`；生产环境设置 `APPS_DOMAIN=ce11.app`，并为泛域名配置 DNS 与 TLS 证书。应用主机名使用 `{应用slug}-{用户slug}.ce11.app`，保留的旧 ID 域名继续解析。精确 Host 路由隔离应用；反向代理终止 TLS 并保留原始 Host，不将应用内容与控制面共享缓存。后端端口仅暴露给代理，不能开放给公网。网关响应为 `private, no-store`。

对象 bucket 必须私有。S3 使用通用配置变量，生产显式注入 `S3_ACCESS_KEY_ID` 和 `S3_SECRET_ACCESS_KEY`，Endpoint 必须为 HTTPS；服务只探测配置的 Bucket，不自动创建生产 Bucket。本地 MinIO、凭证兼容和已有后端升级见 [S3 存储配置](s3-storage.md)。运行匿名 GET 验收时应返回拒绝，不得返回对象内容；仅服务端具备读写权限。不要在客户端发放对象读取签名 URL。对象上传使用有界临时文件，配置足够的临时磁盘并限制并发入口，上传失败不会发布版本。

额度涵盖应用数、文件数、单文件/部署大小、全部暂存和保留版本占用、每应用 UTC 月流量。`UPLOAD_TTL_SECONDS` 和 `RETENTION_SECONDS` 控制回收期限。免费额度耗尽返回错误，不收费。具体数值必须在上线前结合预算配置。

应用数据使用独立 PostgreSQL 数据库，角色与备份流程见 [应用数据数据库运维](app-data-operations.md)。

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

定期使用 PostgreSQL 的备份工具分别导出控制数据库和应用数据数据库，并备份私有对象 bucket；三者必须覆盖同一个可恢复时间范围。秘密配置单独备份，不放仓库。恢复演练确认 active_deployment 指向的对象完整、密钥摘要仍能用原 SECRET 验证。

升级前先备份并应用向后兼容迁移，再替换服务镜像/二进制。服务回滚使用上一版本，保留数据库和对象。不要用 `docker compose down -v` 作为回滚，这会删除持久数据。本期没有用户可操作的版本回滚界面。

## 上线验收

- 使用真实 GitHub OAuth 在浏览器授权 CLI，确认账号绑定使用 GitHub 数字用户 ID，拒绝和过期请求不能领取 token。
- 发布 HTML 和 SPA，分别用两个浏览器输入密钥；深层页面可访问，缺失资源为 404。
- Cloudflare 边缘入口下，访问一个已发布的真实访客域名，确认 `/auth/app` 的 `host` 为该应用标签，登录后回到原域名，页面与同源数据写入可用；不同访客域的写入仍被拒绝。
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

## Cloud Run 部署

### Cloudflare 直连模式

如需由 Cloudflare 为 `console.cellapp.ai` 和 `*.ce11.app` 提供 HTTPS，并绕开 Google HTTPS 负载均衡，在 `infra/cloudrun.json` 中增加：

```json
"cloudflare_edge": {
  "origin_host": "cellapp-nok7htj6gq-uw.a.run.app",
  "secret": "cellapp-edge-origin-secret:1"
}
```

`origin_host` 是该 Cloud Run 服务的稳定 `run.app` 主机名；`secret` 是只授予服务运行账号读取权限的 Secret Manager `name:version` 引用。为 Cloudflare Worker 单独生成随机密钥，并将同一个值设置为 Worker 的 `EDGE_ORIGIN_SECRET` secret binding。不要把密钥值写入配置、仓库或 Worker 源码。Worker 源码见 [cloudflare-worker.mjs](../infra/cloudflare-worker.mjs)，需要配置 `CONTROL_HOST`、`APPS_DOMAIN`、`ORIGIN_URL` 和 `EDGE_ORIGIN_SECRET`；`PROBE_HOST` 只用于临时 workers.dev 测试。

此模式下发布脚本将服务 ingress 设为 `all`，因为 Cloudflare Worker 从公网访问 `run.app`。Go 服务仅在请求的 Host 等于配置的 `origin_host` 且两个专用请求头的密钥匹配时，才接受 `X-Cellapp-Original-Host`；其他直接访问保持原有未知域名响应。Cloudflare 的 DNS 记录要开启代理，并为 `console.cellapp.ai/*` 与 `*.ce11.app/*` 安装 Worker 路由。正式 Worker 名为 `cellapp-edge`；由于它自行请求 `run.app`，这两条代理 A 记录可使用 Cloudflare 推荐的保留占位地址 `192.0.2.0`，不依赖 Google 负载均衡 IP。确保 Cloudflare 安全设置不会对正常访客持续发起人机验证。Cloudflare 免费 Workers 有每日请求额度，达到额度后的行为应按所选套餐设置与监控；它仍产生 Cloud Run、出网、数据库和存储费用。

初次切换时先用独立测试 Worker 和服务验证 `/login` 返回 200、未知访客域返回 404 且带 `X-Request-ID`，并确认直接 `run.app` 访问无法作为有效控制域。随后打开 DNS 代理，检查两个真实域名的 HTTPS 证书、GitHub 登录与访客资源。若 Google 负载均衡尚在，可将 DNS 改回仅 DNS 作为临时回退。当前环境已移除负载均衡；后续应用回滚应保留 Cloudflare Worker、代理 DNS 和专用回源密钥，将正式 Cloud Run 服务流量切回兼容该回源协议的旧修订版本。不能把 DNS 改回已释放的旧 IP。

根目录 `Dockerfile` 从同一提交构建 Go server 与 Web console，并把 Caddy 放在单一运行镜像中。容器内 Go 只监听 loopback，Caddy 监听 Cloud Run 的 `$PORT`；外部 HTTPS 由 Google 负载均衡或 Cloudflare 终止，生产不使用本地 `tls internal`。未配置 `cloudflare_edge` 时，Cloud Run service ingress 为 `internal-and-cloud-load-balancing`，控制域和访客域的 Host 原样转发给 Go。当前按直接连接地址限流，代理后的请求可能共享来源配额；上线前按实际流量评估，不因转发头而放宽授权。

一次命令只发布应用和维护 job。提前准备并验证：

- Google Cloud 项目、启用的 Cloud Build/Cloud Run/Artifact Registry/Secret Manager API、镜像仓库、可部署的服务账号，以及同区域的 Direct VPC 网络与子网，或 VPC connector。仅使用 Cloud SQL 时才配置 `cloudsql_instance`。
- 已迁移或新建的 PostgreSQL、私有 S3 Bucket 和 HTTPS S3 Endpoint；运行身份能访问它们。应用数据使用独立数据库，按[应用数据运维说明](app-data-operations.md#生产准备)初始化，并为 provisioner 和 PostgREST authenticator 设置不同的生产密码。Secret Manager 中保存 `SECRET`、`DATABASE_URL`、`APP_DATA_DATABASE_URL`、`PGRST_DB_URI`、应用数据 JWT、GitHub OAuth secret 和 S3 凭证；服务账号仅获所需 Secret 的读取权限。秘密值不写入配置 JSON、镜像或构建参数。
- GitHub OAuth App 的 callback URL 精确等于 `${CONTROL_ORIGIN}/auth/callback`。`CONTROL_ORIGIN` 为 HTTPS 控制域，`APPS_DOMAIN` 为另一个注册域的访客基域，生产 `APP_PORT` 为空。
- 已配置的 HTTPS 入口：Google 全局外部负载均衡及控制域/访客通配证书，或上文的 Cloudflare Worker、代理 DNS 与专用回源密钥。Cloud Run 原生域名映射不支持所需通配证书；默认 `run.app` 地址不作为产品入口。
- 已启用的 Cloud Scheduler HTTP job，名称和区域见配置文件，目标为 `https://run.googleapis.com/v2/projects/PROJECT/locations/REGION/jobs/SERVICE-cleanup:run`，用具备 Cloud Run job 执行权限的身份调用。Cloud Run 会缩到零，服务内每分钟清理仅作补充；定时 job 是生产清理入口。

复制 [非秘密配置示例](../infra/cloudrun.example.json) 到 `infra/cloudrun.json` 并填写实际资源名称、生产额度/保留期限及 Secret Manager `name:version` 引用；上线前按预算核定额度，生产缺少任一项会拒绝启动。该本地文件已从 Git、Docker 和 Cloud Build 上传上下文排除。当前环境还必须填入上文的 `cloudflare_edge`；脚本发现已有服务使用公开 ingress 但配置缺少此项时，会在构建前拒绝发布，防止旧本地配置将入口改回已删除的负载均衡。Cloud Run 镜像内的 PostgREST 只监听 `127.0.0.1:3001`；`APP_DATA_JWT_SECRET` 与 `PGRST_JWT_SECRET` 必须引用同一 Secret 版本，且不得与平台 `SECRET` 相同。使用具备 Cloud Build、Artifact Registry、Cloud Run、Secret Manager、Cloud Scheduler 和相关资源读取/部署权限的 gcloud 身份，并使用支持 `gcloud run jobs deploy` 的新版 Google Cloud CLI。运行：

示例配置面向俄勒冈 `us-west1` 的验证/测试环境，未配置 Cloud SQL。自建 PostgreSQL 应部署在同一区域，通过私有地址供 Cloud Run 服务和迁移 job 访问；`DATABASE_URL` 在 Secret Manager 中保存，配置文件只引用秘密版本。示例使用 Direct VPC egress 的 `network` 与 `subnet`，无需持续计费的 VPC connector；若使用已有 connector，改为只填写 `vpc_connector`。`s3_region` 取决于实际对象存储区域，无须与 Cloud Run 区域相同。

Google Cloud Storage 可通过 XML API 与现有 S3 客户端兼容：`s3_endpoint` 使用 `https://storage.googleapis.com`，`s3_region` 使用 Bucket 的单区域位置（本环境为 `us-west1`）。为专用服务账号创建 HMAC 凭证，将访问 ID 和密钥分别存入 Secret Manager；该账号只授予目标私有 Bucket 的对象读写删除权限。Bucket 启用统一访问控制和公开访问防护。HMAC 密钥只在创建时可见，丢失后需重新生成；不要放进配置 JSON、镜像或日志。

```sh
python3 scripts/deploy-cloud-run.py --config infra/cloudrun.json
```

脚本以提交 SHA 和时间标记镜像；未提交源码会提示，生产建议在干净 Git 检出上运行。构建完成后记录不可变 digest 作为实际交付标识。它先更新并执行 `SERVICE-migrate` job，成功后更新 `SERVICE-cleanup` job，再以不接流量方式部署服务 revision。Cloudflare 模式会给候选修订版本加 0% 流量标签，确认公开 `/health` 和 Cloud Run 实际健康状态后切流量；两种模式都用 HTTPS 检查控制域 `/login` 和无效访客子域的路由。失败时脚本会标记部署未完成；查看对应 Cloud Build、Cloud Run job execution 和 revision，排除原因后重试。迁移成功但后续失败时数据库可能已升级，不应回滚数据库或删除对象；数据库变更必须保持与上一服务版本兼容。

切流量前记下旧 revision。若新 revision 上线后验收失败，将服务流量切回旧 revision（替换示例值），保留数据库、对象和 Secret：

```sh
gcloud run services update-traffic cellapp --to-revisions=OLD_REVISION=100 --project=PROJECT --region=REGION
```

发布后依照本文件“上线验收”执行真实 GitHub 登录、CLI 授权、应用发布、有效/无效访客域、未知 Host、未授权页面/资源和匿名 S3 读取拒绝检查；同时检查清理 job 的定时执行结果。仅 Docker 构建或脚本替身通过不代表已完成公网验收。
