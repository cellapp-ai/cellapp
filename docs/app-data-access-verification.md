# BYO Supabase 数据访问验收

本轮实现 OpenSpec 变更 `add-byo-supabase-data`：所有者自带 Supabase 项目，同一应用的分享密钥访客共用一份数据。Cellapp 不运行用户后端、不代理查询。

## 通过

- `npm run typecheck`（含 skills CLI 与 `apps/web`）
- `npm --prefix skills test`：12 项通过，含公开配置校验与 HTTP 契约 data 操作
- `npm --prefix skills run check:skill`、`check:boundary`
- `go vet ./apps/server/...`
- `TEST_DATABASE_URL=postgres://ohmyapp:development@127.0.0.1:5432/ohmyapp go test -race -count=1 ./apps/server/...`：含迁移升级、所有者绑定/拒绝 service role、访客会话下 `/_hosting/data`、控制台 Origin 检查
- `npm run lint:web`
- `node scripts/check-public-boundary.mjs`：改为只扫描服务端公开面（`apps/server/` 与既有允许文件），不再把 Web/OpenSpec/文档当作越界

## 跳过 / 未执行

- `RUN_BROWSER_TESTS=1 ... TestBrowserEndToEnd`：本环境无完整 HTTPS 代理与受信任 CA 编排，未跑跨服务真实解锁后直连 `/_hosting/data` 的端到端。
- 未连接真实 Supabase 项目做 Rest/RLS 验收；平台侧只保存并下发公开配置。

## 缺口

将 `typecheck-browser` 补进根 `package.json`，以匹配现有 CI 脚本。未把 `contracts/client-support.json` 升到 1.1.0：该文件注明须在公开契约发布且服务端一致性测试通过后再更新。
