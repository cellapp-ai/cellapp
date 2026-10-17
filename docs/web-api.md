# 浏览器控制面 API

Cellapp Web 从 `https://cellapp.ai/dashboard` 通过同域 `/api/console` 访问浏览器专用 JSON 接口。官网公开页面由 Pages 提供，管理页面和 API 由路径路由转发到控制服务。此契约属于主仓库，不属于公开 CLI 的 Bearer HTTP 契约。

## 认证与来源

所有接口使用现有 `__Host-owner` Cookie（Secure、HttpOnly、SameSite=Lax、Path=/），不接受部署 Bearer 凭证替代浏览器会话。每个资源请求再次限定当前所有者。无会话或过期返回 `401 login_required`；资源不存在、已删除或属于其他所有者返回 404，不泄露其信息。数据库故障返回 500，不伪装成未登录或空列表。

所有 POST/PUT/DELETE 要求 `Origin` 与 `CONTROL_ORIGIN` 精确一致，缺失或不匹配返回 `403 origin_rejected`。有请求体时要求 `application/json`，否则返回 `415 json_required`；设备决策和资料更新最多 1 MiB，只允许声明的字段和一个 JSON 值，无效内容返回 `400 invalid_json`。接口不开放携带凭证的 CORS。API 和页面使用 `private, no-store`。

GET `/auth/login?return=<本站管理路径>` 和 `/auth/callback` 保留现有登录流程，回调 URL 为 `https://cellapp.ai/auth/callback`。管理路径统一位于 `/dashboard`，包括 `/dashboard/applications/{app}`、`/dashboard/credentials`、`/dashboard/device`、`/dashboard/profile` 与 `/dashboard/login`。设备链接为 `/dashboard/device?code=...`；打开或登录不会自动批准。CLI 的设备流程位于 `POST /auth/device`、`POST /auth/device/poll`，当前凭证撤销为 `DELETE /auth/credentials/current`，Bearer 管理位于 `/api/control/v1/*`；旧根路径在兼容期仍可调用，且保持同一字段、错误与权限。旧 `console.cellapp.ai` API 不再转发。

## 接口

| 方法与路径 | 请求与成功响应 |
| --- | --- |
| GET `/api/console/session` | `{ownerId,authMode}`；模式为 `dev` 或 `github` |
| GET `/api/console/profile` | `{displayName}`；仅返回当前登录账户的个人资料 |
| PUT `/api/console/profile` | `{displayName}`；返回更新后的 `{displayName}`；允许空字符串清除，最多 80 个 Unicode 字符，不允许首尾空白或控制字符 |
| POST `/api/console/logout` | 无请求体；`{loggedOut:true}`，删除当前会话并清除 Cookie |
| GET `/api/console/apps` | 未删除自有应用数组，空时 `[]` |
| GET `/api/console/apps/{app}` | 应用字段及可空 `release` |
| GET `/api/console/apps/{app}/data` | `{state,version,digest}`；应用数据库不可用返回 503，不显示为空表 |
| GET `/api/console/apps/{app}/access` | `{requests,grants}`；用户 ID 均为应用内随机 ID |
| POST `/api/console/apps/{app}/visibility/{public|restricted}` | 无请求体；`{visibility}` |
| POST `/api/console/apps/{app}/requests/{user}/{approve|deny}` | 无请求体；`{status}`；重复同决策可安全重试，反向决策返回 409 |
| POST `/api/console/apps/{app}/grants/{user}/revoke` | 无请求体；`{revoked:true}`；撤销后旧密钥不能让该用户自行重新授权 |
| DELETE `/api/console/apps/{app}` | 无请求体；`{deleted:true}`，使用既有墓碑和清理机制 |
| POST `/api/console/apps/{app}/key` | 无请求体；`{key}`，32 字节随机密钥的 64 字符小写 hex |
| GET `/api/console/credentials` | 有效未撤销自有凭证数组 `{id,createdAt,expiresAt}` |
| POST `/api/console/credentials/{credential}/revoke` | 无请求体；`{revoked:true}`；重复撤销自有记录安全成功 |
| POST `/api/console/device/decision` | `{code,decision}`；决策为 `approved` 或 `denied`；返回 `{status}` |

