# Web 控制面本轮验收

日期：2026-09-29。变更：`add-web-control-plane`。本轮实施 `apps/web` 管理界面、`/api/console` 浏览器 API、Go 静态交付、根工作区编排和前端规范；未修改公开客户端实现，不将旧前端基础或品牌变更的验收记录当成本轮证据。

## 已通过

| 检查 | 本轮结果与边界 |
| --- | --- |
| 根 `npm ci --ignore-scripts` | 依赖安装通过；锁文件使用 `apps/web` / `@cellapp/web`，无旧 workspace 条目 |
| 无客户端子模块的独立 Web 安装/构建 | 临时目录仅复制根依赖声明/锁文件和 `apps/web`，离线安装及构建通过，没有 `skills/` |
| `npm run typecheck`、`npm run lint:web` | 主仓库/客户端/Web 类型与 Web lint 通过 |
| 带隔离数据库的 `npm test` | 客户端 11 项、根编排 2 项、Go 数据库测试及 Web 浏览器 6 组检查通过；跨服务 HTTPS 浏览器由独立开关控制，未被本命令记为通过 |
| `npm run build`、`go vet ./apps/server/...` | 配套 `dist/cellapp-server` 与完整 `dist/web/` 构建、Go 静态检查通过 |
| `TEST_DATABASE_URL=<isolated-test-dsn> go test -race -count=1 ./apps/server/...` | 独立 PostgreSQL 17.6，通过权限、来源、会话/凭证失效、设备过期及并发决策、密钥/访客失效、删除/重置并发和原有部署/网关回归；浏览器开关未开启 |
| 配套产物启动检查 | 仅在隔离数据库执行迁移并监听本地临时端口，验证页面/CSP、API 401、未知 API/资源 404 和 CLI `/apps` 原认证；这是后端 HTTP 冒烟检查，不是 HTTPS 浏览器管理验收 |
| 缺少产物与维护命令 | 缺失 `WEB_ROOT` 的正常服务返回非零并说明先构建 Web；同样缺失目录下 `-metrics` 正常执行 |
| 受信任 HTTPS 跨服务浏览器 | `RUN_BROWSER_TESTS=1`、独立 PostgreSQL 17.6、已手动信任的本地测试 CA；`go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting` 通过，浏览器用例 15.16 秒、包检查 16.998 秒；浏览器与 Node 验证 TLS，保留 Secure Cookie |
| OpenSpec 严格校验与工作区检查 | `openspec validate add-web-control-plane --strict`、差异空白检查及当前路径/前缀检查通过 |

Web 浏览器检查使用显式测试夹具，生产界面无模拟数据。覆盖正式产物与本地字体、320/390/768/800/801/1440 px、axe、键盘导航/菜单 Escape/跳过正文、确认取消及焦点恢复、未发布详情深链、临时新密钥、剪贴板失败、写响应丢失且无自动重试、凭证撤销、设备主动拒绝、空/失败/重试、长内容/200% 文字缩放、减少动态效果、会话过期和无脚本说明。报告/截图位于忽略的 `test-results/web/`；桌面及小屏截图已检查，修正小屏导航行高分配。

## HTTPS 验收与环境边界

受信任 HTTPS 验收使用实际 `apps/web/dist` 产物与 Go 控制 API，未关闭 TLS 校验或 Secure Cookie。覆盖真实 Web 登录与设备批准、应用列表/详情深链刷新、Web 密钥重置及旧访客会话失效、新密钥解锁、Web 删除、部署凭证撤销；同时验证原有 CLI 设备创建/轮询、部署发布、丢失发布响应恢复、CLI 密钥重置/退出、两个访客会话、应用间本地数据隔离、Service Worker/CORS 拒绝和资源逐次鉴权。身份提供方与对象存储分别使用测试身份和内存存储替身。

