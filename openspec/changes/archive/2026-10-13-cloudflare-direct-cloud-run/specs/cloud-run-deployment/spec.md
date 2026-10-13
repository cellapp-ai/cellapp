# Spec Delta

## ADDED Requirements

### Requirement: Authenticated third-party edge origin

Cellapp SHALL support an optional third-party HTTPS edge in front of Cloud Run. It MUST recover the browser hostname only from an authenticated edge assertion, route it with the same control and visitor isolation rules as a directly forwarded Host, and reject unauthenticated origin requests.

#### Scenario: Valid control and visitor host
- **WHEN** an authenticated edge request carries the configured control hostname or a single application hostname under the configured visitor domain
- **THEN** the request reaches the matching control or application route with its original path, method, body, and host-scoped cookies intact

#### Scenario: Forged or malformed original host
- **WHEN** a direct origin request omits the edge proof, presents an invalid proof, or names a host outside the configured control and visitor domains
- **THEN** Cellapp does not serve control or application content for that asserted host

#### Scenario: Direct Host compatibility
- **WHEN** a deployment without the third-party edge receives the browser Host directly through a trusted HTTPS ingress
- **THEN** the existing routing and authorization behavior remains available

## MODIFIED Requirements

### Requirement: Production boundary and recovery guidance

部署说明 SHALL 明确前置的 PostgreSQL、私有 S3、Secret Manager、GitHub OAuth、隔离控制域与通配访客域、HTTPS 公网入口/DNS/证书、定时清理触发器，以及 Cloud Run 服务对这些资源的访问权限。生产入口 MUST 保留 Host 精确路由与 TLS 信任边界；不得将开发用 Caddy 内部证书、`AUTH_MODE=dev`、本地 MinIO 或公开后端端口作为生产默认。说明 MUST 给出迁移、发布后权限验收、失败重试和镜像修订版本回滚方法，并说明数据库及对象不会随服务回滚。

#### Scenario: Review production prerequisites
- **WHEN** 操作者准备首次部署
- **THEN** 能识别哪些外部资源必须先存在、如何配置控制域与访客通配域，以及缺少其中任一项时不能将服务标为公网可用

#### Scenario: Roll back a bad revision
- **WHEN** 新版本发布后验收失败
- **THEN** 操作者可把流量恢复到上一可用修订版本，保留数据库、私有对象和秘密，并知晓迁移兼容性限制

#### Scenario: Verify authorization after release
- **WHEN** 操作者按上线验收检查控制域、访客域和私有对象
- **THEN** 已授权路径可用，未知 Host 与未授权访客/对象读取被拒绝，结果区分真实环境验证与本地镜像测试
