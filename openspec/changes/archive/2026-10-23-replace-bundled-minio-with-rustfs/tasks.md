# Tasks

## 1. 固定镜像与协议基线

- [x] 1.1 选定 RustFS 服务与初始化客户端的固定标签或 digest，在目标 `linux/amd64` 和开发 `linux/arm64` 环境核对 manifest、实际拉取与启动；记录来源和版本，拉取失败不得以 `latest` 或重标记缓存替代通过。
- [x] 1.2 在隔离 Bucket 上用现有 Go S3 适配器验证 BucketExists、带 MD5 的上传、Get/Stat、Delete、前缀 ListObjects、健康检查及匿名 GET 拒绝；运行 `RUN_S3_TESTS=1 go test -race -count=1 -run TestLiveS3 -v ./apps/server/internal/hosting` 并记录实际结果，只有确证不兼容时才修改调用并回归显式外部 S3 配置。

## 2. 本地开发默认存储

- [x] 2.1 在 `infra/compose.yaml` 用固定 RustFS 镜像、独立 `rustfs` 卷和 loopback API/控制台端口替换 MinIO，保持 `.env.example` 与 Go 默认 `S3_*` 值一致；验证 Compose 配置和宿主监听地址，确认旧 MinIO 卷未被复用。
- [x] 2.2 将 Bucket 初始化改为对 RustFS 的有界等待、幂等建桶与私有访问验收；在隔离 Compose 项目验证首次运行、重复运行、重启后读取以及不可达/错误凭证的非零退出。
- [x] 2.3 更新 `docs/s3-storage.md`、`docs/development.md` 和本地故障恢复命令，明确旧 MinIO 卷不会自动迁移；逐条执行或静态核对文档命令及相对链接，并记录真实 RustFS 验收结果。

## 3. 单机自部署默认存储

- [x] 3.1 在 `infra/compose.selfhost.yaml` 与 `scripts/selfhost.py` 中生成并使用 RustFS 专用凭证、独立卷、`https://rustfs:9000` 和内部 CA/服务器证书；验证非 root UID 10001 可读取证书及写入卷、应用和初始化服务验证 TLS、对象与管理端口不映射到公网。
- [x] 3.2 为旧 `settings.json`/MinIO 配置加入启动前拒绝路径，确保拒绝时不改写旧 secrets、证书、卷或 `runtime.env`；用隔离旧配置 fixture 验证明确错误，并验证新配置重复 setup 保持秘密字节不变。
- [x] 3.3 更新启动阶段与存储初始化门禁；在隔离单机 Compose 环境验证建桶、匿名拒绝、依赖失败与修复重试、重复启动及重启后的对象读取，入口只在应用健康后启动。
- [x] 3.4 更新 `docs/self-hosting.md` 的新装、备份、隔离恢复与失败处理，备份 PostgreSQL、RustFS 卷和受保护配置；照文档进行停写备份/恢复演练，并在新环境核对控制元数据、应用表和发布对象。

## 4. 当前资料、冲突规格与集成验收

- [x] 4.1 将 README 中英文、根 `AGENTS.md`、当前运维入口与架构 JSON/HTML 的默认存储说明改为 RustFS，保留历史验收和归档原文；核对所有当前 MinIO 命中用途、相对链接、JSON 解析和图表可见内容。
- [x] 4.2 协调 `default-local-minio-storage` 与 `add-compose-self-hosting-and-reader-readme` 的未归档 MinIO 要求和任务状态，使其不会在后续同步/归档覆盖 RustFS 默认值；用 `openspec validate --strict` 检查相关变更并记录协调结果。
- [x] 4.3 使用真实 RustFS 和隔离数据库/测试身份提供方运行浏览器端到端授权、静态发布、应用访问和数据表流程；记录 `go test -race`、浏览器测试、Go vet、相关构建的通过/失败/跳过及未执行项，替身结果不得冒充真实存储验收。

## Workflow follow-up

- 在必要验收满足后同步主规格；同步 `s3-storage` 时更新其仍写 MinIO 的 Purpose，检查两个旧未归档变更与新默认不冲突，再归档本变更。
- 归档成功后按仓库规则仅提交本变更涉及文件；合入、推送和发布分别按 Gitflow 与部署规则执行。
