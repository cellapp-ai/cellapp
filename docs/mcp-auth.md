# Cellapp MCP 授权

Cellapp 在控制域提供远程 MCP 地址 `https://<控制域>/mcp`。连接客户端无需读取 CLI 凭证；第一次连接时，服务返回 Bearer 授权提示，指向 `/.well-known/oauth-protected-resource/mcp`。服务同时提供 `/.well-known/oauth-protected-resource` 和 `/.well-known/oauth-authorization-server`，用于发现资源、授权、令牌、注册和撤销地址。

## 连接流程

1. 客户端通过 `POST /oauth/register` 注册公有客户端，或使用一个公网 HTTPS 客户端 ID 元数据文档。注册需要 1–5 个唯一的回调地址，地址必须是 HTTPS，或本机回环地址上的 HTTP。客户端名称不超过 100 字符。只支持授权码、PKCE S256 和无客户端密钥的公有客户端。
2. 客户端打开 `GET /oauth/authorize`，携带 `response_type=code`、`client_id`、精确注册的 `redirect_uri`、`resource=https://<控制域>/mcp`、`scope`、`code_challenge` 和 `code_challenge_method=S256`。用户先登录 Cellapp，再查看客户端名称和请求权限，明确允许或拒绝。
3. 允许后，授权码发送到已注册回调地址。客户端向 `POST /oauth/token` 提交授权码、同一客户端 ID、回调地址、资源和 PKCE verifier。授权码有效 5 分钟且只能交换一次。
4. 客户端在每次 `/mcp` 请求的 `Authorization: Bearer` 头中携带访问令牌。令牌有效 1 小时；当前版本不签发刷新令牌，到期后重新经浏览器授权。令牌只对 MCP 资源有效，不能用于 CLI REST API。

权限分为 `mcp:read`（查询应用与部署）、`mcp:deploy`（创建应用、上传和发布）及 `mcp:data`（管理应用数据表）。服务每次 MCP 请求检查令牌有效期和撤销状态，每次工具调用检查所需权限以及应用所有者。浏览器 Cookie、CLI 设备令牌和应用访客密钥都不能替代 MCP 令牌。

## 连接管理与恢复

控制台浏览器会话可通过 `GET /api/console/mcp/connections` 列出已授权客户端，通过 `POST /api/console/mcp/connections/{clientId}/revoke` 撤销该客户端的授权、令牌和未交换的授权码。修改请求必须来自控制域。客户端也可向 `POST /oauth/revoke` 提交自己的 `client_id` 和令牌。CLI 凭证在另一列表中管理，不受 MCP 撤销影响。

授权失败时，客户端应重新从资源元数据开始发现并发起浏览器授权。若回调地址、客户端、资源或 PKCE 参数不匹配，应修正客户端配置后重新连接；重复提交旧授权码不会成功。客户端元数据文档拉取只允许公网 HTTPS、限制响应大小和时间，并拒绝跳转。注册接口受控制域请求限速；不要把令牌、数据库凭证或回调中的授权码写进项目文件或聊天内容。
