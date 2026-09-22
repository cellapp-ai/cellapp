# Design

## Context

项目目前仅有 OpenSpec 配置，没有业务代码、测试或技术栈。动机与产品范围见 proposal.md。已确认免费静态托管、应用独立密钥、设备授权、本地构建上传；以下技术栈、会话期限和额度机制属于本提案的设计默认值，而非已上线能力。

## Goals / Non-Goals

**Goals:** 让 Skill 的发布流程可重试、应用地址稳定、发布完整切换；控制面身份和访客权限严格分开；为免费服务设置可执行的资源边界。

**Non-Goals:** 不执行上传代码，不接受服务端运行依赖，不构建通用 PaaS；不保证离线副本可撤销，不替应用代理外部 API，不隐藏已授权访问者下载到本地的文件。公开访问开关留待后续变更。

## Decisions

### 1. 单一服务边界，模块化实现

按用户确认，服务端采用 Go（标准库 HTTP 路由），CLI 采用 TypeScript 工作区：`apps/server` 提供控制 API、服务端渲染授权页及托管网关，`packages/cli` 提供确定性客户端，`skills/deploy-static-app` 描述 Agent 操作流程。控制元数据使用 PostgreSQL，产物使用用户指定的七牛云私有 S3 兼容对象存储；本地容器仅提供 PostgreSQL 和 HTTPS 代理。Bucket 使用项目配置，七牛 AccessKey/SecretKey 仅从服务进程的 `QINIU_ACCESS_KEY`、`QINIU_SECRET_KEY` 系统环境变量注入，不写入 `.env`；真实七牛验收保持待完成，自动化核心测试使用隔离存储替身。版本依赖在实现初始化时固定，不在规划中宣称具体版本兼容性。

相比首版拆成多服务，该结构减少部署组件；相比将产物存入数据库，私有对象存储更适合按文件提供内容。Skill 调用 CLI，不自行拼接身份协议，也不依赖某个 Agent 专用 API。

```text
Agent -> Skill -> CLI -> Control API -> PostgreSQL
                  |          |
                  |          v
                  +----> Private object storage

Browser -> App gateway -> Session check -> Active release -> Objects
Browser -> Owner login -> Device approval -> CLI polling
```

### 2. 所有者登录与设备授权

生产环境的所有者登录采用 GitHub OAuth Web Application Flow。服务端使用 Authorization Code、state 和 PKCE，交换 access token 后调用 GitHub `/user` API，并以固定 issuer `https://github.com` 加 GitHub 不可变数字用户 ID 标识账号；不依赖可变的用户名或邮箱。首版因此不维护密码体系。

本地开发可显式使用 `AUTH_MODE=dev`，访问登录入口时直接创建或复用唯一的固定开发所有者并签发正常浏览器会话，不接受请求参数指定身份。该模式只允许非生产环境和 localhost/loopback 控制域；生产环境或非本地域名配置 dev 模式时服务拒绝启动。测试使用受控 GitHub 替身或该开发模式，不提供可部署到公网的认证绕过入口。

CLI 请求设备授权，获得高熵 device secret、短 user code、验证链接、到期时间及轮询间隔。浏览器登录后显示客户端说明和匹配码，由用户明确确认。仅持有 device secret 的 CLI 能领取凭证；user code 不能直接换取凭证。授权请求默认 10 分钟过期，拒绝、过期、慢速轮询均有独立状态，批准与领取使用原子操作，领取仅一次。

部署 token 默认 30 天到期，服务端仅存摘要，可由浏览器凭证管理页撤销；CLI logout 撤销当前 token 并删除本机副本。CLI 将 token 写入用户私有配置目录（文件权限 0600），禁止写入项目、上传包或日志。所有应用管理 API 校验所有者；浏览器变更操作验证来源及 CSRF 防护。控制域与应用域分开，不向托管代码授予控制面 CORS 权限。

### 3. 本地构建与部署协议

CLI 接受明确的项目路径、构建命令、产物目录和 SPA 开关。纯 HTML 可跳过构建；已有明确项目配置可复用，不明确时 Skill 询问构建入口，不猜测上传仓库根目录。构建失败停止，不创建新线上版本。检测服务端专用输出时解释需要静态导出，不静默删减后声称成功。

仅收集产物目录内普通文件。客户端与服务端均校验相对路径、文件数、单文件和总大小、内容摘要，拒绝目录穿越、符号链接、保留的 `/_hosting/` 路径、`.env`、VCS/凭证文件等。敏感文件名检查不等于内容保密，Skill 必须说明前端打包的密钥会被访问者读取。

API 概念契约：`POST /device-authorizations`、设备轮询与确认端点、`POST /apps`、`GET /apps`、`POST /apps/:id/deployments`、文件上传端点、`POST /apps/:id/deployments/:id/publish`、密钥重置与应用删除端点。上传经受限流式 API 写入私有存储，不暴露对象读取 URL。

应用创建使用幂等请求 ID；部署记录绑定清单摘要和 idempotency key。同 key 不同内容返回冲突，重试相同请求复用资源。上传完整、逐项摘要验证且发布事务成功后才报告成功。项目配置保存 app ID、构建参数和输出目录，不含秘密；后续对同 app ID 发布。重复部署不返回旧密钥明文，只返回应用地址、版本及密钥未变提示；首次密钥由客户端安全生成，提交其原值经 TLS 供服务端摘要化，CLI 成功后向所有者展示一次。丢失则主动重置，不能读取明文旧密钥。

