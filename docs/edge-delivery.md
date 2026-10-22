# 边缘静态分发与回滚

访客静态 GET/HEAD 在独立候选环境验收后由 Worker 和私有 R2 提供；登录、授权变更和 `/_cellapp/*`（包括 `/_cellapp/db/{table}`）仍进 Cloud Run。根域名的 Dashboard 与控制 API 使用同一 Worker 的独立路径路由，公开官网仍由 Pages 提供；根域名切换见[切换清单](root-domain-cutover.md)。以下历史记录描述 2026-10-14 的访客边缘交付，不代表根域名迁移已上线。

## 内部授权协议

Worker 将 `GET /_cellapp/edge-grant` 发往 Cloud Run，并携带既有 `X-Cellapp-Original-Host`、`X-Cellapp-Edge-Secret` 和访客域专属 `__Host-cellapp` 会话 Cookie。该路径不向浏览器开放；Go 仅接受来自配置的回源 Host 和有效回源密钥。成功响应包含签名授权，Worker 作为 `__Host-cellapp-grant` Cookie 设置 `Secure; HttpOnly; SameSite=Lax; Path=/`，有效期最多 60 秒，不延长应用会话。

Ed25519 授权绑定精确 Host、应用 ID、发布版本、SPA 标志、应用会话 Cookie 摘要、签发与到期时间。Worker 每次静态请求先验签、核对 Host、会话和期限，再读取私有 R2。到期后的下一次请求从 Cloud Run 无感续签；同时加载的资源允许产生重复续签请求，数据库访客准入是幂等的。无会话的页面导航沿用 Cloud Run 登录跳转，有会话但无权限的页面沿用申请访问页面。数据 API 每次仍由 Cloud Run 检查当前权限。撤销、停用、删除或换版后，既有静态授权最多再生效 60 秒；正在传输的响应不强行中断。

| 情况 | 结果 |
| --- | --- |
| 无会话 | `401 login_required`；页面导航恢复到登录流程 |
| 无访问权、停用、删除或未发布 | `403 access_required/app_suspended` 或 `404 app_not_found/not_published` |
| 新访客达到每应用每 UTC 月上限 | `429 visitor_limit`，附下月 UTC 首日 `resetAt` |
| 授权依赖、R2 或回源不可用 | `503`，拒绝受保护内容 |
| 源速率限制 | `429`，不读取私有对象 |

R2 对象沿用 `releases/<deployment>/<SHA-256(path)>` 布局；同版本新增私有 `_cellapp_manifest.json`，列出实际发布文件。Worker 只读取清单列出的文件；GET、HEAD、条件请求和 SPA 页面回退都在验权之后处理。响应始终为私有且不在浏览器缓存。Worker 当前不使用共享资源缓存，所以不存在缓存命中绕过验权的问题；每个资源请求仍计入 Workers 请求量。

## 资源与密钥准备

