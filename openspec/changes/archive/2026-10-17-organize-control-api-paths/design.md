# Design

## Context

参见 [proposal.md](proposal.md) 和 [control-api-routing 规格](specs/control-api-routing/spec.md)。当前 Go 在 `apps/server/internal/hosting/server.go` 注册根路径的设备授权、凭证撤销、额度及应用管理；`skills/packages/cli` 直接调用这些路径，`skills/contracts/http-v2.json` 记录其公开契约。根域名 Worker 的 `controlRoute`、Wrangler 路由及发布暂停识别也按旧 `/apps` 等路径工作。`unify-root-domain-dashboard` 正在调整同一根域路由清单，因此本变更实施前要以当时合入的 `main` 为基线复核路由。

## Goals / Non-Goals

**Goals:**

- 新路径和旧路径同时可用，调用同一业务处理，不形成两套权限、状态或幂等实现。
- 服务端和边缘路由先兼容，随后公开新版 CLI；保留旧 CLI 的登录、发布、数据管理与退出能力。
- 使 URL 清楚区分认证、Cookie 控制台和 Bearer 管理，并保持应用域隔离。

**Non-Goals:**

- 不改变设备授权协议、凭证内容、应用/部署/表管理的字段、状态码或配额。
- 不实现 MCP、OAuth 新模式、OpenAPI 全量化或新的 API 错误格式。
- 不在本变更移除旧路径，也不更改 `/api/console/*`、应用域 `/_cellapp/*`、`/rest/v1/*`。

## Decisions

### 1. 用双路由注册共享现有 handler

在 Go 控制域将每个旧路径和对应新路径注册到同一处理函数，避免 301/302/307/308 重定向改变请求体、密钥或 Authorization 的传输方式。新旧映射如下：

| 旧路径 | 新路径 |
| --- | --- |
| `POST /device-authorizations` | `POST /auth/device` |
| `POST /device-authorizations/poll` | `POST /auth/device/poll` |
| `DELETE /credentials/current` | `DELETE /auth/credentials/current` |
| `GET /limits` | `GET /api/control/v1/limits` |
| `/apps` 及全部现有应用、部署、数据子路径 | `/api/control/v1/apps` 及相同后缀 |

`/auth/device` 创建与轮询沿用无 Bearer 的设备秘密流程；`/auth/credentials/current` 与 `/api/control/v1/*` 沿用 Bearer 所有者检查。浏览器设备决策仍在 `/api/console/device/decision`，登录仍使用 `/auth/login`、`/auth/callback`。选择 `/api/control/v1` 而非 `/api/cli` 或 `/mcp`，因为资源能力不属于某个客户端，MCP 将来需要独立协议入口。替代方案是把所有新路径放进 `/api/control/v1`；这会再次把凭证签发混进资源管理，因此不采用。

### 2. 边缘路由与发布保护同步识别新路径

Worker 的控制域路径分类和 Wrangler 根域路由增加 `/api/control/v1` 树，保留旧路径与已有 `/auth*`。匹配须以精确路径或 `/` 边界为准，`/api/control/v1-extra` 不能落入控制路由；未知但位于 `/api/control/v1/*` 的请求应抵达 Go 并得到 404，不能由 Pages 返回 HTML。发布暂停检测同时覆盖旧 `/apps/{id}/deployments*` 与新 `/api/control/v1/apps/{id}/deployments*`，以免回滚期间新路径绕过暂停。边缘仍透传方法、查询、请求体和 Authorization，保持现有回源 Host 断言与私有缓存策略。

仅修改根域私有路由；应用域动态路径和静态交付不受影响。与只在 Go 注册新路径相比，同步 Worker 是必需的：根域公开请求否则可能落到 Pages。

### 3. 公开契约区分旧版与新规范路径

保留 `http-v1.json` 和 `http-v2.json` 作为历史契约，新建当前规范契约文件（如 `http-v3.json`）描述新路径；契约文件版本与 URL 中的 `/v1` 是两个不同维度，文档须说明。新契约逐条覆盖现有 17 个 CLI 操作，明确认证例外、成功形状和错误，不将浏览器 Cookie API 混入 Bearer 契约。CLI 集中定义路径前缀，登录、退出、部署、应用列表、密钥和数据命令均切换新路径；自定义控制 origin 也按相同路径调用。Skill 和公开部署说明优先引导使用 CLI，避免复制协议实现。

替代方案是直接改写 `http-v2.json`，但这会让已有文件的历史含义变化，难以核验旧客户端兼容，因此不采用。

### 4. 兼容验收与安全边界

为每组路径验证成功、错误、权限及同一请求标识跨路径重试；重点检查只带 Cookie 不能调用 Bearer API、只带 Bearer 不能调用浏览器控制台、应用域不能调用控制 API。旧契约测试目前只固定 v1，需补当前规范契约与路由的对应检查；边缘测试核对 Go 注册路径、Wrangler 路由和发布暂停匹配。生产候选环境至少执行一次新客户端设备授权、应用管理、部署和退出，以及旧客户端关键流程。测试替身与真实控制域验收分别记录。

## Risks / Trade-offs

- [新增 Go 路径未被 Worker 代理] → 路由清单测试比对 Go、Worker 与 Wrangler；候选域验证真实请求。
- [发布暂停只识别旧路径] → 双路径负向测试确保暂停状态下无法创建、上传或发布。
- [新 CLI 先于兼容服务发布] → 先发布并验证兼容服务及边缘路由，再推送/发布 CLI；发布规则要求 skills 提交先可获取，再推送主仓引用。
- [双路径长期存在导致维护负担] → 新文档只推荐规范路径，旧路径标注兼容；移除时间和废弃机制留待单独变更。
- [与根域名切换变更冲突] → 实施前复核已合入的根域 Worker、清单和状态；在任务分支解决差异并重新验收。

## Migration Plan

1. 从当时的 `main` 核对根域名切换状态，新增 Go 双路由、边缘分类与发布暂停规则，完成服务及 Worker 测试；不改变旧入口。
2. 在隔离候选环境验证新旧路径、设备批准、Cookie/Bearer 分离、发布暂停和应用域隔离；确认真实根域 Worker 会转发新路径。
3. 更新公开契约、CLI 与文档并完成子仓库独立检查。按发布规则先让兼容服务和 Worker 生效，再发布新版 CLI/Skill。Git 推送不等于这些发布步骤。
4. 若上线验收失败，停止新版客户端发布或回退其版本，保留旧路径与旧版 CLI 作为恢复入口；不要在本变更清除旧路由或数据库状态。