此前自动添加 CA 信任的尝试在 macOS 返回 `SecTrustSettingsSetTrustSettings: The authorization was denied since no user interaction was possible.`，当次尚未进入浏览器管理流程。临时钥匙串搜索列表已恢复、临时钥匙串已清理。用户随后手动信任独立测试 CA，本轮复验通过，任务 4.3 已完成；保留此前失败事实，不将失败尝试记为成功。

独立测试 CA 位于忽略目录 `.local/web-browser-test-ca/ca.crt`，私钥为同目录 `ca.key`，权限 0600，仅供本地测试，证书有效期两天。测试通过 `TEST_BROWSER_CA_CERT` 与 `TEST_BROWSER_CA_KEY` 签发短期 localhost 证书，浏览器及 Node 都验证 TLS。后续在 CA 仍有效且已信任时，从根目录复验：

```sh
npm run build --workspace @cellapp/web
RUN_BROWSER_TESTS=1 \
TEST_DATABASE_URL=<isolated-test-dsn> \
TEST_BROWSER_CA_CERT="$PWD/.local/web-browser-test-ca/ca.crt" \
TEST_BROWSER_CA_KEY="$PWD/.local/web-browser-test-ca/ca.key" \
go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting
```

真实外部 GitHub、七牛对象存储/匿名读取、多浏览器兼容性与生产部署本轮未执行；身份/内存存储替身不能代替这些结果。

## 工作范围及交付状态

本轮文件范围：`apps/web`（由已有前端迁移）、服务端应用业务复用/会话/路由/静态交付/启动配置及相关测试、根依赖声明/锁文件/编排测试、README/AGENTS/环境示例、前端规范、运维说明、浏览器契约和本验收记录、当前 OpenSpec 任务状态。

开始时主仓库已有暂存的仓库迁移、前端基础及品牌工作，均保持；子仓库无本轮源码修改。前置 `add-console-frontend-foundation` 当时 15/16，`unify-cellapp-naming` 当时 7/7，未改写其验收结论或代为归档。本轮归档已将前端基础的三个要求及全部场景合并至 `web-control-plane`，按确认删除 `console-frontend` 主规格，保留 `apps/web` 实现。历史变更保持原记录，后续不得用旧基础增量重建退役能力或覆盖控制面要求。

本变更 24/24 项任务完成，必要验收已通过。2026-09-29 已同步 `web-control-plane` 和 `repository-workflow` 主规格、显式退役并删除 `console-frontend` 主规格，归档到 `openspec/changes/archive/2026-09-29-add-web-control-plane/`；全部六份主规格严格校验通过。

归档后已审查自动提交条件，但尚未 commit、未 push：客户端仍为已有 `62a6d7e`，在空临时 Git 仓库从远程获取该完整提交时返回 `upload-pack: not our ref`，远程仅公布 `2847115`。按项目跨仓库规则，不能提交指向不可获取客户端版本的主仓库交付版本；未擅自推送客户端或改写 gitlink。主仓库保留已有暂存内容和本轮工作区改动，本轮无客户端源码修改或新提交。临时 PostgreSQL 容器已停止并自动移除，后续复验需重新准备独立测试数据库。


## 本地主仓库提交

用户在知晓客户端远程可获取性缺口后，明确要求提交当前仓库。本次按该授权保存主仓库的本地工作快照，包含 Web 控制面及其已有工程迁移、品牌、规范和规格工作；保留客户端 gitlink `62a6d7e`，不擅自改用其他版本。本地提交不代表已完成跨仓库远程交付，后续发布前仍须推送客户端并验证该提交可获取。本次不推送主仓库、不发布 npm 包。

本地开发规则、环境配置、证书及私钥、运行日志、截图、构建产物和依赖目录均保持本地排除，不纳入提交。提交检查通过差异空白与 OpenSpec 全量严格校验；本次没有改动已验收的业务实现，因此沿用本轮实际执行的相关验证结果，不宣称重新执行了全部测试。
