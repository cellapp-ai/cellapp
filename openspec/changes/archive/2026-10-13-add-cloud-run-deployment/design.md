# Design

## Context

参见 [proposal.md](proposal.md)。当前 `apps/server/cmd/server` 用 `LISTEN_ADDRESS` 默认监听 `127.0.0.1:3000`，`WEB_ROOT` 指向 Go 同源提供的 Vite 产物；Host 被直接用于控制域与访客域授权，转发头不参与授权。`infra/Caddyfile` 只适用于本地内部证书和宿主机代理。现有 `-migrate` 使用 PostgreSQL advisory lock 并在事务中执行嵌入的初始 SQL；`-cleanup` 可独立运行，正常服务中的每分钟 ticker 在 Cloud Run 默认按请求计费/缩到零时不能保证准时运行。根目录 `Dockerfile` 在原检出中是用户未跟踪的空文件，本规划不修改它。Cloud Run 官方文档要求入口容器在 `0.0.0.0:$PORT` 监听；原生域名映射不支持通配证书，因此现有控制域与 `*.apps` 域需要外部 HTTPS 负载均衡和通配证书。

## Goals / Non-Goals

**Goals:**
- 单一镜像包含配套 server、console 和 Caddy，由一个发布命令完成构建、迁移、Cloud Run 修订版本发布及入口检查。
- 保持 Go 的 Host、Origin、Cookie、访客授权与私有存储语义，凭证通过秘密引用进入运行环境。
- 让失败停在可诊断的阶段，并提供复用上一镜像/修订版本的恢复路径。

**Non-Goals:**
- 不创建 PostgreSQL、S3、Secret、OAuth App、DNS、证书、负载均衡或定时触发器。
- 不将公开 CLI/Skill 打入运行镜像，不改变现有应用 API、数据库结构或访客域名模型。
- 不在 Cloud Run 容器内终止公网 TLS，也不依赖 Cloud Run 默认 `run.app` 域名承载产品流量。

## Decisions

### 单镜像、双进程入口

使用多阶段 Dockerfile：Node 阶段按根锁文件构建 `apps/web`，Go 阶段构建 server，最终精简镜像复制 server、`WEB_ROOT` 和固定版本 Caddy。生产 Caddy 只以明文 HTTP 监听 Cloud Run 注入的 `PORT`，将请求代理至 `127.0.0.1:3000` 的 Go 服务，保留 Host/路径；Go 只监听 loopback。入口负责启动两个进程、转发 SIGTERM、等待退出，并在任一进程异常退出时使容器失败。构建上下文排除 `.env*`、本地状态、`skills/`、构建输出和 Git 元数据；以明确允许的 `.env.example` 文档例外处理，最终镜像再检查秘密及工具链残留。

考虑过 Cloud Run 多容器 sidecar：平台支持，但需要两个镜像与服务 YAML/多容器编排，增加一次命令的配置面，console 仍由 Go 提供，因此不采用。也考虑让 Go 直接做 ingress：最省进程，却不满足本次明确包含 Caddy 的要求。

### 只将 Caddy 作为内部代理

外部 HTTPS 负载均衡负责控制域和通配访客域 TLS、DNS 与证书；Cloud Run 服务使用只允许内部及负载均衡入口的 ingress 设置，运行镜像不使用本地 Caddy 的 `tls internal`。控制域和应用域保持不同注册域，`CONTROL_ORIGIN` 与 `APPS_DOMAIN` 按真实公网域名配置，生产 `APP_PORT` 为空。Caddy 不能改写 Host，也不能从客户端提供的 `X-Forwarded-Host` 推断授权目标。Go 仍直接检查 Host 与 Origin。现有按直接连接地址限流在代理后会共享来源配额；本变更不引入信任转发 IP 的代码，部署文档要求据真实入口流量评估配额并回归安全边界。

考虑 Cloud Run 原生域名映射，但其不支持通配证书，无法服务动态应用子域；因此作为本次部署路径不适用。

### 构建、迁移、发布分阶段

在仓库脚本中接受明确的项目、区域、服务名、镜像仓库与非秘密生产配置，并以 Secret Manager 版本引用传递 `SECRET`、`DATABASE_URL`、GitHub OAuth secret 与 S3 凭证。脚本先做语法与资源引用预检，再通过 Cloud Build 构建带不可变标签的镜像，更新/执行使用同一镜像的 Cloud Run migration job（入口覆盖为 server `-migrate`），仅在 job 成功后发布 Cloud Run 服务。迁移 job 与服务使用同一服务账号、Direct VPC 网络与子网（或已有 VPC connector）、可选 Cloud SQL 连接、环境变量和秘密引用；测试环境默认 Direct VPC 以避免 connector 持续费用。脚本记录镜像 digest、job execution、服务 revision，并验证就绪与现有自定义入口。不得把密钥写进 Docker build args、镜像层、命令日志或仓库文件。

现有 `001_initial.sql` 的运行方式和未来迁移版本跟踪属于独立数据库变更；本次不新增 SQL 迁移。迁移命令只调用现有执行器。重复部署不删除数据；新迁移将来必须先满足向后兼容和版本跟踪规范。

### 清理与上线验收

发布脚本更新同一镜像的 `-cleanup` Cloud Run job，并核对预先配置的定时触发器；不能把服务内 ticker 当作唯一清理保证。若定时触发器不存在，脚本不报告生产就绪。上线检查通过既有控制域和一个无效应用子域核对 TLS、Host 路由及服务响应标记；真实 GitHub 登录、授权访客与私有 S3 的完整验收按 `docs/operations.md` 人工/隔离测试执行并如实记录，不能由启动探针替代。

## Risks / Trade-offs

- [首次服务部署时负载均衡可能尚未指向新服务] → 文档明确前置接入步骤；服务 revision 已发布但域名验证失败时返回非零并保留 revision，完成入口配置后重试验证。
- [迁移成功后服务发布失败] → 迁移必须保持向后兼容；继续运行旧 revision 并记录已执行迁移，修复发布后重试。回滚只切流量，不回滚数据库或对象。
- [Cloud Run 实例缩到零或被重启] → 周期清理使用预先配置的定时 job；服务内 ticker 仅作补充，不宣称精确定时。
- [镜像包含两个进程] → 启动器监督两者并处理 SIGTERM；镜像级测试覆盖单方异常退出与 Cloud Run 端口。
- [Cloud Run/CDN/代理改变 Host 或来源地址] → 验收用真实两个域名和未知 Host 验证路由；不信任转发头修正授权或限流，容量问题单独评估。
- [秘密引用或后端网络错误] → 发布前验证引用和权限，migration job 与服务共享连接设置，失败分阶段报告且隐藏秘密值。

## Migration Plan

1. 在现有 Google Cloud 项目预配生产 PostgreSQL、私有 S3、Secret Manager、GitHub OAuth、服务账号/网络、HTTPS 负载均衡及两个隔离域；访客域使用 DNS 授权的通配证书和通配 DNS，定时触发器指向清理 job。
2. 为当前服务记录可回退的镜像 digest/revision，按文档执行一次部署命令；它先迁移再发布，配置新 revision 处理流量。
3. 核对控制域、无效/有效访客域、GitHub 登录、CLI 授权发布、私有对象拒绝和失败路径；记录哪些是实际云环境检查。
4. 若新 revision 异常，将流量切回前一 revision；保留数据库、对象、Secret 与已执行的兼容迁移。修复后重用已构建镜像或重新构建并再次执行部署命令。
