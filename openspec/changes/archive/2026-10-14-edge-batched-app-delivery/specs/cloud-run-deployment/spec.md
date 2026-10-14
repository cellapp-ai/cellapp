# Spec Delta

## MODIFIED Requirements

### Requirement: Production boundary and recovery guidance
部署说明 SHALL 明确前置的 PostgreSQL、私有 S3、Secret Manager、GitHub OAuth、隔离控制域与通配访客域、HTTPS 公网入口/DNS/证书、定时清理触发器，以及 Cloud Run 服务对这些资源的访问权限。启用边缘静态分发时还 SHALL 明确私有 R2、Worker 绑定及独立签名密钥的配置与验收。生产入口 MUST 保留 Host 精确路由与 TLS 信任边界；不得将开发用 Caddy 内部证书、`AUTH_MODE=dev`、本地 MinIO 或公开后端端口作为生产默认。说明 MUST 给出迁移、发布后权限验收、失败重试和服务及边缘版本回滚方法，并说明数据库及对象不会随服务回滚。

#### Scenario: Review production prerequisites
- **WHEN** 操作者准备首次部署
- **THEN** 能识别哪些外部资源必须先存在、如何配置控制域与访客通配域、边缘私有存储和签名密钥，以及缺少其中任一项时不能将服务标为公网可用

#### Scenario: Roll back a bad revision
- **WHEN** 新版本发布后验收失败
- **THEN** 操作者可把流量恢复到上一可用的服务与边缘组合，保留数据库、私有对象和秘密，并知晓迁移兼容性限制

#### Scenario: Verify authorization after release
- **WHEN** 操作者按上线验收检查控制域、访客域和私有对象
- **THEN** 已授权的静态资源及动态数据路径可用，未知 Host 与未授权访客/对象读取被拒绝，结果区分真实环境验证与本地镜像测试

### Requirement: Authenticated third-party edge origin
Cellapp SHALL support an optional third-party HTTPS edge in front of Cloud Run. It MUST recover the browser hostname only from an authenticated edge assertion and use that hostname consistently for control and visitor routing, application login handoff, and visitor write origin checks. In edge mode, the edge MAY serve verified private static files; control, auth and app data MUST reach the authenticated Cloud Run origin. It MUST reject unauthenticated origin requests.

#### Scenario: Valid control and visitor host
- **WHEN** an authenticated edge request carries the configured control hostname or a single application hostname under the configured visitor domain
- **THEN** the matching control or application route receives its original path, method, body, and host-scoped cookies, or a verified static request is served by the edge

#### Scenario: Forged or malformed original host
- **WHEN** a direct origin request omits the edge proof, presents an invalid proof, or names a host outside the configured control and visitor domains
- **THEN** Cellapp does not serve control or application content for that asserted host

#### Scenario: Direct Host compatibility
- **WHEN** a deployment without the third-party edge receives the browser Host directly through a trusted HTTPS ingress
- **THEN** the existing routing and authorization behavior remains available

#### Scenario: Published visitor application through the edge
- **WHEN** a visitor opens a published application through an authenticated edge request, signs in, and performs a same origin write
- **THEN** the authorization handoff returns to the requested application hostname, static assets load under a verified grant, and the same origin write passes its origin check at Cloud Run

#### Scenario: Forged edge host or cross origin write
- **WHEN** the edge proof is invalid or a visitor write Origin differs from the authenticated application hostname
- **THEN** the server rejects the forged route or write without issuing access to another host

## ADDED Requirements

### Requirement: Coordinated edge delivery release
生产发布 SHALL 在转移静态分发前验证私有 Bucket 中的现有产物、签名密钥、Worker 与 Cloud Run 的兼容性，并分别记录服务和边缘版本。任何阶段失败 MUST 保持上一可用路径或拒绝受保护内容，不得报告完整发布成功；回滚 MUST 保留已发布对象且不要求回滚数据库迁移。

#### Scenario: Candidate validation
- **WHEN** 新边缘版本和服务修订版本已准备但尚未承接访客静态流量
- **THEN** 操作者能验证登录与续签、已授权资源、应用数据请求、未知 Host、未授权对象读取及额度边界

#### Scenario: Storage or key mismatch
- **WHEN** Bucket 缺少已发布文件、签名密钥版本不匹配或 Worker 无法验证服务签发的授权
- **THEN** 切换停止，访客继续使用上一可用路径或收到明确失败，私有文件不公开
