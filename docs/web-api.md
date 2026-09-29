# 浏览器控制面 API

Cellapp Web 通过控制域下的 `/api/console` 访问浏览器专用 JSON 接口。此契约属于主仓库，不属于公开 CLI 的 Bearer HTTP 契约。

## 认证与来源

所有接口使用现有 `__Host-owner` Cookie（Secure、HttpOnly、SameSite=Lax、Path=/），不接受部署 Bearer 凭证替代浏览器会话。每个资源请求再次限定当前所有者。无会话或过期返回 `401 login_required`；资源不存在、已删除或属于其他所有者返回 404，不泄露其信息。数据库故障返回 500，不伪装成未登录或空列表。

所有 POST/DELETE 要求 `Origin` 与 `CONTROL_ORIGIN` 精确一致，缺失或不匹配返回 `403 origin_rejected`。有请求体时要求 `application/json`，否则返回 `415 json_required`；设备决策最多 1 MiB，只允许声明的字段和一个 JSON 值，无效内容返回 `400 invalid_json`。接口不开放携带凭证的 CORS。API 和页面使用 `private, no-store`。

GET `/auth/login?return=<本站管理路径>` 和 `/auth/callback` 保留现有登录流程。设备链接仍为 `/device?code=...`；打开或登录不会自动批准。CLI 的 `/apps`、设备创建/轮询、发布等路径及 Bearer 授权保持原契约。

## 接口

| 方法与路径 | 请求与成功响应 |
| --- | --- |
| GET `/api/console/session` | `{ownerId,authMode}`；模式为 `dev` 或 `github` |
| POST `/api/console/logout` | 无请求体；`{loggedOut:true}`，删除当前会话并清除 Cookie |
| GET `/api/console/apps` | 未删除自有应用数组，空时 `[]` |
| GET `/api/console/apps/{app}` | 应用字段及可空 `release` |
| DELETE `/api/console/apps/{app}` | 无请求体；`{deleted:true}`，使用既有墓碑和清理机制 |
| POST `/api/console/apps/{app}/key` | 无请求体；`{key}`，32 字节随机密钥的 64 字符小写 hex |
| GET `/api/console/credentials` | 有效未撤销自有凭证数组 `{id,createdAt,expiresAt}` |
| POST `/api/console/credentials/{credential}/revoke` | 无请求体；`{revoked:true}`；重复撤销自有记录安全成功 |
| POST `/api/console/device/decision` | `{code,decision}`；决策为 `approved` 或 `denied`；返回 `{status}` |

应用字段为 `id,name,activeDeployment,suspended,url`，其中 `activeDeployment` 可空。当前发布字段为 `id,status,createdAt,publishedAt,spa,fileCount,bytes`，其中 `publishedAt` 可空，时间均为 RFC3339；详情不返回发布历史、存储凭证或密钥摘要。凭证列表不返回 token，也不存在设备名称或头像字段。

## 错误与操作结果

错误结构为 `{error,message,details}`。应用 404 使用 `app_not_found`，凭证 404 使用 `credential_not_found`；无效决策使用 `400 invalid_decision`，设备码无效、过期或已处理使用 `400 invalid_code`。设备状态转换沿用原子 pending 条件，不覆盖既有结果。服务端内部错误为 `500 internal_error`，限流沿用 `429 rate_limited`。

重置密钥会原子更新摘要与访问代次，旧密钥及旧访客会话失效。新密钥只在这次响应中展示，服务端不保存明文，界面只保存在当前页面内存，离页/退出/会话失效后清除。禁止进入 URL、浏览器持久存储、日志或遥测。普通详情不能重新读取密钥。

危险操作确认后提交，服务端成功后更新界面。写请求响应丢失时不得自动重试或宣称未生效；先查询当前状态，或由用户主动再次重置（会使上次生成的密钥失效）。撤销只影响后续部署访问，不撤回已发布页面；删除不因应用或静态产物回滚而复活。
