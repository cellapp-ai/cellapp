# 单机自部署本轮验收

## 单域名补充验收（2026-10-24）

- `example.com` 搭配 `*.example.com`、`example.com` 搭配 `*.apps.example.com`，以及不同注册域配置均通过 Go 生产配置检查和隔离的自部署初始化测试。`control.example.com` 搭配 `*.example.com` 被拒绝。
- 初始化测试使用临时自签名证书，检查证书覆盖、生成的 Caddy Host 规则、重复初始化时证书及秘密保持不变；通过。它不代表真实公网证书和 DNS 验收。
- 在隔离 PostgreSQL 容器上，`go test -race -count=1 ./apps/server/...` 通过；同域 Host 隔离、控制台 Cookie、应用来源或缺少 Origin 的写入拒绝及 CLI Bearer 行为由集成测试覆盖。`go vet ./apps/server/...`、Python 测试、两项 OpenSpec 严格校验和 `git diff --check` 通过。
- 本次未运行完整浏览器发布流程，也未在公网域名检查 GitHub OAuth、真实证书、DNS 和实际应用页面。这些仍需在部署主机验证。

以下为 2026-10-22 的原始历史记录，当时使用 MinIO 和双域名测试配置；不代表本轮 RustFS 或单域名验收结果。

日期：2026-10-22。执行环境：macOS arm64、Docker Desktop、隔离的 Compose 项目和临时自签名控制域/应用通配域证书。所有凭证与卷均为本轮测试资源，未连接生产 GitHub OAuth、公共 DNS 或用户数据。

## 通过

- `python3 scripts/selfhost.py setup` 完成域名、证书/私钥、OAuth 输入预检。重复执行保留相同 `secrets.json` 字节，秘密文件权限为 0600；重叠域名被拒绝。
- `docker compose -f infra/compose.selfhost.yaml config --services` 列出 PostgreSQL、数据库初始化、MinIO、Bucket 初始化、迁移、应用和入口；解析后的主机端口仅属于入口。
- 同源镜像构建出配套 Web、Go、PostgREST、内部 Caddy；控制数据库/应用数据数据库与三个角色可用，跨库角色连接被拒绝。
- 首次启动与重复启动健康；MinIO 以内部 HTTPS 提供私有 Bucket，签名对象上传/下载成功、匿名策略为 private，重启后对象仍在。
- 临时 TLS 证书下，控制域 `/health` 为 200、`/dashboard/` 为 200、`/auth/login` 为 302、HTTP 跳转 HTTPS、应用通配域到达网关，未知 Host 在 TLS 握手时被拒绝。
- MinIO 未以 TLS 启动时 Bucket 阶段失败且卷保留；故意使控制迁移连接失效时准确报告阶段，恢复配置后再次启动成功。
- 停写后备份配置、PostgreSQL 与 MinIO 卷，在另一个 Compose 项目的新卷中恢复；控制数据库、应用数据数据库和对象标记均可读取，恢复应用健康。
- `RUN_BROWSER_TESTS=1` 的 `TestBrowserEndToEnd` 使用独立 Compose PostgreSQL、PostgREST 和 MinIO，以及测试身份提供方；浏览器登录、授权、发布、公开访问、密钥轮换与撤销、CRUD、隔离和删除通过。
- `go test ./apps/server/...`、`go vet` 相关入口、OpenSpec 严格校验、Caddy 配置校验、文档相对链接、Python 语法和 `git diff --check` 通过。

## 尚需运营环境验证

- 本机拉取 `minio/minio:RELEASE.2025-04-22T22-12-26Z` 与对应 `minio/mc` 固定镜像时 Docker 仓库连续返回 EOF。Compose 启动与恢复验收使用本机缓存的 MinIO `RELEASE.2026-09-22T19-25-18Z` 和 mc `RELEASE.2025-08-13T08-35-41Z` 临时映射；仓库中的 Compose 仍固定原镜像版本。应在部署主机拉取并重跑固定版本。
- 测试身份提供方只存在于浏览器测试中。真实 GitHub OAuth 回调、公共 DNS、浏览器受信任证书及公网应用访问需在运营域名逐项验证。
- 单机故障切换、远程备份介质和 Cloud Run 发布不属于本轮 Compose 验收。