### 4. 数据模型、发布与删除

核心记录：Owner、DeviceAuthorization、DeploymentCredential、App（owner、随机不可复用 hostname ID、active deployment、key hash、key generation、状态）、Deployment（manifest、base version、状态、幂等 ID）、AccessSession（app、generation、expiry）、Usage/Reservation。

Deployment 状态为 uploading -> ready -> published，失败或过期终止；先写私有版本对象，再以事务更新 active deployment。发布使用 base version 比较，竞争发布中后完成的过期请求返回冲突，避免旧版本覆盖新版本。不会通过复制覆盖在线目录更新。

稳定地址为 `https://<app-id>.<apps-domain>/`；显示名称不影响地址。旧版本保留到有界清理期限，不提供本期用户回滚 UI；短期保留不保证长时间打开的旧页面能继续懒加载旧资源。删除先设置 tombstone、阻断访问和管理写入并撤销会话，再异步清理对象；hostname ID 永不复用。过期上传和未引用版本也异步回收，清理失败可重试。

### 5. 应用独立的密钥访问网关

应用使用独立 origin，且托管域与控制域使用不同注册域。密钥由密码学安全随机源生成，至少 128 位熵，以带服务端 secret 的摘要保存。网关的 `/_hosting/` 命名空间保留给访问页和验证端点。输入密钥通过 POST 验证，不放 URL、日志、分析事件或 Referer。

验证成功创建随机服务端会话并设置 host-only、Secure、HttpOnly、SameSite=Lax 的 `__Host-` cookie，默认 7 天到期。每次 HTML、JS、图片、HEAD、条件请求等资源请求都先验证 app、会话期限、key generation 及删除状态，再解析当前版本。无会话的文档导航显示输入页并保留合法站内目标；子资源返回 401，不把输入页当作 JS。返回路径只能为当前应用内路径。

密钥重置原子更新摘要并增加 generation；所有旧会话在下一次请求失效。重新部署不更换密钥或会话。对验证失败按 app 和来源限速，不同应用密钥不能互换。控制面授权不得接受分享密钥。

私有存储不允许匿名读，第一版受保护响应使用 `Cache-Control: private, no-store`，禁止共享 CDN 缓存绕过鉴权。禁止应用注册 Service Worker，防止其接管网关认证路径；应用内容以 CSP `worker-src 'none'` 执行此限制。已下载的内容无法追溯删除。平台保留端点拒绝被嵌入，验证 POST 检查来源。

静态文件存在则返回正确 MIME；仅在应用显式启用 SPA、请求接受 HTML 且路径属于页面导航时回退到 index.html，缺失资源保持 404。平台本身不提供应用数据 API；浏览器 localStorage 按应用 origin 隔离。

### 6. 免费额度与运维边界

配置项包含每所有者应用数、单文件字节、单部署文件数/字节、所有者总存储、每应用每 UTC 月流量、上传期限和请求速率。实现提供有限开发默认值，生产启动必须显式配置正数额度；具体商业数值不属于本期规格。

创建/上传前使用事务预留应用和存储配额，暂存与保留版本均计入存储；上传流超过限制立即终止，失败和清理释放预留。发布验证实际清单，不能仅信任客户端大小。网关在发送前原子预留响应体长度的流量，超过额度返回 429 和重置时间，不能让并发请求无限超支；HEAD 不计正文流量，失败响应与入口限流单独控制。拒绝更新不影响现有版本，流量耗尽只阻断内容分发，控制面仍可用。响应返回明确错误码及恢复办法，不自动收费。

记录部署状态、失败原因、资源使用与请求 ID，脱敏授权头、cookie、密钥和验证正文。提供运营侧停用应用的能力以处理资源滥用，不建设完整运营后台。

## Risks / Trade-offs

- [免费流量被滥用] → 有界配额、入口限流、私有存储及运营停用；具体成本上限需在上线配置中确定。
- [用户把秘密打包进前端] → Skill 提示并拒绝明显敏感文件；已授权访问者可读取前端代码，无法承诺内容扫描识别所有秘密。
- [禁止 Service Worker 影响 PWA] → 首版仅保证在线工具，不支持离线 PWA；以可靠的在线访问撤销为默认。
- [第三方内置浏览器清理 cookie] → 再次输入密钥即可，设备授权可使用系统浏览器。
- [网关逐次鉴权增加延迟] → 首版接受此成本；未来缓存优化必须保留每次访问的权限判定。
- [构建脚本在用户机器执行] → Skill 使用项目已有或用户明确指定的命令，遵守 Agent 本身的执行授权机制。

## Migration Plan

没有现有数据迁移。实现先提供本地完整联调环境与数据库建表迁移，再配置 GitHub OAuth App、私有存储、控制域及应用通配符 TLS，以少量账号验收授权、发布、直接资源保护、更新失败保留和密钥重置。上线失败回滚服务镜像；数据库迁移保持向后兼容，不自动销毁对象或元数据。首次生产部署需要运营配置，生成规划不代表已发布服务。

## Open Questions

- 生产 GitHub OAuth App、七牛 S3 区域/Bucket/凭证及域名的实际配置值。
- 免费额度具体数值、旧版本/临时上传清理期限；调整不得改变配额执行语义。
