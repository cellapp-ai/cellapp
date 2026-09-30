# Web control plane Spec Delta

## Purpose

为 Cellapp 应用所有者提供基于实际托管资源的浏览器控制面，使其通过现有身份与权限边界查看应用、管理访问密钥和部署凭证、处理设备授权，并获得准确的操作结果与失败恢复指引。

## ADDED Requirements

### Requirement: Owner session and management entry

Web SHALL 使用现有浏览器所有者会话，提供登录、退出和安全的登录后返回。登录后的默认入口 MUST 是真实应用列表；不得以产品介绍或模拟仪表盘替代管理入口。会话与私有数据 MUST 不向应用访客域开放。

#### Scenario: Open the control domain
- **WHEN** 用户访问控制域根地址
- **THEN** 已登录用户进入应用列表，未登录用户进入登录入口；会话读取期间显示加载状态，不显示未确认的私有信息

#### Scenario: Return from login
- **WHEN** 用户完成现有登录流程，返回目标是本站管理页面或含授权码的设备页面
- **THEN** 返回该安全页面并保留设备授权上下文；外部返回地址不能导致开放重定向

#### Scenario: Logout or expired session
- **WHEN** 用户退出，或浏览器 API 检测到会话缺失/过期
- **THEN** 退出使服务端当前会话失效；界面清除私有数据及本次新密钥并进入登录，API 返回明确 401 而非登录 HTML

### Requirement: Actual owner applications

Web SHALL 列出当前所有者未删除的真实应用，提供详情、访问地址和当前发布状态。MUST 区分加载、无应用、尚未发布、资源不可访问与请求失败；不得用模拟数据、部署历史或未经提供的配额统计填补内容。

#### Scenario: List and inspect applications
- **WHEN** 所有者打开应用列表并进入某个自有应用
- **THEN** 列表及详情来自服务端授权查询，详情显示正确访问地址和可空当前发布信息；刷新详情深链仍能进入对应页面

#### Scenario: No applications or no release
- **WHEN** 所有者没有应用，或应用尚无当前发布
- **THEN** 空列表提供简短的本机 Skill/CLI 部署指引，详情明确尚未发布，不展示虚构成功、可用发布或浏览器上传入口

#### Scenario: Query fails or resource is inaccessible
- **WHEN** 查询发生服务端/网络错误，或应用不存在、已删除、属于其他所有者
- **THEN** 错误有可操作反馈且不伪装为空数据；不可访问资源返回 404，不泄露其他所有者的信息

#### Scenario: Copy an application address
- **WHEN** 用户复制服务端提供的访问地址
- **THEN** 只有剪贴板操作成功后显示复制成功，失败提供明确反馈和手动复制方式；地址不包含分享密钥

### Requirement: Confirmed application access operations

Web SHALL 提供分享密钥重置和应用删除，提交前说明后果并要求确认，提交中阻止重复操作。服务端 MUST 在每次请求中验证所有者并沿用原子重置、访问代次和删除语义；成功结果必须以服务端响应为准。

#### Scenario: Reset a share key
- **WHEN** 所有者确认重置自有应用的分享密钥且操作成功
- **THEN** 服务端生成高熵新密钥并只保存摘要，原子使旧密钥与旧访客会话失效；界面仅在本次结果中展示/复制新密钥

#### Scenario: Sensitive result lifecycle
- **WHEN** 用户离开密钥结果页面、退出或会话失效
- **THEN** 新密钥从界面内存清除，不进入 URL、持久存储或日志；后续详情读取不能重新获得密钥明文

#### Scenario: Delete an application
- **WHEN** 所有者确认删除自有应用且服务端成功
- **THEN** 应用从可管理列表移除，原访问与后续部署不能继续使用已删除应用；后续清理沿用既有机制

#### Scenario: Cancel or uncertain write result
- **WHEN** 用户取消确认，或写请求失败且无法确定服务端是否已执行
- **THEN** 取消不会提交；结果不确定时不自动重试且不声称操作未生效，界面提供重新查询或主动再次操作的指引

