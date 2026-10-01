# s3-storage Specification

## Purpose

为静态托管提供供应商无关的私有 S3 对象存储配置和验证边界，使新开发环境能够使用本地 MinIO 复现健康检查及对象读写，同时保持显式外部配置与私有对象边界，避免生产环境意外使用开发凭证或公开产物。

## Requirements

### Requirement: Local MinIO development defaults

系统 SHALL 在非生产且未显式配置 S3 时使用本地 MinIO；默认 Endpoint SHALL 为 `http://127.0.0.1:9000`，Region 为 `us-east-1`，Bucket 为 `cellapp`，开发凭证与本地容器一致。本地开发启动入口 SHALL 提供持久化 MinIO 和私有 Bucket 初始化，S3/管理端口 MUST 仅绑定 loopback。初始化 MUST 幂等、保留已有对象，初始化失败 MUST 明确报告失败。

#### Scenario: Fresh local setup
- **WHEN** 开发者按照开发说明启动本地依赖并使用默认 S3 配置
- **THEN** 私有 Bucket 创建完成后服务健康检查和签名对象读写可用，无需云资源或七牛配置

#### Scenario: Repeat setup and restart
- **WHEN** Bucket 已包含对象并重复执行初始化或重启容器
- **THEN** Bucket 保持私有且已有对象仍可读取，不删除 Bucket、对象或持久卷

#### Scenario: Failed initialization
- **WHEN** MinIO 不可达、认证失败或 Bucket 初始化失败
- **THEN** 初始化命令返回非零状态，开发说明提供可定位的检查及重试方式，不报告环境已就绪

### Requirement: Generic S3 configuration without legacy credentials

系统 SHALL 接受显式 `S3_ENDPOINT`、`S3_REGION`、`S3_BUCKET`、`S3_ACCESS_KEY_ID` 和 `S3_SECRET_ACCESS_KEY`。通用凭证 MUST 成对提供；旧 `QINIU_ACCESS_KEY`/`QINIU_SECRET_KEY` MUST NOT 被读取或用作回退。只提供一项通用凭证时 MUST 拒绝配置，不使用旧值或开发默认值补齐。显式配置的 S3 后端 MUST 不被本地默认 Endpoint 覆盖。

#### Scenario: Explicit S3 credentials
- **WHEN** 提供完整通用凭证和显式 S3 Endpoint/Region/Bucket
- **THEN** 对象请求使用通用凭证及显式后端配置

#### Scenario: Legacy credentials are ignored
- **WHEN** 仅设置旧供应商凭证变量
- **THEN** 非生产环境仍使用本地开发凭证，生产环境拒绝缺失通用凭证，不尝试以旧变量访问对象

#### Scenario: Incomplete credentials
- **WHEN** 通用凭证组仅有一项
- **THEN** 服务拒绝配置并指出缺失变量，不使用旧变量或开发默认值补齐凭证，错误不暴露秘密值

### Requirement: Explicit production storage

生产环境 MUST 显式提供 S3 Endpoint、Region、Bucket 和完整通用凭证组，MUST 拒绝 HTTP Endpoint，MUST NOT 使用开发配置回退或自动创建 Bucket。Bucket 缺失、认证错误或存储不可达 MUST 使存储健康检查失败。

#### Scenario: Missing or insecure production configuration
- **WHEN** 生产配置缺少任一必填 S3 项或 Endpoint 使用 HTTP
- **THEN** 服务拒绝启动，不回退到本地 MinIO 或开发凭证

#### Scenario: Missing bucket
- **WHEN** 指定的生产 Bucket 不存在
- **THEN** 健康检查失败，不创建或替换生产 Bucket

### Requirement: Private storage round trip verification

系统 SHALL 使用签名 S3 请求读写、探测和删除指定 Bucket 内的对象，访客 SHALL 经应用网关读取产物，不获得对象读取签名直链。真实存储验收 MUST 覆盖健康检查、写入后内容一致、匿名 GET 无法读取、删除后不可读和测试对象清理；替身测试 MUST 不被报告成真实 MinIO 或外部 S3 验收。

#### Scenario: Live private bucket probe
- **WHEN** 对隔离本地 MinIO 显式执行真实存储验收
- **THEN** 签名读写内容一致、匿名读取被拒绝、删除后不可读取，成功及失败路径均清理随机测试对象
