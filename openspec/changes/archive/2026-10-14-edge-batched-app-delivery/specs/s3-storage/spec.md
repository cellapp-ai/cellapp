# Spec Delta

## MODIFIED Requirements

### Requirement: Private storage round trip verification
系统 SHALL 使用签名 S3 请求读写、探测和删除指定 Bucket 内的对象；启用边缘分发时，访客 SHALL 仅经完成授权的应用边缘网关读取私有对象，不获得对象读取签名直链。真实存储验收 MUST 覆盖健康检查、写入后内容一致、匿名 GET 无法读取、删除后不可读和测试对象清理；迁移已有产物时 MUST 验证当前发布版本及可回滚版本的对象完整性。替身测试 MUST 不被报告成真实 MinIO 或外部 S3 验收。

#### Scenario: Live private bucket probe
- **WHEN** 对隔离本地 MinIO 显式执行真实存储验收
- **THEN** 签名读写内容一致、匿名读取被拒绝、删除后不可读取，成功及失败路径均清理随机测试对象

#### Scenario: Authorized edge object probe
- **WHEN** 生产边缘网关对私有 Bucket 执行已授权的读验收，并另以匿名请求访问 Bucket
- **THEN** 已授权的版本化产物可读取，匿名请求与未授权缓存请求不能读取对象

#### Scenario: Migration integrity and rollback
- **WHEN** 生产存储从现有 Bucket 切换到边缘可读的私有 Bucket
- **THEN** 系统在切换访客流量前核对当前和保留期内发布产物的路径、大小与摘要，并在回滚窗口内保留上一服务版本可读取的对象；缺失或摘要不一致时停止切换
