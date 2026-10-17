# 控制 API 路径改造验收记录

本记录对应 OpenSpec `organize-control-api-paths`。本轮使用任务 worktree `codex/control-api-paths`，在隔离 PostgreSQL 17.6 容器 `cellapp-control-api-test-20261017` 上运行服务端集成检查；跨服务验收另用独立 Compose 项目 `cellapp-control-api-acceptance`，包含 PostgreSQL、PostgREST 16.3 与私有 MinIO 测试桶。所有修改都尚未发布。

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
| 受控根域候选 Host | 未执行 | 尚无已部署的本次 Go/Worker 候选版本、实际 Host 和边缘版本，不能将本地 Worker 替身视为候选验收 |

Go 隔离测试覆盖设备待批准、限频、拒绝、过期、重放和无效凭证；CLI 协议替身覆盖发布响应丢失后的同标识重试。真实 HTTPS 链路的强制网络超时未执行。首次拉取固定版 MinIO 镜像时 Docker Hub 超时，随后使用本机缓存的 Chainguard MinIO 与 MinIO Client 镜像，在新建 Compose 项目完成验收，没有连接已有生产资源。设备授权仍由浏览器明确批准；打开验证页或轮询不会自动批准。服务与 Worker 兼容版本须先部署并在候选 Host 验收，再发布新版 CLI/Skill。候选任务未完成前，OpenSpec 不归档，代码不声称已发布。
