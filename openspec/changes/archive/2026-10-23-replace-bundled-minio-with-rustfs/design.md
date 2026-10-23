# Design

## Context

见 [proposal.md](proposal.md)。开发 Compose 的 `minio`/`minio-init` 使用 loopback 9000/9001 与命名卷；单机自部署 Compose 使用私有网络、内部 TLS、生成凭证和启动阶段检查。`scripts/selfhost.py` 将 MinIO 凭证、证书路径和 `https://minio:9000` 写入受保护配置。Go 服务通过 `S3_*` 配置及 `minio-go` S3 客户端使用 path-style、SigV4、指定 Bucket 的探测和对象读写；这属于通用 S3 客户端依赖，不意味着服务端必须运行 MinIO。

当前主规格 `s3-storage`、`repository-workflow` 和两个未归档变更仍指向 MinIO。RustFS 官方文档列出 S3 Bucket/对象操作的兼容覆盖，但明确不保证所有 MinIO 行为一致；官方镜像以非 root UID 10001 运行，TLS 需指定目录中的 `rustfs_cert.pem` 与 `rustfs_key.pem`。[S3 兼容矩阵](https://docs.rustfs.com/en/reference/s3-compatibility)、[Docker 安装](https://docs.rustfs.com/en/installation/container/docker)、[TLS 配置](https://docs.rustfs.com/en/integration/tls-configured)。这些是设计依据，不是本项目已通过的真实 RustFS 验收。

## Goals / Non-Goals

**Goals:** 新开发环境与新单机自部署环境默认使用可复现的 RustFS；保持私有 Bucket、生产内部 HTTPS、明确失败、独立持久卷、可恢复备份和现有 `S3_*` 协议边界；用真实 RustFS 验证 Cellapp 实际调用。

**Non-Goals:** 本变更不搬迁、转换或读取旧 MinIO 数据卷，不把现有单机部署自动升级到 RustFS，不替换外部 S3 服务、Go 的通用 S3 客户端或发布协议，也不引入 RustFS 集群/高可用。旧部署继续运行原版本；如需迁移对象，另立变更。

## Decisions

### 1. 两套 Compose 都换默认服务，存储卷独立

`infra/compose.yaml` 和 `infra/compose.selfhost.yaml` 均使用固定 RustFS 服务镜像与独立 `rustfs` 命名卷；初始化服务使用经验证的固定客户端镜像。实施时选择可从实际部署网络获取、同时覆盖目标 `linux/amd64` 与开发用 `linux/arm64` 的明确标签或 digest，并记录验证值；不提交 `latest` 或用本机临时重标记镜像当作固定版本验收。开发端保留 Go 默认 `http://127.0.0.1:9000`、Region/Bucket 和本地凭证，以减少无关的服务端改动；仅 9000/9001 映射到 loopback。自部署只映射 HTTPS 入口，RustFS 两个端口留在私网。

另一选择是只替换自部署 Compose，但会使本地测试继续验证 MinIO 而非新默认。直接复用 `minio` 卷也不可作为切换方式：RustFS 的 [MinIO 格式互操作说明](https://github.com/rustfs/rustfs/blob/main/docs/architecture/minio-file-format-compat.md)列出构建变体和对象类型限制，默认镜像不能被视为旧卷的通用读取器。

### 2. 新自部署配置生成 RustFS 凭证和独立 TLS 材料

新配置使用 `RUSTFS_ACCESS_KEY`/`RUSTFS_SECRET_KEY`，分别生成符合 SigV4 形式的随机值，继续映射到应用的 `S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY`。保持 `S3_ENDPOINT=https://rustfs:9000` 与生产 HTTP 禁止规则。为 `rustfs` DNS 名生成服务器证书和 CA；只将服务器证书/私钥放入 RustFS 可读目录，CA 私钥不进入容器。RustFS 默认非 root，因此证书文件和数据卷要按 UID 10001 权限设计，不能沿用仅 root 可进入的 MinIO 证书目录。应用与初始化客户端只信任该 CA，不禁用证书校验。[RustFS 凭证说明](https://docs.rustfs.com/en/operations/credentials)、[TLS 配置](https://docs.rustfs.com/en/integration/tls-configured)。

另一选择是保留 `MINIO_ROOT_*` 兼容别名或把生产存储改回内部 HTTP。前者延续错误命名，后者违反现有服务生产 HTTPS 约束，因此新配置均不采用。

### 3. 初始化以 S3 结果和匿名拒绝为完成条件

Bucket 初始化在 RustFS readiness 成功后运行，有界重试、幂等建桶并验证私有策略和匿名 GET 拒绝；任何阶段失败都阻止应用/入口就绪。候选工具是官方 `rc` 镜像或能信任内部 CA 的通用 S3 客户端；在实施时以固定镜像及实际命令验证决定，不复用 MinIO 专属 `mc ready` 成功状态。启动脚本和健康检查保持明确阶段信息，重启不得删除对象。RustFS 提供 `/health/ready` 与 S3 Bucket 操作，仍需对选定版本实测。[状态探针](https://docs.rustfs.com/en/operations/status-check)、[S3 兼容矩阵](https://docs.rustfs.com/en/reference/s3-compatibility)。

另一选择是只等容器启动并假定默认 Bucket 私有；这无法证明 Bucket 建成、凭证有效或匿名请求被拒绝。

### 4. 旧自部署配置启动时拒绝隐式切换

新 `settings.json` 记录存储后端为 RustFS。现有配置缺少该标记且含 MinIO 凭证/证书时，`setup`/`start` 在改写 `runtime.env` 或启动入口前失败并提示继续使用旧版本；旧 `secrets.json`、卷和证书均不改动。新的 Compose 卷名与旧卷分离，开发文档明确旧 MinIO 卷仍在但不会自动出现在 RustFS Bucket 中。此变更不实现对象复制或升级助手。

另一选择是对旧配置直接生成新空 Bucket；数据库仍指向已有发布版本，会造成成功启动却缺少产物。

### 5. 保持通用 S3 适配器，以实际操作验收兼容

先使用现有 `minio-go` 客户端，对 RustFS 运行 Cellapp 实际使用的 BucketExists、带 MD5 的 PutObject、Get/Stat、Delete、按前缀 ListObjects、健康检查和完整发布/读取流程；只在真实测试揭示不兼容时作局部调整，并回归外部 S3 配置。对象 API、CLI 契约和客户端包不因服务名称更换而变化。自部署备份改为 RustFS 卷、PostgreSQL 卷和受保护配置的一致备份与隔离恢复。

另一选择是同时更换 Go S3 SDK，会扩大回归面且不能替代真实服务测试。

### 6. 协调仍在进行的 OpenSpec 变更

实施时同步修订 `default-local-minio-storage` 和 `add-compose-self-hosting-and-reader-readme` 尚未归档的 MinIO 要求与任务状态，再同步主规格，避免归档顺序将旧默认重新写回。历史验收报告、已归档规划保留当时事实。当前 README、AGENTS、存储/自部署/运维说明和架构图统一使用 RustFS 默认值。

## Risks / Trade-offs

- [固定镜像或客户端镜像拉取失败] → 在提交默认镜像前从真实目标环境拉取并运行固定版本；失败记录为未完成，不用其他版本重标记代替。
- [S3 行为不完全一致] → 以 Cellapp 实际调用及匿名拒绝作真实验收；失败时停止切换并定位特定调用，不以文档兼容矩阵当作通过。
- [非 root 无法读取 TLS 私钥或写卷] → 验证证书路径、权限、UID 10001 和重启；CA 私钥不挂载到服务。
- [旧安装误指向空 Bucket] → 旧配置检测在生成/启动前失败；旧卷与配置不自动修改或删除。
- [单机单卷无存储冗余] → 文档继续要求离机备份并进行隔离恢复演练，不宣称高可用。

## Migration Plan

1. 新安装使用 RustFS 固定镜像、独立卷和新生成配置，先完成真实存储与端到端验收，再把它标为当前默认。
2. 已有 MinIO 自部署安装继续使用原提交与原配置；新版启动前检测并拒绝直接切换。旧 MinIO 卷保留，不在本变更内复制对象。已有外部 S3 配置继续使用原 Endpoint 和凭证。
3. 新 RustFS 安装的后续更新先备份数据库、RustFS 卷和配置；若新镜像失败，恢复同一时间点的备份并使用先前固定镜像。回滚不涉及把 RustFS 卷交给 MinIO。
