# 本地 MinIO 存储变更验收记录

变更：`default-local-minio-storage`。日期：2026-09-30。主规格已同步，变更按用户确认带未完成验收警告归档；下列结论仅代表本轮实际执行结果。

## 工作区与配置

当前实施工作区：`/Users/wuyanxin/.codex/worktrees/local-minio/cellapp`；主仓库与子仓库分支均为 `codex/local-minio-storage`。原检出的文件保留，未执行自动审批拒绝的批量还原操作。测试期间曾用现有 `node_modules` 符号链接复用依赖，测试后已移除，未纳入交付。

开发默认值为 `http://127.0.0.1:9000`、`us-east-1`、`cellapp`；服务只读取通用 `S3_ACCESS_KEY_ID` / `S3_SECRET_ACCESS_KEY`。旧 `QINIU_*` 变量不再生效，生产未配置通用凭证会拒绝启动。已有外部环境需在升级前迁移真实密钥，原 Endpoint/Region/Bucket 继续保留；不自动迁移对象。

Compose 现默认采用用户提供的 [Supabase S3 Compose](https://github.com/supabase/supabase/blob/master/docker/docker-compose.s3.yml) 中的 Chainguard 镜像。服务 `cgr.dev/chainguard/minio:latest` 已下载，digest `sha256:4692462f35d97d7e82c30371d82f057703c5d9489bcae726010594c812f2d285`，实际版本 `RELEASE.2026-09-22T19-25-18Z`；客户端 `cgr.dev/chainguard/minio-client:latest-dev` 的 digest 为 `sha256:a552f9a01603b3db647712ae056e8bb4c04800365707f5bd4abffa05c9e4198e`，`mc --version` 为 `DEVELOPMENT.GOGET`。两个镜像都已成功下载并通过隔离 Compose 验证。`CELLAPP_MINIO_IMAGE` 和 `CELLAPP_MINIO_CLIENT_IMAGE` 可覆盖镜像。此前 Docker Hub 的 `minio/minio:latest` 超时、`dhi.io/minio:latest` 返回 401、`dhi.m.daocloud.io/minio:latest` 解包报 `invalid tar header`；这些是其他渠道的失败结果。

已从官方 `github.com/minio/minio@latest` 源码构建本机二进制，解析版本为 `v0.0.0-20260212201848-7aac2a2c5b7c`，其版本字符串是 `DEVELOPMENT.GOGET`，不是官方容器发布版本。以本机 `127.0.0.1:19000/19001` 和独立目录启动，并创建隔离 Bucket `cellapp-verify-20260930`。该实例仅用于实际 S3 验收，不代替 Compose 发布镜像验证。另将同一源码交叉编译为 Linux arm64 二进制，构建本地镜像 `cellapp-minio-verify:source-20260212`（本地 image ID `858c42767f23`），仅用于独立 Compose 项目测试；它不是可从仓库拉取的发布镜像；测试后本地镜像和工作区内临时二进制均已清理。

## 已通过

- 真实 MinIO 的 `RUN_S3_TESTS=1 go test -race -count=1 -run TestLiveS3 -v ./apps/server/internal/hosting`：Bucket 健康、签名写入/读取与内容一致、匿名 GET 拒绝、删除后不可读和探针清理均通过。端点为本机隔离端口，Bucket 为 `cellapp-verify-20260930`。
- 在本机隔离 MinIO 上写入专用对象，停止并重启进程后读取内容一致；随后清理对象和 Bucket。
- 使用本地源码测试镜像在独立 Compose 项目 `cellapp-minio-verify-0930` 验证：`9000/9001` 均只绑定 `127.0.0.1`；首次和重复 `minio-init` 均成功且私有；写入对象后重启容器内容保留；错误密钥和不可达 Endpoint 均在有界重试后以退出码 1 失败；对 Compose 9000 端口再次执行 `TestLiveS3` 通过。测试对象、容器和专用卷已清理。
- Supabase Chainguard 发布镜像在隔离项目 `cellapp-chainguard-verify-0930` 验证：API/控制台仅绑定 `127.0.0.1`；配套客户端首次与重复初始化成功且 Bucket 私有；真实 `TestLiveS3` 通过；容器重启后对象内容一致；错误密钥和不可达 Endpoint 均以退出码 1 失败。测试对象、容器和卷已清理。初次 Go 测试卡在本机内置 vet 版本查询，关闭内置 vet 后 `go test -vet=off -race -count=1 -run TestLiveS3 -v ./apps/server/internal/hosting` 通过；此前独立 `go vet ./apps/server/...` 已通过。
- 持续开发实例 `cellapp-local-minio-minio-1` 已从当前默认镜像启动，私有 `cellapp` Bucket 初始化成功；`/minio/health/live` 返回 200，真实 `TestLiveS3` 再次通过。它使用独立项目与命名卷，保留运行供本地开发，未触碰已有的 Supabase `cellapp` 项目。
- 前轮代码的 `npm run typecheck`、`npm test`、`npm run build`、`go vet ./apps/server/...` 均通过。Web 浏览器检查 6 组通过。独立 PostgreSQL 临时容器上的 `TEST_DATABASE_URL=... go test -race -count=1 ./apps/server/...` 通过。
- 配置和存储单元测试覆盖本地默认值、显式参数、通用凭证成对读取、缺项失败、旧变量被忽略、生产缺项/HTTP 拒绝、Bucket 缺失和认证失败。
- 客户端类型检查、11 项单元测试、构建、Skill、公开边界、包检查及打包预览通过；19 项批准文件，不包含存储运维资料。
- Compose 配置解析、初始化脚本语法、文档链接、图表 JSON 引用、图表 HTML 浏览器呈现和 OpenSpec 严格校验通过。

## 阻碍与未执行

- Docker Hub 与 DHI 渠道在本机不可用；当前默认的 Supabase Chainguard 发布镜像已完成容器与存储验收。
- SSH 远程浏览器访问尚未验收：开发机上 `127.0.0.1:9001/` 的 HTML、JS、CSS 均返回 200，MinIO 健康端点返回 200；用户在本地浏览器连接 `localhost:9001`、`127.0.0.1:9001` 以及尝试 SSH 转发后的 `127.0.0.1:19001` 仍报告连接失败。本机服务正常，但客户端转发链路未定位完成。
- 显式浏览器验收已连接隔离 PostgreSQL 和真实 MinIO 配置，但在 macOS 临时 CA 信任设置处失败：`SecTrustSettingsSetTrustSettings: The authorization was denied since no user interaction was possible.` 未关闭 TLS 校验；发布/CLI 全链路因此尚未通过。
- 浏览器与完整 CLI 验收仍未满足；主规格已同步，变更归档于 `openspec/changes/archive/2026-09-30-default-local-minio-storage/`，原任务状态如实保留。

## 七牛遗留引用分类

盘点项目文本中的 `qiniu|qiniucs|七牛`，排除 Git 内部、依赖、生成产物、测试结果、本地秘密目录与个人 `.env`；个人配置未读取或修改。当前默认值、服务凭证读取、README、AGENTS、配置示例和图表均已清理。

| 文件 | 保留原因 |
| --- | --- |
| `apps/server/internal/hosting/config_storage_test.go` | 仅作旧变量不会生效的回归输入；服务代码不读取旧变量 |
| `docs/s3-storage.md`、`docs/qiniu-storage.md` | 旧配置迁移说明与历史链接入口 |
| `docs/verification.md` 及其他 `docs/*verification.md` 历史记录 | 保留旧轮次的真实验收或未执行事实，不冒充 MinIO 验收 |
| `openspec/changes/archive/2026-09-22-static-app-hosting/` | 保留原设计与任务事实 |
| 当前 OpenSpec 变更及本记录 | 说明停用旧变量、迁移和清理范围 |

客户端 `public-boundary.json` 将旧 `docs/qiniu` 排除项泛化为 `docs/`，子仓库当前文本不再含供应商配置；既有公开允许文件不变。
