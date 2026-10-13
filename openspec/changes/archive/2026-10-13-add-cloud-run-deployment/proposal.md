# Proposal

## Why

Cellapp 目前只有本地 Compose 代理和配套 Go/Web 构建说明，缺少可复现的生产镜像与 Cloud Run 发布入口。部署者需要手动组合 server、console 和 Caddy，并自行处理迁移、密钥注入及域名路由，容易得到能启动却无法提供访客域服务的修订版本。

## What Changes

- 增加根目录多阶段 Dockerfile，将同一源码版本的 Go server、Web console 静态产物和 Caddy 打入可用于 Cloud Run 的镜像；Caddy 监听 Cloud Run 注入的端口并转发给容器内 Go 服务。
- 增加一次执行的 Cloud Run 部署脚本：校验预先配置的项目、区域、服务、域名、数据库、私有 S3 和 Secret 引用，构建并推送镜像，执行数据库迁移，发布服务并核对结果；任一步失败即停止，不报告成功。
- 增加生产代理配置、构建上下文排除规则及部署/回滚说明，明确控制域与通配应用域所需的现有 HTTPS 负载均衡、DNS、证书、GitHub OAuth 和外部持久服务前置条件。
- 增加镜像与部署验证，覆盖端口和 Host 转发、配套 Web 产物、秘密不入镜像、迁移失败阻断及控制域/访客域的真实验收路径。现有本地开发入口继续可用。

## Capabilities

### New Capabilities

- `cloud-run-deployment`: 可复现的 Cloud Run 应用镜像、发布命令、前置条件、安全边界与故障恢复。

### Modified Capabilities

无。现有控制面、存储和公开客户端协议不改变。

## Impact

涉及根目录 `Dockerfile`、`.dockerignore`、Cloud Run 部署脚本及配置、生产 Caddy 配置、部署文档和相关检查。Go 服务继续提供 Web console 与应用网关；不修改公开 `skills/` 子仓库。需要预先存在 PostgreSQL、私有 S3、Secret Manager 引用、两个隔离的生产注册域及支持通配访客域的 HTTPS 入口及定时清理触发器；本变更不创建这些云资源，也不在提案阶段部署。
