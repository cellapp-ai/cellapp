# Cloud Run 部署变更验收记录

变更：`add-cloud-run-deployment`。以下按日期记录执行结果；一键发布、两个自定义域名路由及 GitHub 登录已通过，定时清理实际执行与隔离浏览器联调仍待验收。

## 2026-10-13：us-west1 云端准备

| 检查 | 结果 |
| --- | --- |
| Artifact Registry | 在 `cellapp-511002/us-west1` 创建 Docker 仓库 `cellapp-images`。 |
| Cloud Build 镜像 | 构建 `bcd1bc12-ecb8-445b-a244-0f1ee9bebb97`、`8c69d433-cd2f-4e2b-8150-cde756cca0e7`、`23694bc0-fcfd-4208-8dba-fbc3de8bcac1`、`1584685c-310b-4b36-873a-8ad346c4e4cc` 和 `a278ccb0-a976-4be3-8250-3c9c2052ffd3` 成功。当前 Cloud Run 修订版本使用不可变 digest `sha256:debe7102e3e9bf34bb90bbadefcb91ffabb6762f1e83d684394d2c8b5e9ce888`。构建来源是本工作树的未提交内容。 |
| 运行身份和数据库 Secret | 创建 `cellapp-run@cellapp-511002.iam.gserviceaccount.com`，仅对所需 Secret 授予读取权限。数据库 Secret 版本 1 尾部有换行，首次迁移因此失败；保留旧版并新增去除换行的 `cellapp-database-url:2`，迁移执行 `cellapp-migrate-g48q7` 成功。 |
| 数据库内网连通 | 临时 Cloud Run Job `cellapp-db-network-check` 使用 Direct VPC (`cellapp-test` / `cellapp-test-us-west1`) 连接 `10.92.0.2:5432` 成功；执行 `cellapp-db-network-check-gnlxg` 成功后已删除临时 Job。这只证明端口可达，不证明应用已完成数据库登录或迁移。 |
| 私有对象存储 | 创建 `gs://cellapp-511002-artifacts-us-west1`，单区域 `US-WEST1`、Standard、统一存储桶级访问、公开访问防护。保留默认 7 天软删除；匿名访问返回 HTTP 403。专用服务账号 `cellapp-storage@cellapp-511002.iam.gserviceaccount.com` 仅获该桶的 `roles/storage.objectAdmin`。HMAC 凭证直接写入 `cellapp-s3-access-key:1` 与 `cellapp-s3-secret-key:1`，两个版本均已启用，Cloud Run 运行身份可读取。使用 Cellapp 实际 S3 适配器完成 Bucket 健康检查、上传、读取和删除，全部通过；测试对象已删除。 |
| 应用与 GitHub Secret | 创建稳定的 `cellapp-secret:1`；项目已有 `cellapp-github-oauth-Ov23limVEZjXdRicZdzw:1`。两项均已授权 Cloud Run 运行身份读取，秘密值未写入配置或日志。 |
| 部署脚本 | 增加 Direct VPC 网络和子网配置及预检；允许现有 GitHub Secret 名称中的大写 Client ID；首次创建 Cloud Run 服务时不使用不支持的 `--no-traffic`，已有服务升级仍保留先检查后切流量。15 项单元测试通过，`git diff --check` 通过。 |
| 域名与配置 | 本地配置为 `https://console.cellapp.ai` 和访客基域 `ce11.app`；GitHub Client ID 根据现有 Secret 名称填写。配置格式与云端资源只读预检通过。控制域和访客域各有 DNS 授权、Google 托管证书及证书映射。用户先指定 `cellapp.me`，随后改为 `ce11.app`；旧访客域的证书映射、证书及 DNS 授权均已删除。用户将四条 Cloudflare 记录从代理改为仅 DNS 后，权威与公开递归 DNS 均返回负载均衡 IP 和预期证书 CNAME。控制域证书 `cellapp-console-cert` 和访客通配证书 `cellapp-ce11-wildcard-cert-v2` 均为 `ACTIVE`；旧访客证书未使用且已删除。用户选择先保持 Google 托管证书直连。公开 `/auth/login` 返回 302，GitHub Client ID 和回调 `https://console.cellapp.ai/auth/callback` 正确；用户通过真实 GitHub 授权后看到控制台（用户现场反馈，非自动化浏览器日志）。为未来落地页另建覆盖 `cellapp.ai` 与 `*.cellapp.ai` 的证书，并在 DNS 授权 CNAME 生效后另建第二张触发签发；两张仍在签发，现有控制台证书和流量不受影响。 |
| 公开 HTTPS 入口 | 创建全局外部 Application Load Balancer，IP `34.120.142.70`，443 转发规则 `cellapp-public-https`、证书映射 `cellapp-public-cert-map`、后端服务 `cellapp-public-backend` 和 us-west1 serverless NEG `cellapp-run-neg`；NEG 已连接 Cloud Run 服务。控制域真实 HTTPS `/login` 返回 200、证书校验通过并带 `X-Request-ID`。真实 `https://cellapp-invalid-probe.ce11.app/` 通过通配证书校验，返回预期 404、`Cache-Control: private, no-store` 和 `X-Request-ID`。 |
| 定时清理 | 启用 Cloud Scheduler API，创建每日 04:00 UTC 的 `cellapp-cleanup-trigger`，使用专用服务账号 OAuth 调用 `cellapp-cleanup:run`；账号仅对该 Job 获 `roles/run.invoker`。定时任务已启用，尚无实际成功执行记录。手动立即触发被自动审批拒绝，因为可能删除过期产物；保留既定计划。 |
| Cloud Run 服务 | 首次部署脚本在迁移成功后因 `--no-traffic` 不支持新服务而停止；修复脚本后，用同一镜像 digest 创建初始服务。访客域变更后修订版本 `cellapp-00002-clm` 已就绪。完整部署命令后来构建新镜像、成功执行迁移 `cellapp-migrate-4ftm2` 并发布修订版本 `cellapp-00003-jtz`；命令在最后的域名检查阶段返回非零，因为当时证书尚未分发。两张证书激活后重跑完整部署命令，构建成功、迁移 `cellapp-migrate-5lzfv` 成功、修订版本 `cellapp-00004-9m2` 已就绪并接收 100% 流量，控制域和访客域检查通过，命令退出 0。`APPS_DOMAIN=ce11.app`，ingress 限制为负载均衡入口。 |