#### Scenario: Unauthorized or concurrent mutations
- **WHEN** 请求针对非自有应用，或重置与删除并发发生
- **THEN** 非自有资源返回 404 且无变化，并发结果保持原有事务与访问失效约束，不复活已删除应用或旧访问权限

### Requirement: Deployment credential management

Web SHALL 展示当前所有者有效、未撤销的部署凭证的标识、创建与到期时间，并支持确认后撤销。MUST 不返回凭证明文、不索取部署 Bearer 凭证，不将记录描述为有实际设备元数据的设备列表。

#### Scenario: List active credentials
- **WHEN** 所有者打开部署凭证页面
- **THEN** 只显示自有有效凭证及真实元数据，加载、空列表与失败可辨别，响应不包含秘密

#### Scenario: Revoke an owned credential
- **WHEN** 所有者确认撤销自己的凭证
- **THEN** 成功后其无法再调用部署接口并从有效列表移除；重复撤销自有记录可安全成功，撤销其他所有者记录返回 404

### Requirement: Explicit device authorization decision

Web SHALL 接管现有设备授权浏览器入口并保留 CLI 创建和轮询协议。批准或拒绝 MUST 由已登录所有者主动提交，只能对有效待处理请求原子执行一次；不得因打开 URL 或登录成功自动批准。

#### Scenario: Login and approve a pending request
- **WHEN** 未登录用户打开带授权码的设备链接，完成登录并主动批准有效请求
- **THEN** 授权码保持，页面说明将授予部署访问；服务端绑定当前所有者，CLI 依原有轮询协议取得凭证，浏览器不取得该部署秘密

#### Scenario: Deny a pending request
- **WHEN** 所有者主动拒绝有效待处理请求
- **THEN** 页面显示拒绝结果，CLI 轮询获得原有拒绝结果且不会取得凭证

#### Scenario: Invalid expired or already processed request
- **WHEN** 授权码不存在、已过期、已处理，或两个决策并发提交
- **THEN** 页面报告明确状态/失败，不覆盖已确定决策，不重复签发或绑定到不同所有者

### Requirement: Browser API security and existing client compatibility

浏览器 API SHALL 统一使用 `/api/console` 路径前缀，并使用独立的会话认证边界、逐资源所有者鉴权和同源写请求检查。所有写请求 MUST 拒绝缺失或不匹配控制域的 Origin；有请求体时限制大小并校验输入。既有 CLI Bearer API、OAuth 回调、设备轮询和应用网关 MUST 保持原有协议与权限边界。

#### Scenario: Uniform browser API prefix
- **WHEN** Web 请求会话、退出、应用管理、部署凭证或设备授权决策接口
- **THEN** 所有这些浏览器 JSON 接口均位于 `/api/console` 下，页面路由与既有 CLI 接口继续使用各自路径

#### Scenario: Same origin owner mutation
- **WHEN** 已登录所有者从控制域提交合法管理写请求
- **THEN** 请求经会话、来源、输入和资源归属验证后执行，私有响应不可被共享缓存

#### Scenario: Cross origin or missing origin
- **WHEN** 写请求的 Origin 缺失、来自外站或应用访客域
- **THEN** 服务端拒绝且不改变应用、会话、凭证或设备授权状态

#### Scenario: Browser cookie cannot authenticate the CLI API
- **WHEN** 仅携带所有者 Cookie 调用既有 Bearer 应用 API，或从访客域请求浏览器 API
- **THEN** 不获得额外访问权限；原有合法 CLI、授权轮询与访客密钥访问仍按各自契约工作

### Requirement: Reproducible same origin Web delivery

Go 服务 SHALL 在控制域提供配套 Web 页面及静态资源，管理深链可刷新。服务启动 MUST 在启用正常服务时检查 Web 产物，不将缺失产物默认为成功运行；静态路由不得覆盖既有 API 或访客网关。页面不得内嵌私有数据，脚本/字体资源使用本站来源并禁止被嵌入。

