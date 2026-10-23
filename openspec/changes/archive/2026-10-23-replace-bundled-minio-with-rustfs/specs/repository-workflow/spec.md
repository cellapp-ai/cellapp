# Spec Delta

## MODIFIED Requirements

### Requirement: Reproducible local storage guidance

项目 SHALL 提供与实际配置一致的本地 RustFS 启动、私有 Bucket 初始化、服务健康检查、真实存储验收及失败恢复说明。当前开发和自部署说明、架构图 SHALL 使用 RustFS 默认值或通用私有 S3 描述，不将七牛或 MinIO 表述为新环境默认值。七牛及 MinIO 的历史验收、旧配置迁移说明和已归档规划 SHALL 保留当时事实；公开客户端边界 MUST 继续排除存储运维材料。

#### Scenario: Current development guide
- **WHEN** 开发者从当前 README、配置示例和存储说明启动新环境
- **THEN** 无需外部云存储账号即可启动本地 RustFS、初始化私有 Bucket 并执行真实读写及匿名拒绝验收，重试与持久数据保留方法明确

#### Scenario: Existing storage configuration
- **WHEN** 已有环境使用显式配置的外部 S3，或旧单机安装仍使用 MinIO
- **THEN** 外部 S3 配置不被默认值覆盖；旧安装说明明确 MinIO 卷不会自动迁移或被 RustFS 复用，保留旧版本的运行路径

#### Scenario: Residual reference audit
- **WHEN** 审查当前代码、配置、文档、图表、子仓库及历史规划中的七牛和 MinIO 引用
- **THEN** 当前默认引用更新为 RustFS，每处保留的旧名称都有历史、迁移或依赖用途，不把历史验收当成本轮 RustFS 验证

#### Scenario: Storage documentation public boundary
- **WHEN** 检查公开客户端文件清单和发布包
- **THEN** 存储运维文档仍被排除，客户端独立检查与打包不依赖私有服务端资料