本地忽略的 `infra/cloudrun.json` 已填入项目、区域、镜像仓库、运行身份、Direct VPC、私有 GCS、五项 Secret 引用和实际域名。控制域和无效访客域的真实 HTTPS 路由、用户 GitHub 登录已通过；仍需执行有效访客域、定时清理实际执行及回滚验收。镜像构建、迁移成功、服务就绪和存储读写检查不能替代这些检查。

## 2026-10-08：本地验收

| 检查 | 本轮结果 |
| --- | --- |
| `docker build --pull=false -t cellapp-cloudrun:local .` | 通过。Go server、Web console、Caddy 和监督入口均进入最终镜像。最终镜像以 `cellapp` 非 root 用户运行，仅声明 8080/TCP。 |
| 最终镜像文件与构建历史检查 | 通过。存在 server、入口、Web index/assets；未发现源码、`skills/`、`.env`、npm 缓存或秘密构建参数。该检查不能证明任意二进制中不存在秘密，构建过程未传秘密。 |
| 本地 Docker + 隔离 PostgreSQL 路由 | 通过。控制域 `/login` 为 200；无效访客域和未知 Host 为 404；响应含 `X-Request-ID`；Go 监听 `127.0.0.1:3000`，Caddy 监听 8080。使用示例外部 S3 配置，未验收真实对象读写。 |
| 镜像失败路径 | 通过。无效 `PORT`、缺少生产配置或缺少 Web 产物均非零退出；Go 提前退出后监督入口能终止 Caddy，未打印秘密值。 |
| 部署脚本检查 | 当日 12 项通过，覆盖配置/秘密引用、资源和定时触发器预检、digest 固定、迁移失败阻断服务发布、未就绪修订版本保留旧流量、域名错误不得报告成功。使用命令替身，未调用真实 Cloud Run。 |
| `npm run typecheck`、`npm test`、`npm run build`、`go vet ./apps/server/...` | 通过；客户端子模块固定在主仓库记录的提交，未修改子仓库。 |
| `npm test --workspace @cellapp/web` | 通过，Web 浏览器检查 6 组。 |
| 隔离 PostgreSQL 的 `go test -race -count=1 ./apps/server/...` | 通过，包含服务端数据库集成检查。 |
| `RUN_BROWSER_TESTS=1 ... TestBrowserEndToEnd` | 失败：macOS 拒绝临时测试 CA 授信，提示 `SecTrustSettingsSetTrustSettings: The authorization was denied since no user interaction was possible.`；测试在浏览器流程前终止，不能记为浏览器联调通过。 |
| 真实 Google Cloud 构建、迁移、域名/OAuth/S3、回滚 | 当日未执行。彼时无本变更的隔离 Google Cloud 配置；Google Cloud SDK 为 412.0.0。2026-10-13 已升级到 588.0.0 并完成镜像构建；其余项目仍待验收。 |

上线前仍需完成有效访客域、定时清理实际执行及回滚验收，并在可授信测试 CA 的浏览器环境重跑跨服务验收。真实迁移、服务发布、控制域/无效访客域、用户 GitHub 登录和私有 S3 读写已于 2026-10-13 通过；历史验证和命令替身结果不能替代其余检查。