1. 在 Cloudflare 创建与 GCS 隔离的私有 R2 Bucket，不开启 `r2.dev` 或公开自定义域名。为 Go 的 S3 适配器创建只限此 Bucket 的读写凭证；`S3_ENDPOINT` 使用 R2 账户 HTTPS endpoint，`S3_REGION=auto`。Worker 通过 `R2_BUCKET` 绑定读取，无需 S3 凭证。
2. 在仓库外运行 `go run scripts/generate-edge-key.go /private/tmp/cellapp-edge-private /private/tmp/cellapp-edge-public`，把私钥文件作为 `EDGE_GRANT_SIGNING_KEY` 的 Secret Manager 版本上传，`EDGE_GRANT_KEY_ID` 为该版本的非秘密标识。仅把公钥的 base64url 文本配置到 Worker 的 `EDGE_GRANT_PUBLIC_KEYS` JSON。签名密钥、回源密钥和通用 `SECRET` 必须相互独立；不要把密钥值放进仓库、日志或聊天。
3. 根据 [Worker 配置样例](../infra/wrangler.edge.example.jsonc)绑定私有 R2、独立 `EDGE_RATE_LIMITER`、回源地址与公钥。将既有回源密钥作为 Worker secret binding 配置。样例的每来源每分钟 600 次限制是突发保护，不代替每应用 1,000 位月访客限制；Cloudflare 的限速计数按节点近似执行，见[官方说明](https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/)。
4. 在 Cloudflare 将正式 Worker 路由设为 **fail closed**。Free 套餐每日 10 万次请求且每次 10 ms CPU 上限，应监测用量与 `1102`、`1027` 错误；配额耗尽时宁可拒绝静态访问，也不能绕过 Worker。见[官方限制](https://developers.cloudflare.com/workers/platform/limits/)。

轮换签名密钥时，先让 Worker 同时接受旧、新公钥，再令 Cloud Run 用新私钥和新 key ID 签发。等待超过 60 秒且确认续签正常后移除旧公钥。回滚到旧服务前确认 Worker 仍接受旧服务签发的 key ID。回源密钥轮换也需先在两端重叠并验证候选链路。

## 迁移与候选验收

1. 暂停新建部署、上传和发布：在待部署的服务配置中设置 `publication_paused=true`，在正式 Worker 配置中设置 `PUBLICATION_PAUSED="true"`。两层都阻止发布 API，返回 `503 publication_paused`；控制台及数据 API 继续可用。整个回滚窗口保持 Worker 的暂停开关，旧 Cloud Run 修订版不识别服务端开关，单靠它无法阻止旧版发布。
2. 用 `go run ./apps/server/cmd/copy-releases` 从 GCS 复制当前和保留期内的已发布版本。工具需要 `DATABASE_URL`、`COPY_SOURCE_{ENDPOINT,REGION,BUCKET,ACCESS_KEY_ID,SECRET_ACCESS_KEY}`、`COPY_DEST_{ENDPOINT,REGION,BUCKET,ACCESS_KEY_ID,SECRET_ACCESS_KEY}` 与 `ROLLBACK_RETENTION_SECONDS`，后者不小于实际 `RETENTION_SECONDS`。工具按数据库清单核对源对象路径、长度和 SHA-256，逐个复制并复验目标；重复执行会跳过已验证对象。完成后再以 `--verify-only` 运行一次。任何缺失或摘要不一致均停止切换；测试只用隔离 Bucket，不触及生产对象。
3. 先参考 [Cloud Run 边缘配置样例](../infra/cloudrun.edge.example.json)部署候选修订版本，再将 Worker 候选版本挂在隔离测试路由。候选应配置上述 Secret Manager 私钥引用和相同的公钥 key ID；不要先切正式访客路由。按[部署运维说明](operations.md)检查候选健康、未知 Host 和回源证明。
4. 用真实测试账号验收首次登录、受限/公开应用、直接资源、并行资源、超过一分钟后的无感续签、数据读写、撤销后数据立即拒绝及静态访问 60 秒内终止、超限 `429`、未知 Host、匿名 R2 GET 拒绝、R2 故障和 Worker 限速。另用明确开启匿名页面的公开应用验证未登录深链接、资源读取、匿名授权续签以及关闭页面开关后最长 60 秒失效；未开启的旧应用仍须登录。确认静态资源未逐个调用 Cloud Run，`/_cellapp/db` 仍调用 Cloud Run。记录候选 Worker 版本、Cloud Run revision、签名 key ID、R2 Bucket 与镜像 digest。
5. 上述检查通过后才将访客路由切向候选 Worker；控制域先保持原路由也可独立验收。Cloudflare 正式 Worker 的配置文件应使用既有名称 `cellapp-edge`，并保留两条正式路由。部署前记录 `wrangler deployments list --name cellapp-edge` 的当前版本；用 `wrangler deploy --config <正式配置文件>` 切换（该命令立即让新版本服务正式路由）。监测 `grant_issued`、`grant_denied`、`visitor_limit`、`r2_error`、`edge_rate_limited` 日志事件，以及 Workers 每日请求、CPU、R2 读写、Cloud Run 请求和出网。事件不含 Cookie、签名或存储凭证。

回滚时先将**同一 Worker 代码**的 `EDGE_STATIC_MODE` 改为 `origin`，保持 `PUBLICATION_PAUSED="true"`，发布并确认静态 GET 走 GCS 回源、部署创建/上传/发布返回 `503 publication_paused`。再执行 `gcloud run services update-traffic cellapp --to-revisions=<旧 Cloud Run revision>=100 --project=<项目> --region=us-west1`。不要直接恢复不认识发布暂停开关的旧 Worker 版本；那会让旧服务继续写 GCS 而不写 R2。旧服务继续从未删除的 GCS 读对象。数据库迁移、GCS/R2 对象和 Secret 版本不随服务回滚。回滚结束后若重新允许发布，下次切向 R2 前必须重新复制并校验新增对象。只有确认不再需要旧版本后，才能规划 GCS 退役。Cloudflare 版本切换和绑定限制见[官方说明](https://developers.cloudflare.com/workers/versions-and-deployments/rollbacks/)。

## 本地验收记录（2026-10-14）

- **通过**：隔离 PostgreSQL 的 Go 全量 race 测试、旧版和新增边缘浏览器流程、14 项 Worker 测试、17 项公开客户端测试与类型检查、主仓库 `npm test`（含 8 组 Web 浏览器检查）、服务端 vet、主仓库构建、22 项部署脚本测试、OpenSpec 严格校验、两个边缘配置样例解析。浏览器测试使用真实 Go 服务、PostgreSQL、PostgREST、MinIO 与 Worker 代码，R2 绑定由私有内存测试替身提供；检查了并行资源不逐个调用 Cloud Run、数据请求仍回源、续签、撤销与停用后最多 60 秒的授权窗口。
- **真实 R2 通过**：使用已授权的 Wrangler 在 Cloudflare 账户创建独立私有 Bucket `cellapp-edge-acceptance-20261014`（WNAM、Standard）；远程写入、读回、SHA-256 校验、删除均通过，确认对象数归零。`r2.dev` 公开访问关闭且没有自定义域名；匿名 S3 GET 返回 `400 InvalidArgument: Authorization`，没有返回对象。无正式路由的短时候选 Worker 通过真实 R2 绑定读到测试对象并返回匹配摘要，预览已停止、测试对象已清理。Go `TestLiveS3` 随后以 `region=auto` 通过 Bucket 健康、签名上传、读取、匿名拒绝和删除后不可读验收；再次确认 Bucket 对象数为零。测试 Bucket 保留供后续候选验收。
- **失败**：首次在受限沙箱运行 `npm test` 时，Go 的 `httptest` 无权监听本机端口；在允许本机测试端口的环境中重跑后全部通过，无尚未解决的检查失败。
- **未运行**：实际 Go 授权端点与真实 R2 Worker 绑定组成的同一远程候选链路，以及生产 Cloud Run、对象复制、DNS/路由切换和用量观察。浏览器端到端验收使用私有 R2 绑定替身，真实 R2 的独立 Worker 绑定和 Go S3 适配器已分别通过；生产操作仍需要单独的发布请求及候选环境验收。
- **兼容性检查**：`monthlyVisitors` 替换公开限额里的 `trafficBytes`；历史 `usage.bytes` 数据保留但不再用于限流。新 Go 服务仍提供直接静态回源路径。回滚到旧 Go 修订版时，保持新版 Worker 的 `origin` 模式和 `PUBLICATION_PAUSED` 开关，防止旧版向 GCS 单边发布。

## 线上发布记录（2026-10-14）

- 在 `cellapp-511002/us-west1` 部署 Cloud Run 修订版 `cellapp-00016-c7k`（镜像摘要 `sha256:303e7ab9a4719378c98996f5226df45a2ae691064c5497622ec2e63f7a2e9009`），100% 流量；正式 Worker `cellapp-edge` 版本 `8617764f-d7c2-4c6d-bea4-c33c15336bd2` 使用 R2 模式。签名 key ID 为 `key-20261014`；发布暂停已解除。
- 私有 Bucket `cellapp-uswest-prod` 位于 WNAM、Standard。其专用凭证通过 Secret Manager 注入服务，不写入仓库。迁移作业 `cellapp-copy-releases-pzw7t` 复制并校验 1 个活跃发布版本；独立 `--verify-only` 执行 `cellapp-copy-releases-vqr8t` 再次通过，私有清单包含 2 个文件。旧 GCS 对象保留作回滚源。
- 先用独立 Worker 的单应用路由验收，再切正式通配路由并删除临时 Worker。已登录浏览器打开 `lean-trainer-wyx.ce11.app`，页面、`app.js` 和数据 API 正常；超过 60 秒后刷新仍正常。Cloud Run 记录到成功签发授权，浏览器的静态 `app.js` 请求未进入 Cloud Run。
- 真实公网检查：控制台登录页 `200`、未知访客 Host `404`、匿名静态资源 `401`、公开请求内部授权路径 `404`、无回源密钥的直连伪造 Host `404`、匿名 R2 对象请求未返回文件内容。暂停期间，发布写入被 Worker 拒绝为 `503`；解除暂停后只做了只读验收。
