# 控制 API 路径改造验收记录

本记录对应 OpenSpec `organize-control-api-paths`。本轮使用任务 worktree `codex/control-api-paths`，在隔离 PostgreSQL 17.6 容器 `cellapp-control-api-test-20261017` 上运行服务端集成检查；跨服务验收另用独立 Compose 项目 `cellapp-control-api-acceptance`，包含 PostgreSQL、PostgREST 16.3 与私有 MinIO 测试桶。服务端和 Worker 已于 2026-10-17 发布；npm 包按本轮决定暂不发布。

| 检查 | 结果 | 实际范围 |
| --- | --- | --- |
| Go 路由清单 | 通过 | 以公开 HTTP v2 的 17 个操作核对新旧路径均注册；v3 契约检查方法、成功状态、必要字段与错误 |
| Go 集成与 race | 通过 | `TEST_DATABASE_URL=<隔离 DSN> go test -race -count=1 ./apps/server/...`；设备待批准、限频、批准、拒绝、重复领取，凭证撤销，跨路径应用和部署幂等/冲突，Cookie/Bearer、跨所有者、应用域与未知路径隔离 |
| Worker | 通过 | `node --test infra/cloudflare-worker.test.mjs`，17 项；新旧路由、Wrangler 声明、未知 API 404、近似路径拒绝、新旧发布写入在暂停时均拒绝 |
| CLI 子仓库 | 通过 | `npm run typecheck`、`npm test`（22 项）、`npm run build`、`npm run check:skill`、`npm run check:boundary`、`npm run check:package`、`npm pack --dry-run --json`；v3 协议替身覆盖登录、应用、部署、数据、退出与失败恢复 |
| 主仓库 | 通过 | `npm run typecheck`、`npm test`、`npm run build`、`go vet ./apps/server/...`；Web 浏览器替身 8 组 |
| OpenSpec | 通过 | `openspec validate organize-control-api-paths --strict` |
| 独立 HTTPS 浏览器与真实对象存储 | 通过 | `RUN_BROWSER_TESTS=1 ... TEST_LEGACY_CLI_BIN=<旧版产物> go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting`；真实 Chrome、PostgreSQL、PostgREST 和私有 MinIO 测试桶，约 19 秒。新版 CLI 设备登录、应用创建、部署、数据状态和退出成功 |
| 旧版 CLI 可执行程序对隔离服务 | 通过 | 从固定的旧 skills 提交 `5031002` 构建 CLI；旧路径登录、应用读取、数据状态、再次部署及退出均成功。浏览器明确点击批准后才取得凭证 |
| 真实边缘到服务链路 | 部分通过 | 在 `cellapp.ai/api/control/v1*` 暂时挂载独立候选 Worker `cellapp-edge-api-candidate`，验证新路径到 Go 的 `401`、旧路径 `401`、未知与近似路径 `404`、错误凭证隔离、应用域 `404`；候选 Worker 版本 `987db640-b47e-4460-9772-46fa0790c994` 开启发布暂停后，伪应用 ID 的部署写入返回 `503 publication_paused`。候选 Worker 已清理 |

Go 隔离测试覆盖设备待批准、限频、拒绝、过期、重放和无效凭证；CLI 协议替身覆盖发布响应丢失后的同标识重试。真实 HTTPS 链路的强制网络超时未执行。首次拉取固定版 MinIO 镜像时 Docker Hub 超时，随后使用本机缓存的 Chainguard MinIO 与 MinIO Client 镜像，在新建 Compose 项目完成验收，没有连接已有生产资源。设备授权仍由浏览器明确批准；打开验证页或轮询不会自动批准。

## 线上发布与未完成验收（2026-10-17）

- Cloud Run `cellapp-511002/us-west1`：镜像 `sha256:419a0e65f0ba1609daf38df12cc291511d80ded6a678ef2de99890264aa65254`；迁移执行 `cellapp-migrate-vrj8p` 成功；修订版 `cellapp-00018-n8f` 通过候选健康检查并接收 100% 流量。上一服务修订版为 `cellapp-00017-p9s`。
- 正式 Worker `cellapp-edge`：版本 `694b3ade-2919-4dd5-9667-45f37f42557c` 接收 100% 流量；原有 R2、限流、回源 Secret 与 9 条路由保留，新增 `cellapp.ai/api/control/v1*`，10 条正式路由均已从 Cloudflare 控制面核对。上一版本为 `b443a8b6-5fcd-4da8-95d2-cb1e23788933`。
- 正式域名只读检查：`/dashboard/login` 返回 `200`；新旧 `/limits` 均返回相同的 `401 authorization_required` JSON；新路径无效 Cookie、控制台无效 Bearer 均返回 `401`；未知及近似新 API 路径、未知应用 Host 的控制 API 返回 `404`。网络偶发超时的探针已用可达 Cloudflare 地址重试；未将超时记为通过。
- 受控联调使用正式根域的**新路径**短暂挂候选 Worker，未获得独立候选 Host，也未使用有效线上 Cookie/Bearer 完成正向授权检查；旧路径的发布暂停仅在 Worker 本地测试中验证。OpenSpec 4.2 保持未完成；2026-10-17 按用户要求带此警告归档，未把缺口写成通过。线上测试没有写入生产应用或数据。
- 本次不发布 npm；`@cellapp/cli` 线上仍为 `0.2.1`，本地待发布版本为 `0.3.0`。公开 Skill 若先推送，其 `@latest` 命令仍运行旧 CLI，服务与 Worker 保留旧路径兼容。

## 后续 npm 发布（2026-10-17）

- 发布依据：主仓 `main` 为 `111906a`，固定 skills `main` 的 `c4d7610`；Cloud Run `cellapp-00018-n8f` 接收 100% 流量，正式 Worker `694b3ade-2919-4dd5-9667-45f37f42557c` 已提供新控制 API 路径。线上只读探针确认新路径未授权返回 `401 authorization_required`、未知新路径返回 `404`，旧路径未授权仍返回 `401`。IPv6 探针曾超时，改用 IPv4 后通过；没有将超时算作成功。
- 官网先发布 Cloudflare Pages 生产部署 `eb8258e7-f157-4ae4-bfc6-f6480a6f0780`（来源 `111906a`）。`cellapp.ai` 上部署指引和应用设计指导的中英文四份 Markdown 均以 `text/markdown` 返回，内容与源码逐字一致。网站 lint、构建和生产预览浏览器检查通过；初次构建因旧 Parcel 缓存导致预渲染与脚本不一致，禁用该缓存重建后，浏览器检查无运行时错误，上传的是复验后的产物。
- 发布前 npm 最新版为 `0.2.1`，目标 `0.3.0` 尚未占用；`NPM_ACCESS_TOKEN` 以免交互方式验证为 `cellapp-ai`，未输出或写入 token。skills `c4d7610` 的 typecheck、22 项测试、build、Skill、公开边界和打包检查通过；38 个打包文件包括中英文 Skill，本地打包安装验证了 `cellapp` 与兼容命令 `ohmyapp`。
- 从 skills `main` 提交 `c4d7610` 发布 `@cellapp/cli@0.3.0` 成功。公共 npm 注册表的 `latest` 已更新为 `0.3.0`；从注册表全新安装后，两条命令均能显示帮助，中英文 Skill 文件与仓库源码一致。没有重复发布。
- 正式环境没有可复用的 Cellapp 用户凭证，因此本次未执行带有效用户授权的线上正向读写验收；此前隔离环境的真实浏览器、PostgreSQL、PostgREST 和 MinIO 联调通过，此缺口仍按上节记录保留。
