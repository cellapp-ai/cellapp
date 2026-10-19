# Spec Delta

## Purpose

允许远程 Agent 通过标准 MCP 连接 Cellapp，在应用所有者明确授权后查询应用、管理应用数据表、上传本地构建的静态产物并原子发布，同时使凭证、额度和失败恢复行为可预测。

## ADDED Requirements

### Requirement: Remote MCP interoperability
控制域 SHALL 提供 HTTPS Streamable HTTP MCP 服务，支持工具发现与调用，并公布实际支持的协议版本。Codex、豆包及其他兼容客户端 SHALL 能使用相同工具契约；不支持的协议版本或方法 MUST 返回协议错误，而非网页内容。

#### Scenario: Compatible agent connects
- **WHEN** 兼容客户端连接 MCP 地址并列出工具
- **THEN** 客户端获得可调用的工具名称、输入结构和说明，不需要解析现有 REST 路径

#### Scenario: Unsupported protocol or malformed request
- **WHEN** 客户端发送不支持的协议版本、未知方法或无效参数
- **THEN** 服务返回符合已协商协议的错误，不执行应用管理操作

### Requirement: Owner approved MCP authorization
远程 MCP 私有工具 SHALL 使用独立的 OAuth 授权码流程、PKCE、授权发现与浏览器所有者同意。访问凭证 MUST 绑定 MCP 资源、客户端、所有者、授予权限和有效期；服务 SHALL 支持撤销，并在每次工具调用时校验凭证与权限。

#### Scenario: First agent authorization
- **WHEN** 未授权 Agent 连接并请求所有者工具，用户登录后确认客户端与所请求的权限
- **THEN** Agent 取得仅可访问获准 MCP 工具的凭证，并可完成当前请求

#### Scenario: Rejected, expired or revoked authorization
- **WHEN** 用户拒绝授权，授权码或凭证过期，或用户撤销该连接
- **THEN** 相关工具调用被拒绝且不修改应用，客户端可重新发起授权

#### Scenario: Credential from another surface
- **WHEN** 调用者仅持有浏览器 Cookie、CLI 设备凭证、访客分享密钥或其他资源的令牌
- **THEN** MCP 所有者工具拒绝该调用，不将这些凭证升级为 MCP 权限

### Requirement: Scoped application discovery and creation
MCP SHALL 提供 `list_apps` 和 `create_app` 工具；结果只包含当前所有者可见的应用及部署状态，不返回凭证或访问密钥明文。创建操作 SHALL 使用稳定请求标识实现幂等；同一标识对应不同输入 MUST 冲突。

#### Scenario: List and create own application
- **WHEN** 获准所有者查询应用并以稳定请求标识创建新应用
- **THEN** 只返回该所有者的应用，重试创建返回同一应用及稳定地址

#### Scenario: Cross-owner target
- **WHEN** Agent 指定另一个所有者的应用或使用只读授权创建应用
- **THEN** 服务拒绝或隐藏目标，且不泄露其部署或密钥

### Requirement: Scoped application data management
MCP SHALL 提供 `data_status`、`data_reconcile`、`data_plan`、`data_apply` 和 `data_verify` 工具，复用现有应用数据服务的版本化清单、计划、执行与验证语义。状态查询须有 `mcp:read` 权限；协调、计划、应用与验证须有独立的 `mcp:data` 权限和目标应用所有者身份。工具 MUST NOT 接受任意 SQL、数据库连接信息或跨应用 schema 名称。

#### Scenario: Create tables for own application
- **WHEN** 获准所有者为自有应用提交合法的新版本表与操作权限清单，先取得计划并应用相同内容
- **THEN** 系统仅在该应用隔离的数据范围内创建或修改表，记录版本，并返回独立于静态发布的执行和验证结果

#### Scenario: Destructive change requires explicit intent
- **WHEN** 计划包含删除表或列，而调用未明确允许破坏性变更或应用内容不同于已审阅计划
- **THEN** 系统拒绝应用，保留现有表和版本，并返回需要重新审阅的原因

#### Scenario: Provisioning failure or cross-owner request
- **WHEN** 应用数据尚未就绪、数据库暂时不可用，或 Agent 指定其他所有者的应用
- **THEN** 工具返回可定位的状态或拒绝操作；重试协调保持幂等，不返回数据库凭证、SQL 内情或其他应用数据

### Requirement: Bounded static artifact transfer
MCP SHALL 提供部署准备、分块上传与文件完成工具，使 Agent 可上传本机生成的静态文件；服务端 MUST NOT 获取本机路径、执行构建命令或运行上传程序。清单、路径、文件摘要、额度及总大小 SHALL 满足现有静态发布限制；不完整或失配的文件 MUST NOT 可发布。

#### Scenario: Upload valid local build
- **WHEN** Agent 在本地构建静态产物并提交含 `index.html` 的清单与全部匹配分块
- **THEN** 服务将文件记录为已上传，并允许该部署进入发布确认

#### Scenario: Unsafe or incomplete artifact
- **WHEN** 清单含敏感路径、越界路径、超限文件，或分块缺失、越界、重复内容不一致、摘要不符
- **THEN** 服务拒绝对应操作；已发布版本保持有效，上传状态可供恢复

### Requirement: Recoverable and atomic MCP publication
MCP SHALL 提供 `get_deployment` 和 `publish_deployment` 工具，复用现有部署幂等标识、基准版本冲突及原子发布语义。仅在服务确认发布成功后才返回成功与应用地址；失败后 SHALL 返回可定位的状态与重试信息。

#### Scenario: Publish complete deployment
- **WHEN** 获准所有者上传完整产物并发布基于当前版本的部署
- **THEN** 应用地址保持稳定，新版本原子生效，工具返回部署标识和地址

#### Scenario: Interrupted upload or concurrent publication
- **WHEN** 上传中断后 Agent 查询部署状态并重试，或另一个部署先发布
- **THEN** 相同内容可恢复上传；版本冲突不会覆盖先发布版本，也不会错误报告成功

### Requirement: Agent connection guidance
Cellapp SHALL 提供 Codex 与豆包的远程 MCP 连接步骤、授权、本地静态构建及应用数据清单说明，并公开首版工具范围、额度与失败恢复方法。连接示例 MUST NOT 要求用户把令牌或数据库凭证放入项目文件或聊天文本。

#### Scenario: Agent setup from published guide
- **WHEN** 用户按文档将 MCP 地址加入支持的 Agent 并完成浏览器授权
- **THEN** Agent 可发现工具、列出应用，并在有本地产物的环境中完成一次测试发布