#### Scenario: Build and deploy matching artifacts
- **WHEN** 开发者执行完整构建并使用文档规定的静态目录启动服务
- **THEN** 配套 Go 和 Web 产物可供登录与管理，页面、脚本、样式及字体加载成功；缺失指定产物时启动明确失败，独立维护命令无需 Web 产物

#### Scenario: Refresh and unknown routes
- **WHEN** 用户刷新有效管理深链，或访问未知 API、资源文件及非管理路径
- **THEN** 有效深链加载 Web 页面；未知路径返回 404，不返回伪装成成功的 HTML，现有 `/apps` 等接口继续返回其原有响应

#### Scenario: Verify secure browser integration
- **WHEN** 执行跨服务浏览器验收
- **THEN** 使用隔离数据库、真实 Web 产物和 HTTPS 控制域验证 Secure 会话及管理流程，记录身份/存储替身与真实外部服务的差别，未执行项目不得记为通过

### Requirement: Independently runnable frontend foundation

项目 SHALL 提供位于 `apps/web`、包名为 `@cellapp/web` 的前端应用及文档化的 `dev:web`、`preview:web`、`lint:web`、类型检查和构建入口；安装和构建结果 MUST 可由已提交的依赖声明及锁文件复现，构建产物不得包含秘密。

#### Scenario: Developer starts the console
- **WHEN** 开发者完成主仓库文档规定的依赖安装并执行 Web 开发入口
- **THEN** 可以在 loopback 地址访问具有 Cellapp 品牌、明确标题和正文的管理界面，不要求公开客户端子模块初始化；未连接后端时显示明确的会话服务不可用反馈，不伪造业务数据

#### Scenario: Preview a production build
- **WHEN** 开发者构建 Web 并通过文档化入口预览生成目录
- **THEN** 页面、脚本、样式、字体与品牌资产可以正常加载，字体不依赖运行时外部字体服务，失败构建返回非零退出码；真实管理验收通过同源控制服务进行

### Requirement: Usable responsive and accessible shell

Web 管理界面 SHALL 提供真实管理目的地的导航、可见键盘焦点、跳过正文入口、明确的语义结构及可访问的确认/反馈；在 320、390、768、1440 px 宽度下 MUST 保持主要内容可读且无页面整体横向溢出，非必要动效 MUST 尊重减少动态效果偏好。

#### Scenario: Keyboard navigation
- **WHEN** 用户仅使用键盘浏览管理界面、操作移动菜单和危险操作确认
- **THEN** 可以到达导航、正文与操作，焦点可见；菜单展开状态与可访问名称准确且可关闭，确认可以取消并恢复到合理焦点

#### Scenario: Narrow viewport and long content
- **WHEN** 用户在规定的小屏宽度查看页面，包括长应用名称、地址或凭证标识
- **THEN** 内容能换行或在适用的局部容器内滚动，不通过隐藏主要内容消除整体溢出

#### Scenario: Reduced motion or unavailable scripts
- **WHEN** 用户启用减少动态效果或浏览器无法执行 JavaScript
- **THEN** 非必要动态效果停用，无法执行脚本时获得需要启用脚本的有意义说明，不呈现虚假的可操作管理界面

### Requirement: Truthful foundation capability boundary

Web SHALL 准确表达当前可用管理能力；MUST NOT 展示伪造的应用、部署、额度或管理成功结果，也不得索取或持久化部署 Bearer 凭证。交付说明 MUST 区分独立前端预览与经真实控制域验证的登录/管理流程，构建和部署继续使用本机 Skill/CLI。

#### Scenario: User opens the foundation page
- **WHEN** 用户打开 Web 管理页面
- **THEN** 页面请求实际会话和资源接口，展示对应加载、未登录、空数据、成功或失败状态；所有操作指向实际管理行为，不以产品落地页替代控制面

#### Scenario: Developer evaluates control plane readiness
- **WHEN** 开发者阅读 Web 运行与交付说明
- **THEN** 可以找到真实控制域接入和本轮验收方法，明确哪些登录、管理和外部服务路径已执行验证以及哪些仍未执行，不以独立预览成功代表真实管理验收