应用字段为 `id,name,namespace,slug,aliasUrl,activeDeployment,suspended,url,visibility,dataState`，其中 `activeDeployment` 可空。当前发布字段为 `id,status,createdAt,publishedAt,spa,fileCount,bytes`，其中 `publishedAt` 可空，时间均为 RFC3339；详情不返回发布历史、存储凭证或密钥摘要。凭证列表不返回 token，也不存在设备名称或头像字段。

## 错误与操作结果

错误结构为 `{error,message,details}`。应用 404 使用 `app_not_found`，凭证 404 使用 `credential_not_found`；无效显示名称使用 `400 invalid_display_name`；无效决策使用 `400 invalid_decision`，设备码无效、过期或已处理使用 `400 invalid_code`。设备状态转换沿用原子 pending 条件，不覆盖既有结果。服务端内部错误为 `500 internal_error`，限流沿用 `429 rate_limited`。

轮换访问密钥会原子更新摘要。旧密钥不能再授权新用户，已登录用户的既有授权仍有效；单独撤销某位用户后，旧密钥不能让该用户自行恢复授权。新密钥只在这次响应中展示，服务端不保存明文，界面只保存在当前页面内存，离页/退出/会话失效后清除。禁止进入 URL、浏览器持久存储、日志或遥测。普通详情不能重新读取密钥。

危险操作确认后提交，服务端成功后更新界面。写请求响应丢失时不得自动重试或宣称未生效；先查询当前状态，或由用户主动再次重置（会使上次生成的密钥失效）。撤销只影响后续部署访问，不撤回已发布页面；删除不因应用或静态产物回滚而复活。

## 应用访问与数据接口

应用域的静态页面与资源先检查主站登录和应用授权。未登录的导航跳转至主站登录并通过一次性票据回到原路径；非导航请求返回 `401 login_required`。受限应用已登录但未授权时显示申请或输入密钥页面；待审核和已拒绝分别显示状态及重新申请入口，`/_cellapp/me` 分别返回 `403 access_pending` 与 `403 access_denied` 且不暴露用户 ID，资源和 `/rest/v1` 返回 403。`/_cellapp/me` 返回稳定的应用内随机 `{id,access}`，不暴露平台用户 ID 或 OAuth subject。应用域 `__Host-cellapp` Cookie 仅属于当前 host，并受主站会话有效性约束。

`POST /_cellapp/request` 和 `POST /_cellapp/redeem` 只接受本应用 Origin。访问密钥明文只用于该次兑换，不放入 URL、持久状态或日志。公开应用仍要求登录；从公开切回受限时，只有已有授权的用户继续访问。

`/rest/v1/{table}` 支持 `supabase-js` 2.117.3 的表级 select、insert、update、delete、eq、order、range 与写后 select。每次请求重新检查登录、应用访问、表授权及行策略，浏览器提交的 Authorization 和 profile 不生效。Auth、Storage、Realtime、Functions、RPC 等路径返回 `501 unsupported_feature`；其他 Supabase API 不在本期范围内。所有者通过 Bearer CLI 的 `/api/control/v1/apps/{app}/data`、`/plan`、`/apply`、`/verify` 管理表；表版本和静态发布独立。

应用显示名允许重复。`namespace` 是账户的用户 slug，`slug` 是应用 slug，新应用 URL 为 `https://{slug}-{namespace}.<APPS_DOMAIN>`，组合后的单个 DNS 标签最长 63 位。用户 slug 至少 3 位；两种 slug 均使用小写字母开头，后续可包含小写字母、数字、连字符，且拒绝 `accounts`、`cloud`、`login`、`app` 等平台保留词。组合域名全局唯一并在删除、改名后保留占用。控制域 `aliasUrl` 对应 `@namespace/slug`，临时重定向到当前 URL。内部 ID 仍用于登录授权、应用数据和部署隔离；旧 ID 域名及先前分配的组合域名可继续访问原应用。历史别名过长或冲突时保留旧 ID 域名，设置合规 slug 后启用组合域名。新所有者首次创建应用可通过 CLI 指定 `namespace` 和 `slug`，控制台的 Profile 与应用详情页可分别修改这两个值。
