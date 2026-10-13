# 应用数据与访问验收记录

本轮变更：`add-application-data-access`。真实服务测试分别使用隔离的 `cellapp_test`/`cellapp_appdata_test` 与 `cellapp_namespace_test`/`cellapp_namespace_appdata` 数据库、独立 PostgREST 16.3 容器和本地 MinIO 测试桶；未连接生产数据库或生产对象桶。浏览器测试使用临时 HTTPS 证书，并只在本次 Chrome 进程中按 SPKI 指纹信任它；Node 使用同一临时 CA。测试身份提供方仅在 Go 测试进程里启用。UI 预览测试使用明确的 API fixture，不能当作真实后端验收。

| 检查 | 本轮结果 | 依据 |
| --- | --- | --- |
| 控制库迁移：全新/001 升级/003 升级/重复/并发 | 通过 | `go test -race -count=1 ./apps/server/...`，隔离 PostgreSQL |
| 应用库独立、角色隔离、可重试建库、DDL 跨应用与函数/扩展阻断 | 通过 | `apps/server/appdata` 真实角色测试 |
| Manifest 校验、RLS 的 USING/WITH CHECK、版本冲突、回滚、删列计数、并发存储限额、超时 | 通过 | `apps/server/appdata` 真实 PostgreSQL 测试 |
| 自定义表命名：`users` 同名跨应用隔离、`rpc`/系统列/生成索引冲突预检 | 通过 | `TestManifestValidationAndPlan`、`TestSameUserTableNameInDifferentApplications`，隔离 PostgreSQL；客户端 manifest 单元测试 |
| 应用命名空间：旧库回填、同名应用、跨所有者空间、空间/别名冲突、删除后别名占用 | 通过 | `TestRunFromLegacySchema`、`TestRunFromAppSessionsSchema`、`TestApplicationNamespaceAliasAndTombstone`，隔离 PostgreSQL |
| 控制台授权、单次登录票据、过期/重放、按应用用户 ID、撤销与密钥轮换 | 通过 | `apps/server/internal/hosting` 集成测试 |
| 真实 PostgREST：表 CRUD、过滤/排序/分页、伪造请求头、跨应用/撤销/未登录拒绝 | 通过 | `TestPostgREST...` 及 `TestBrowserEndToEnd` |
| HTTPS 浏览器与真实对象存储：登录、申请/审批/撤销、公开/受限、旧密钥、`supabase-js@2.117.3` 笔记 Demo 静态前端建表、页面新增/编辑/删除及 `supabase-js` CRUD、跨用户 RLS、Cookie 域隔离、`@空间/别名` HTTPS 跳转、删除 | 通过 | `RUN_BROWSER_TESTS=1 ... go test -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting -v` |
| 控制台窄屏、键盘、焦点、别名与分享链接、空/失败状态、一次性密钥 | 通过（UI fixture） | `npm test --workspace @cellapp/web` |
| 独立公开 CLI、契约、Skill 检查 | 通过 | `npm --prefix skills run typecheck`, `test`, `build`, `check:skill`, `check:package` |
| 主仓库类型检查、构建、测试、Go vet | 通过 | `npm run typecheck`, `npm test`, `npm run build`, `go vet ./apps/server/...` |

真实跨服务浏览器测试需要环境变量 `TEST_DATABASE_URL`、`TEST_APP_DATA_DATABASE_URL`、`TEST_APP_DATA_PROVISIONER_URL`、`TEST_POSTGREST_URL`、`S3_ACCESS_KEY_ID`、`S3_SECRET_ACCESS_KEY`，以及本机 Chrome、私有 MinIO bucket。未配置时浏览器测试明确跳过；PostgreSQL/PostgREST 集成测试也会标明所需变量。`npm test --workspace @cellapp/web` 使用 fixture；上线前仍需在目标环境核对内部 PostgREST 端口不可公网直连、真实域名证书及三个数据源同一恢复点的备份。

恢复：保持控制库、应用数据库与对象存储同一时间范围的快照；先恢复数据库和私有桶，再启动控制服务及 PostgREST；检查应用 `dataState`、CLI `data status`/`data verify`、活动发布资源和登录授权。删除清理失败时保留控制库墓碑，由维护任务重试，不复用已删除应用 ID。参见 [应用数据运维](app-data-operations.md)。
