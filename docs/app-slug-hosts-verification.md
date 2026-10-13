# 应用与用户 slug 域名验收

本轮在一次性 PostgreSQL 17.6 容器的随机本地端口运行隔离测试 schema；浏览器检查使用本地 Chrome、只对临时测试证书的公钥启用信任，并保持浏览器自身的 HTTPS 验证。未访问生产数据库、对象存储或域名。

| 检查 | 结果 |
| --- | --- |
| `npm test --workspace @cellapp/web` | 通过；8 组浏览器 UI 检查，包括两种 slug 编辑、反馈、地址刷新和布局 |
| `TEST_DATABASE_URL=<isolated-dsn> go test -race -count=1 ./apps/server/...` | 通过；迁移、保留词、跨账户完整主机名冲突、旧链接、登录回跳、同源写入和原有服务端回归 |
| `RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=<isolated-dsn> go test -race -count=1 -run '^TestSlugHostBrowser$' -v ./apps/server/internal/hosting` | 通过；真实 HTTPS Chrome 验证控制台编辑、组合域名与旧 ID 域名登录回跳、Host-only Cookie 隔离 |
| `go vet ./apps/server/...`、前端 typecheck/lint、OpenSpec strict | 通过 |
| skills typecheck、13 项测试、build、Skill/boundary/package 检查与 pack dry-run | 通过；package 检查使用独立临时 npm 缓存 |

完整服务端 race 检查首次运行时，原有设备授权轮询测试一次出现 `authorization_pending`（预期 `429`）；该测试单独复测与随后完整重跑均通过。完整数据与对象存储浏览器验收未运行，本轮新增的浏览器测试仅覆盖 slug 设置、登录回跳与 Cookie 域隔离。生产切换还需配置 `APPS_DOMAIN=ce11.app`、`*.ce11.app` 的 DNS 和 TLS。
