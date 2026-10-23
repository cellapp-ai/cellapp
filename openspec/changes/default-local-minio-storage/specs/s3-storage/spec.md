# Spec Delta

## MODIFIED Requirements


### Requirement: Local RustFS development defaults

系统 SHALL 在非生产且未显式配置 S3 时使用本地 RustFS；默认 Endpoint SHALL 保持 `http://127.0.0.1:9000`，Region 为 `us-east-1`，Bucket 为 `cellapp`，开发凭证与本地容器一致。本地启动入口 SHALL 提供持久化 RustFS 和私有 Bucket 初始化，S3/管理端口 MUST 仅绑定 loopback。初始化 MUST 幂等、保留已有对象，失败 MUST 明确报告。

#### Scenario: Fresh local setup
- **WHEN** 开发者按照当前说明启动本地依赖并使用默认 S3 配置
- **THEN** 私有 Bucket 就绪，服务健康检查及签名对象读写可用，无需外部云存储

#### Scenario: Repeat setup and restart
- **WHEN** RustFS Bucket 已包含对象并重复初始化或重启容器
- **THEN** Bucket 仍为私有，已有对象可读取，流程不删除 Bucket、对象或持久卷

#### Scenario: Failed initialization
- **WHEN** RustFS 不可达、认证失败或 Bucket 初始化失败
- **THEN** 初始化返回非零状态并指出失败阶段，不报告环境已就绪

### Requirement: Private bundled RustFS for single-host deployment

单机自部署 SHALL 默认使用持久化的私有 RustFS Bucket，并通过经验证的内部 HTTPS 连接生产服务；生成的凭证 MUST 不是开发默认值。对象 API 和管理端口 MUST 不对公网开放；显式选择的其他 HTTPS S3 兼容服务 SHALL 继续可用。

#### Scenario: Fresh self-host setup
- **WHEN** 运营者使用新的单机 Compose 配置启动完整平台
- **THEN** RustFS 的 Bucket 在应用对外就绪前创建并验证为私有，应用使用受信任的内部 TLS 和独立凭证连接

#### Scenario: Restart and unauthorized access
- **WHEN** 存储重启后访客通过应用网关读取已发布版本，或匿名客户端尝试直连对象/管理入口
- **THEN** 授权读取仍成功，匿名对象读取被拒绝，公网无法连接对象与管理端口

### Requirement: Legacy MinIO volume isolation

RustFS 默认部署 MUST 使用独立数据卷，MUST NOT 将已有 MinIO 卷作为 RustFS 卷挂载、删除或声称已迁移。对检测到的旧单机自部署配置，启动流程 MUST 拒绝无提示地切换到空 RustFS Bucket，并说明本变更不包含数据迁移。

#### Scenario: Existing self-host installation
- **WHEN** 现有自部署环境仍使用 MinIO 卷或配置，直接运行新版 RustFS 启动流程
- **THEN** 流程在对外启动前给出明确错误，保留旧卷与配置，不把现有应用指向空 Bucket

#### Scenario: Existing local development volume
- **WHEN** 本地开发环境原有 MinIO 命名卷，随后启动新版默认依赖
- **THEN** 旧卷不被复用或删除，文档明确新的 RustFS Bucket 不包含旧对象

### Requirement: Explicit production storage

生产环境 MUST 显式提供 S3 Endpoint、Region、Bucket 和完整通用凭证组，MUST 拒绝 HTTP Endpoint，MUST NOT 使用开发配置回退或自动创建 Bucket。Bucket 缺失、认证错误或存储不可达 MUST 使存储健康检查失败。

#### Scenario: Missing or insecure production configuration
- **WHEN** 生产配置缺少任一必填 S3 项或 Endpoint 使用 HTTP
- **THEN** 服务拒绝启动，不回退到本地 RustFS 或开发凭证

#### Scenario: Missing bucket
- **WHEN** 指定的生产 Bucket 不存在
- **THEN** 健康检查失败，不创建或替换生产 Bucket

### Requirement: Private storage round trip verification

系统 SHALL 使用签名 S3 请求读写、探测和删除指定 Bucket 内的对象；启用边缘分发时，访客 SHALL 仅经完成授权的应用边缘网关读取私有对象，不获得对象读取签名直链。真实存储验收 MUST 覆盖健康检查、写入后内容一致、匿名 GET 无法读取、删除后不可读和测试对象清理；迁移已有产物时 MUST 验证当前发布版本及可回滚版本的对象完整性。替身测试 MUST 不被报告成真实 RustFS 或外部 S3 验收。

#### Scenario: Live private bucket probe
- **WHEN** 对隔离本地 RustFS 显式执行真实存储验收
- **THEN** 签名读写内容一致、匿名读取被拒绝、删除后不可读取，成功及失败路径均清理随机测试对象

#### Scenario: Authorized edge object probe
- **WHEN** 生产边缘网关对私有 Bucket 执行已授权的读验收，并另以匿名请求访问 Bucket
- **THEN** 已授权的版本化产物可读取，匿名请求与未授权缓存请求不能读取对象

#### Scenario: Migration integrity and rollback
- **WHEN** 生产存储从现有 Bucket 切换到边缘可读的私有 Bucket
- **THEN** 系统在切换访客流量前核对当前和保留期内发布产物的路径、大小与摘要，并在回滚窗口内保留上一服务版本可读取的对象；缺失或摘要不一致时停止切换

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
