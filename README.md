# OhMyApp

免费静态应用托管：在本机通过 Skill / CLI 构建发布，使用固定 HTTPS 地址和应用独立分享密钥访问。服务端为 Go，CLI 为 TypeScript；PostgreSQL 保存控制元数据，七牛云私有 S3 兼容存储保存发布产物。

## 开发

需要 Go 1.24+、Node.js 22.12+ 和 Docker Compose。

```sh
npm ci
go mod download
docker compose -f infra/compose.yaml up -d
cp .env.example .env
```

先按 [七牛存储配置](docs/qiniu-storage.md) 填写 S3 区域、S3 空间名及密钥。本地 Compose 只运行数据库与 HTTPS 代理，不创建云端 Bucket。

本地开发使用 `AUTH_MODE=dev`，访问登录入口时直接进入唯一的固定开发账号，无需连接 GitHub。该模式只允许非生产环境和 localhost/loopback 控制域；生产环境或非本地域名配置 dev 模式会拒绝启动。它仍签发正常浏览器会话，因此设备授权和权限检查走与生产相同的后续流程。

生产设置 `AUTH_MODE=github`，并在 GitHub 的 **Settings → Developer settings → OAuth Apps** 创建 OAuth App。填写真实控制域作为 Homepage URL，Authorization callback URL 填写 `${CONTROL_ORIGIN}/auth/callback`，然后将 Client ID 和新生成的 Client secret 写入 `GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`。服务端使用 OAuth state 与 PKCE，并通过 GitHub API 返回的不可变数字用户 ID 绑定所有者；不会读取邮箱或仓库权限。

加载 `.env` 后执行（仅加载你自己维护的文件）：

```sh
set -a
. ./.env
set +a
npm run migrate
npm run dev
```

Docker 内代理连接宿主服务时，将开发 `LISTEN_ADDRESS` 设置为 `0.0.0.0:3000`，并用本机防火墙限制直连；生产监听仅允许可信反向代理访问。应用入口仍按精确 Host 检查。当前限流按直接连接地址计数，代理后的请求会共享来源配额；不要启用未验证的转发头信任。

本地入口为 `https://control.localhost:8443` 和 `https://<app-id>.apps.localhost:8443`。Caddy 使用本地 CA，浏览器和 CLI 都必须信任该 CA；不要关闭 TLS 验证。导出 CA：

```sh
mkdir -p .local
docker compose -f infra/compose.yaml cp proxy:/data/caddy/pki/authorities/local/root.crt .local/root.crt
export NODE_EXTRA_CA_CERTS="$PWD/.local/root.crt"
```

根据操作系统将此证书加入本地测试浏览器的受信任证书，或使用隔离测试浏览器配置。若系统不解析 `*.localhost`，为本次应用 host 配置本地 DNS。

## 发布静态页面

```sh
npm run cli -- login --origin https://control.localhost:8443
npm run cli -- deploy --project /absolute/project --output dist --build 'npm run build' --spa --origin https://control.localhost:8443
npm run cli -- apps --origin https://control.localhost:8443
```

普通 HTML 目录可省略 `--build`。建议产物位于项目的专用子目录，避免把源码或配置文件当成网页。`ohmyapp.json` 保存应用关联和待完成部署标识，不保存秘密。部署凭证位于 `~/.config/ohmyapp/` 的私有文件。首次密钥只向所有者展示，丢失后可主动使用 `reset-key <app-id>`；重置会让旧会话失效。

更新时在同一项目运行 deploy。网络错误可以重试同一命令；如果发生版本冲突，先确认最新应用版本，再移除 `ohmyapp.json` 的 `pending` 字段以创建新部署。不要删除 `appId`，否则会创建另一个应用。

## CLI 与 Skill 安装

构建后，CLI 入口是 `dist/packages/cli/src/main.js`，需要一起保留 `dist/packages/shared`。在开发机可使用 `npm run cli -- ...`。`npm pack --ignore-scripts` 生成包含 CLI/shared 与 Skill 的安装包；用户可主动运行 `npm install -g /path/to/ohmyapp-0.1.0.tgz`，之后使用 `ohmyapp`。当前尚未发布到 npm 注册表。

将仓库内 `skills/deploy-static-app/` 复制到目标 Agent 的技能目录（例如 `~/.codex/skills/deploy-static-app`），并确保 `ohmyapp --help` 可用。这是用户主动安装步骤，本仓库不会自动修改全局 Agent 配置。

## 验证

```sh
npm run typecheck
npm test
npm run build
TEST_DATABASE_URL=postgres://ohmyapp:development@127.0.0.1:5432/ohmyapp go test -race -count=1 -v ./apps/server/internal/hosting
RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=postgres://ohmyapp:development@127.0.0.1:5432/ohmyapp go test -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting
```

未提供 `TEST_DATABASE_URL` 时，真实数据库集成测试会明确跳过。集成测试创建随机独立 schema，结束后清理，不操作其他 schema。浏览器测试默认使用本机 macOS Chrome，可用 `CHROME_PATH` 指定其他平台 Chrome 可执行文件。测试身份提供方和存储替身仅编译在测试中；生产服务没有认证绕过入口。内存对象存储测试替身不能代替真实七牛 S3 与匿名读取验收。

范围：在线 HTML/JS/CSS 与静态资源；不提供后端运行、数据库同步、公开免密、自定义域名或离线 PWA。更多运维配置见 [operations.md](docs/operations.md)。
