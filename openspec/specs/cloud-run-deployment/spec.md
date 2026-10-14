# Cloud Run Deployment Specification

## Purpose

为已有生产依赖和 HTTPS 域名入口的环境提供可复现的 Cellapp Cloud Run 发布流程，使控制面与访客应用域共享一个受保护的服务修订版本，并在配置或发布失败时给出可恢复的结果。

## Requirements

### Requirement: Paired production image

项目 SHALL 从固定源码构建包含 Go server、配套 Web console 静态产物及生产 Caddy 入口的镜像。镜像 MUST 能在 Cloud Run 指定的端口接收 HTTP 请求，代理到仅供容器内部访问的 Go 服务，同时保留原始 Host、路径及必要请求信息。镜像 MUST NOT 包含部署凭证、应用密钥、本地 `.env` 或开发数据。

#### Scenario: Start a matching revision
- **WHEN** 以完整生产配置和 Cloud Run 端口启动镜像
- **THEN** 控制域能加载配套 console、浏览器 API 与 OAuth 路径，访客域请求进入同一版本的应用网关，Go 后端不能从容器外直接访问

#### Scenario: Missing build input or runtime config
- **WHEN** 前端构建失败、静态产物缺失、端口无效或生产必填配置缺失
- **THEN** 构建或启动明确失败，不能把不完整镜像当作可用版本

### Requirement: One-command Cloud Run release

项目 SHALL 提供一个文档化命令，在预先配置的 Google Cloud 项目和外部依赖上完成镜像构建、发布前数据库迁移及 Cloud Run 服务修订版本发布。命令 MUST 校验项目、区域、服务、生产域名、外部数据库/私有对象存储及秘密引用等必填输入，MUST 将敏感值从受管理的秘密引用注入运行环境，MUST 在迁移失败时停止服务发布，且任何失败都不得报告完整成功。重复执行同一版本 MUST 不删除持久数据。

#### Scenario: Existing environment deploys successfully
- **WHEN** 操作者为已有项目、私有后端和两个隔离域提供有效配置并运行部署命令
- **THEN** 可得到指向本次镜像的 Cloud Run 修订版本，迁移在承接新流量前完成，命令报告镜像标识、修订版本及可访问入口

#### Scenario: Preflight or migration fails
- **WHEN** 必填配置、秘密引用、镜像构建、推送或数据库迁移失败
- **THEN** 命令非零退出，指出失败阶段且不打印秘密；迁移失败时不将新镜像发布到服务

#### Scenario: Runtime or route verification fails
- **WHEN** 新修订版本未就绪，或预配置的控制域/通配访客域未正确路由到该服务
- **THEN** 命令不宣称完整部署成功，保留可定位的修订版本和恢复指引，不删除数据库或对象

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

### Requirement: Authenticated third-party edge origin

Cellapp SHALL support an optional third-party HTTPS edge in front of Cloud Run. It MUST recover the browser hostname only from an authenticated edge assertion and use that hostname consistently for control and visitor routing, application login handoff, and visitor write origin checks. It MUST reject unauthenticated origin requests.

#### Scenario: Valid control and visitor host
- **WHEN** an authenticated edge request carries the configured control hostname or a single application hostname under the configured visitor domain
- **THEN** the request reaches the matching control or application route with its original path, method, body, and host-scoped cookies intact

#### Scenario: Forged or malformed original host
- **WHEN** a direct origin request omits the edge proof, presents an invalid proof, or names a host outside the configured control and visitor domains
- **THEN** Cellapp does not serve control or application content for that asserted host

#### Scenario: Direct Host compatibility
- **WHEN** a deployment without the third-party edge receives the browser Host directly through a trusted HTTPS ingress
- **THEN** the existing routing and authorization behavior remains available

#### Scenario: Published visitor application through the edge
- **WHEN** a visitor opens a published application through an authenticated edge request, signs in, and performs a same origin write
- **THEN** the authorization handoff returns to the requested application hostname, the page loads, and the same origin write passes its origin check

#### Scenario: Forged edge host or cross origin write
- **WHEN** the edge proof is invalid or a visitor write Origin differs from the authenticated application hostname
- **THEN** the server rejects the forged route or write without issuing access to another host
