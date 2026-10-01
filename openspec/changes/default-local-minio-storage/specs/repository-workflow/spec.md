# Spec Delta

## ADDED Requirements

### Requirement: Reproducible local storage guidance

项目 SHALL 提供与实际配置一致的本地 MinIO 启动、私有 Bucket 初始化、服务健康检查、真实存储验收及失败恢复说明。当前开发示例、架构说明和图表 SHALL 使用本地 MinIO 或通用私有 S3 描述，MUST NOT 将七牛设为默认或必选供应商。七牛遗留引用 SHALL 有完整清单及用途说明，仅保留历史事实、旧配置兼容或明确的历史链接入口；历史验收及已归档规划 MUST 保留当时事实。公开客户端边界 MUST 继续排除存储运维材料。

#### Scenario: Current development guide
- **WHEN** 开发者从当前 README、配置示例和存储说明启动新环境
- **THEN** 无需七牛账号即可启动本地 MinIO、初始化私有 Bucket 并执行真实读写及匿名拒绝验收，重试与持久数据保留方法明确

#### Scenario: Existing storage configuration
- **WHEN** 已有环境升级并按迁移说明保留原 S3 Endpoint/Region/Bucket 和凭证
- **THEN** 原对象可继续访问，不自动改写个人配置、搬迁云对象或删除原 Bucket，回滚方式明确

#### Scenario: Residual reference audit
- **WHEN** 审查当前代码、配置、文档、图表、子仓库及历史规划中的七牛引用
- **THEN** 每处命中都有更新或保留理由，当前默认定位被清理，历史记录和兼容入口明确，不把历史验收当成本轮验证

#### Scenario: Storage documentation public boundary
- **WHEN** 检查公开客户端文件清单和发布包
- **THEN** 通用存储运维文档仍被排除，客户端独立检查与打包不依赖私有服务端资料
