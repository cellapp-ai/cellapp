# 控制域与应用域接口

Cellapp 的控制入口位于 HTTPS 控制域（默认 `https://cellapp.ai`）。CLI 可用 `--origin` 或 `CELLAPP_ORIGIN` 指定另一个 HTTPS 控制域。应用访客入口位于独立的应用域。下表描述本次兼容服务和 Worker 上线后的路径；当前部署状态见[验收记录](control-api-verification.md)。方法、请求字段、成功形状和错误码以[公开 HTTP v3 契约](../skills/contracts/http-v3.json)及[浏览器 API](web-api.md)为准。

| 域 | 路径 | 调用者与凭据 | 用途 |
| --- | --- | --- | --- |
| 认证 | `POST /auth/device`、`POST /auth/device/poll` | CLI，设备秘密 | 创建授权请求、轮询并领取一次性部署凭证；所有者须在浏览器明确批准 |
| 认证 | `DELETE /auth/credentials/current` | CLI，Bearer 部署凭证 | 撤销当前部署凭证 |
| 认证 | `/auth/login`、`/auth/callback` | 浏览器 | 所有者登录与 OAuth 回调 |
| 控制台 | `/api/console/*` | 浏览器，所有者 Cookie 与同源校验 | 应用、访问授权、凭证和资料管理 |
| 程序管理 | `/api/control/v1/limits`、`/api/control/v1/apps*` | CLI 或其他程序客户端，Bearer 部署凭证 | 额度、应用、部署、数据管理；URL 的 `v1` 与契约文件版本分别演进 |
| 应用内部 | `/_cellapp/*`、`/rest/v1/*` | 应用访客，应用域会话与访问检查 | 登录交接、申请/兑换访问、应用内身份与受管表数据 |

`/_cellapp/*` 不提供 CLI 管理接口；可用路径和响应按[浏览器 API 的应用访问章节](web-api.md#应用访问与数据接口)执行。MCP 尚无服务端入口，未来如支持 MCP 连接，应另行设计协议端点，不把现有 REST 路径直接命名为 MCP。

## 兼容期

旧 CLI 的 `/device-authorizations`、`/device-authorizations/poll`、`/credentials/current`、`/limits`、`/apps` 及其全部子路径继续可用，和规范路径共用处理逻辑。服务端不会把旧请求重定向到新路径；同一 `requestId` 跨路径重试沿用幂等规则。旧路径退役需要独立变更。新版 CLI 只发送规范路径；旧版 CLI 仍可使用旧路径。生产发布须先让兼容服务和 Worker 路由生效，再发布新版 CLI。

Cookie 不能替代 Bearer 调用程序管理接口；Bearer 不能替代浏览器会话调用控制台接口。应用域不会承载控制路径。未知 `/api/control/v1/*` 返回 404，不回退为官网或应用静态页面。根域路径路由与发布暂停配置见[切换清单](root-domain-cutover.md)。
