# Proposal

## Why

当前服务开发默认指向七牛云且需要事先配置云端 Bucket/凭证，本地 Compose 无对象存储，无法直接复现完整托管环境。生产示例已经采用通用 S3 凭证变量，代码仍读取七牛专用变量，造成配置与实现不一致。

## What Changes

- 将非生产环境 S3 默认配置切换为本地 MinIO，Compose 提供持久化存储和幂等私有 Bucket 初始化。
- 使用 `S3_ACCESS_KEY_ID`、`S3_SECRET_ACCESS_KEY` 和供应商无关的内部字段；旧 `QINIU_ACCESS_KEY`/`QINIU_SECRET_KEY` 仅保留成对兼容回退，通用配置优先。
- 生产继续要求显式 S3 配置、私有 Bucket 与 HTTPS，不继承本地默认凭证，不自动创建生产 Bucket。
- 更新当前 README、配置示例、协作说明、存储文档和图表，去掉七牛默认供应商定位，记录所有七牛遗留命中及保留理由。
- 保留历史验收、归档事实以及旧配置兼容说明；已有云存储数据不自动迁移，显式配置的 S3 后端继续可用。

## Capabilities

### New Capabilities

- `s3-storage`: 本地 MinIO 开发默认、通用 S3 凭证及兼容优先级、生产配置边界和私有对象存储验收。

### Modified Capabilities

- `repository-workflow`: 提供可复现的本地对象存储启动与恢复说明，当前材料使用通用 S3 描述，七牛残留仅用于明确的历史或兼容用途。

## Impact

- 主仓库：`apps/server/internal/hosting/config.go`、`storage.go` 及配置/存储测试、`infra/compose.yaml`、`.env.example`、`.env.production.example`、README、AGENTS、运维与存储说明、图表源文件及 HTML。
- 子仓库：`skills/public-boundary.json` 中 `docs/qiniu` 排除项改为覆盖通用存储运维文档的规则，保持公开边界，不改变 CLI 协议或包 API。
- 复用既有 `minio-go/v7` 和 Storage 接口；新增本地 MinIO 与初始化工具容器，不引入新的 Go SDK、数据库迁移或服务端程序托管。
- 默认后端切换只针对未显式配置的新开发环境；旧环境需保留原 Endpoint/Region/Bucket/凭证，或单独安排数据迁移。用户自己的 `.env` 和云资源不在自动修改范围。
