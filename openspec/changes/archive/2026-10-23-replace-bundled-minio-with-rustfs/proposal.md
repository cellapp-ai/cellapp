# Proposal

## Why

Cellapp 的本地开发和单机自部署目前都捆绑 MinIO，且当前自部署变更因固定版本镜像无法完成真实验收。希望将项目自带的对象存储改为 RustFS，同时保持私有 S3 边界。更换存储实现本身不能证明镜像可获取或协议兼容，因此这些必须成为实施验收条件。

## What Changes

- 新建本地开发与单机自部署环境默认启动固定版本的 RustFS，创建私有 Bucket，保持本地端口仅绑定 loopback、自部署存储端口不对公网开放；自部署服务到存储仍使用经验证的内部 HTTPS。
- **BREAKING**：新环境的 Compose 服务、命名卷及生成的存储专用配置从 MinIO 改为 RustFS。已有 MinIO 数据卷不能作为 RustFS 卷直接挂载；本变更不迁移 MinIO 对象，也不把旧数据卷当作新存储。已有部署需要保留旧版本或自行另行安排迁移。
- 保留 `S3_*` 配置、通用 S3 存储适配器和外部 HTTPS S3 服务支持。只有经过真实 RustFS 验收确认不兼容的调用才调整实现，不因产品改名替换 Go 的 S3 客户端库。
- 更新初始化、启动、备份恢复和故障排查说明，以及 README、AGENTS、当前架构图和本轮验收记录；历史验收与归档保留原有 MinIO 事实。
- 协调尚未归档的 `default-local-minio-storage` 与 `add-compose-self-hosting-and-reader-readme` 变更，避免其 MinIO 要求在规格同步时重新覆盖 RustFS 默认值。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `s3-storage`：本地默认存储与真实 S3 验收改为适用于 RustFS，并明确旧 MinIO 卷不会被复用或自动迁移。
- `repository-workflow`：当前开发与自部署资料应使用 RustFS 默认值，保留准确的历史 MinIO 记录。

## Impact

涉及 `infra/compose.yaml`、`infra/compose.selfhost.yaml`、Bucket 初始化脚本、`scripts/selfhost.py`、开发/自部署配置与备份、真实 S3 和浏览器验收，以及当前 README、AGENTS、存储/运维文档与图表。服务端公开 API、CLI 契约、应用数据表和 `skills` 子仓库无需预设修改；显式外部 S3 后端不得受默认值切换影响。
