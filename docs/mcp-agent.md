# 用 Agent 连接 Cellapp

Cellapp 的远程 MCP 入口是 `https://cellapp.ai/mcp`，使用 Streamable HTTP。服务端当前按 MCP `2025-06-18` 实现并以无状态请求处理；Agent 必须能访问该 HTTPS 地址、读取本机构建产物并发起浏览器 OAuth 授权。实际连接前请先确认目标环境已经部署 MCP 路由；仓库代码或本地测试通过不代表公开地址已上线。授权详情见 [MCP 授权](mcp-auth.md)。

## Codex

在本机 Codex CLI 中添加连接，然后打开浏览器授权：

```sh
codex mcp add cellapp --url https://cellapp.ai/mcp --oauth-resource https://cellapp.ai/mcp --oauth-client-registration dcr
codex mcp login cellapp --scopes mcp:read,mcp:deploy,mcp:data
codex mcp list
```

授权页会显示客户端名称和所需权限。只需查看应用时可以仅请求 `mcp:read`；发布需要 `mcp:deploy`，管理表需要 `mcp:data`。Codex CLI 与 IDE 扩展共用 MCP 配置。连接失效时再次执行 `codex mcp login cellapp`。这些命令使用本机 Codex 的安全授权存储，不要求把令牌放进项目配置。

## 豆包与其他 Agent

在支持远程 Streamable HTTP MCP 且支持 OAuth 授权码、PKCE S256、动态客户端注册或客户端 ID 元数据的客户端中，填写同一 `/mcp` 地址。客户端应读取 `/.well-known/oauth-protected-resource/mcp` 和 `/.well-known/oauth-authorization-server`，请求资源 `https://cellapp.ai/mcp`，并在浏览器中完成所有者同意。

火山引擎 AgentKit 网关的“导入已有 MCP Server”可以导入 Streamable HTTP 服务，但其示例要求预先配置出站凭据。若使用 AgentKit 网关，应选择可将**用户委托 OAuth**凭据传给下游的连接方式并在候选环境验证；固定 API Key/静态 Bearer 出站配置不能代表 Cellapp 用户，也不能长期使用一小时到期的访问令牌。若豆包产品界面未提供用户委托 OAuth 连接，应报告该客户端尚未通过验收，勿粘贴个人令牌作为网关共享凭据。

## 工具与权限

| 工具 | 用途 | 权限 |
| --- | --- | --- |
| `list_apps` | 列出自己的应用和当前发布版本 | `mcp:read` |
| `create_app` | 用稳定 `requestId` 新建应用 | `mcp:deploy` |
| `prepare_deployment` | 校验清单、额度与基准版本，返回部署 ID 和分块上限 | `mcp:deploy` |
| `upload_file_chunk` | 上传单个 base64 分块 | `mcp:deploy` |
| `complete_file` | 合成并校验一个文件 | `mcp:deploy` |
| `get_deployment` | 查询部署及每个文件的下一个分块偏移 | `mcp:read` |
| `publish_deployment` | 原子切换到完整部署 | `mcp:deploy` |
| `data_status` | 查询应用数据就绪状态和表版本 | `mcp:read` |
| `data_reconcile` | 重试应用数据开通 | `mcp:data` |
| `data_plan` | 审阅版本化表清单及其摘要 | `mcp:data` |
| `data_apply` | 应用与已审阅摘要一致的表清单 | `mcp:data` |
| `data_verify` | 通过应用数据 API 探针检查版本 | `mcp:data` |

`create_app` 输入包含 `name` 和 32 位小写十六进制 `requestId`，可选 `namespace`、`slug`。同一请求标识与相同内容可重试；不同内容会冲突。工具不返回应用访问密钥；访问设置由现有所有者控制台管理。所有工具只操作当前授权所有者的应用，不提供任意 SQL 或物理数据库创建能力。

## 发布本地静态产物

1. 在 Agent 能读取文件的机器上运行项目原有构建命令，得到含 `index.html` 的静态目录。服务端不执行构建，也不读取 Agent 的本地路径。
2. 为每个文件计算字节长度和 SHA-256，调用 `prepare_deployment`，提交 `appId`、稳定 `requestId`、`manifest`（每项含 `path`、`size`、`sha256`）、当前 `baseVersion` 和可选 `spa`。首次发布的 `baseVersion` 省略或为 `null`。清单按服务端规则校验：单文件默认最多 10 MB，部署默认最多 50 MB，路径不得包含敏感文件或隐藏目录。
3. 对清单中的每个文件按偏移切分，单块原始数据最多 256 KiB。`upload_file_chunk` 传入 `appId`、`deploymentId`、`fileIndex`、`offsetBytes`、`sizeBytes`、块的 `sha256` 与 `dataBase64`。分块必须从偏移 0 连续上传。同偏移相同内容可重试；内容不同会冲突。
4. 每个文件上传完调用 `complete_file`（`appId`、`deploymentId`、`fileIndex`）。服务端会从私有暂存块流式合成，并对照清单再次校验文件长度和 SHA-256。
5. 全部文件完成后调用 `publish_deployment`。仅在原子发布事务确认后返回 `published` 和稳定应用 URL。应用数据表版本与静态发布结果分别报告。

断线后调用 `get_deployment`；其 `nextOffsetBytes` 按文件索引给出已记录的连续字节数，再从该偏移重试。若部署过期、发布暂停或 `baseVersion` 与当前版本冲突，应先查询应用状态，再创建新的部署；失败不会替换已发布版本。上传块和已完成文件的暂存块由服务端定期清理。

## 建立应用数据表

应用的数据**物理 PostgreSQL 数据库由运维配置**。`create_app` 只会为该应用开通隔离 schema；Agent 使用现有受限版本化清单创建表和权限，不能提交 SQL、数据库地址或管理员密码。

先调用 `data_status`；若尚未就绪，用 `data_reconcile` 重试。然后向 `data_plan` 提交 `appId` 和 `manifest`，清单结构与 [`cellapp.data.json`](app-data-operations.md) 一致。清单中的 `version` 从 1 开始，表、列、索引和 `policies` 受服务端白名单约束；没有明确授予的前端操作保持拒绝。记录计划的 `digest`、`operations` 和 `requiresDestructive`，供用户审阅。

确认后调用 `data_apply`，提交**完全相同**的 `manifest`、`reviewedDigest` 和 `appId`。若计划删除表或列，清单本身须设置 `allowDestructive: true`，调用还须单独设置 `allowDestructive: true`；少任何一步都会被拒绝。最后调用 `data_verify`，通过当前应用的数据 API 探针确认版本。若 API 探针暂时失败，表变更可能已经提交；先查 `data_status` 与 `data_verify`，再用相同版本和内容安全重试，不要把静态发布成功误认为建表成功。

## 验收状态

直接在浏览器打开 `/mcp` 会得到 HTTP 401 与 `invalid_token`，因为浏览器尚未带 MCP 令牌。这是客户端发现 OAuth 的起点；若连接器随后失败，应检查客户端是否访问了 OAuth 元数据、动态注册与授权端点，而不能仅凭该页面判断令牌错误。不要把令牌拼入 MCP URL。

本仓库的 Go MCP SDK、隔离 PostgreSQL、隔离 PostgREST 与 Worker 路由测试已覆盖主要协议和业务路径。`https://cellapp.ai/mcp` 已通过公开的授权挑战与 OAuth 元数据检查；真实 Codex 与豆包浏览器授权、上传和建表连接仍需使用隔离测试账户单独验收。当前公开地址可用于发起连接，端到端工具流程尚未标记为已验证。
