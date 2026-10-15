# 根域名与 Dashboard 切换清单

本清单适用于 `unify-root-domain-dashboard`。当前实现使公开官网继续由 Cloudflare Pages 服务，`https://cellapp.ai/dashboard` 和下表中的管理路径由 `cellapp-edge` Worker 转发到 Cloud Run。`*.ce11.app` 的访客静态分发仍走现有 Worker。切流前必须在受控候选 Host 完成验收；Git 推送不会发布 Pages、Worker、Cloud Run 或 npm。

## 路径边界

| 来源 | Worker 路由 | Cloud Run 用途 |
| --- | --- | --- |
| `cellapp.ai` | `/dashboard*` | 页面、`/dashboard/assets/*`、`/dashboard/logos/*`、浏览器表单 |
| `cellapp.ai` | `/api/console*` | 浏览器会话 API |
| `cellapp.ai` | `/auth*` | OAuth、登出与应用交接 |
| `cellapp.ai` | `/device-authorizations*` | 设备授权创建与轮询 |
| `cellapp.ai` | `/apps*`、`/limits*`、`/credentials/current*` | Bearer 部署 API |
| `cellapp.ai` | `/a/*` | 应用别名跳转 |
| `*.ce11.app` | `/*` | 访客网关和受保护资源 |

星号使带查询参数的精确入口也命中 Cloudflare 路由。Worker 的 `controlRoute` 再按路径边界拒绝 `/dashboard-extra`、`/apps-extra` 等近似路径。`/`、`/deploy/`、`/pricing/`、`/docs/`、`/blog/` 及官网资源不挂 Worker 路由。`/health` 仅供 Cloud Run 探针，不能经公开根域名访问。清单与 Go 注册端点由 `node --test infra/cloudflare-worker.test.mjs` 对照检查；新增私有端点时须同步路由、清单和测试。

## 候选准备

1. 记录主仓与 skills 固定提交、Pages deployment、`cellapp-edge` Worker version、Cloud Run revision、npm 版本及可回退版本。只使用隔离的候选服务、测试账号、测试 Bucket 和测试项目进行写操作。发布脚本从 `deploy/*` 分支执行，顺序和权限见 [发布规则](../.agents/rules/deploy.md)。
2. 用 [Cloud Run 边缘配置样例](../infra/cloudrun.edge.example.json)准备 `control_origin=https://cellapp.ai`，并使 GitHub OAuth callback URL 精确为 `https://cellapp.ai/auth/callback`。保留现有 `SECRET`、数据库和对象存储；Cookie 使用 `__Host-`、`Secure`、`HttpOnly`、`SameSite=Lax`、`Path=/`。浏览器迁移后需重新登录。
3. 根据 [Worker 样例](../infra/wrangler.edge.example.jsonc)在候选 Zone/Host 安装路径路由。设置 `CONTROL_HOST` 为候选根 Host、`ORIGIN_URL` 为候选 Cloud Run `run.app` HTTPS 地址，并通过 secret binding 设置专用 `EDGE_ORIGIN_SECRET`。Go 的 `CONTROL_ORIGIN` 和 OAuth callback 必须与候选 Host 一致。Worker 转发时重建 `X-Cellapp-Original-Host` 和 `X-Cellapp-Edge-Secret`，不能信任浏览器提供的同名头；响应使用 `private, no-store`。候选 Host 要验证 Pages custom domain 与 Worker 路由优先级。
4. 构建并预览官网正式 `dist/`、Dashboard Web 产物和服务镜像。确认官网页和 `/deploy/AGENTS.md` 仍由 Pages 返回，管理页及资产从 Cloud Run 返回。旧 `console.cellapp.ai` 不得再路由 `/apps`、`/device-authorizations`、`/api/console` 或其他带凭证 API；本变更不依赖旧浏览器书签重定向。

## 验收与切流

在候选 Host 逐项记录实际 URL、时间、版本和结果：官网导航与公开资产；Dashboard 直达和刷新；OAuth 登录与 callback、登录返回及设备码保留；Secure Cookie 的 Host 范围；匿名和跨来源写入拒绝；浏览器 API 与 Bearer API 互不代替；设备授权需用户在浏览器批准；旧项目 `cellapp.json` 精确旧来源迁移、同一 app ID 继续发布、新来源首次重新授权、第二个项目复用用户凭证、错误账号不能更新旧应用；访客应用登录、权限和资源访问。测试替身、受控候选、真实生产分别标注，不相互冒充。

候选通过后，按[发布规则](../.agents/rules/deploy.md)先发布兼容服务，再更新 Pages 和 Worker 的正式路径路由；确认旧控制域 API 路由已撤除。服务验收后才推送公开 Skill 并发布包含新默认来源的 npm 包。切流期间避免让新客户端指向尚未就绪的新路由。上线后从真实 `cellapp.ai` 与 `*.ce11.app` 重做关键读写和权限检查，并记录 Pages、Worker、Cloud Run、skills commit、npm 的实际版本。未完成真实验收时保持迁移状态为未完成。

## 失败恢复

保留数据库、旧发布对象和两边的版本记录。候选或切流验收失败时，停止后续发布并协调恢复先前的 Pages、Worker 路由和 Cloud Run revision；不要只切回服务，路径和 OAuth callback 也必须匹配。旧 API 有意退役，因此若已发出新来源客户端，恢复旧入口须提供与该客户端兼容的临时服务，或暂停部署并明确告知恢复路径。旧域凭证不能作为新域凭证使用。先确认受保护访客访问仍拒绝未授权请求，再解除任何部署暂停。Cloudflare 路由与 Pages custom domain 的行为以受控候选实测为准。

## 本轮验证状态

本清单记录程序和操作顺序，实际候选版本及每项远程验收结果在切流时另行填写。没有候选版本或真实 OAuth/Pages/Cloud Run 检查结果时，不得勾选 OpenSpec 的候选验收任务或归档变更。
