# Cellapp

免费静态应用托管：在本机通过 Skill / CLI 构建发布，使用固定 HTTPS 地址和应用独立分享密钥访问。服务端为 Go，CLI 为 TypeScript；PostgreSQL 保存控制元数据，私有 S3 兼容存储保存发布产物，本地开发默认使用 MinIO。

## 开发

需要 Go 1.24+、Node.js 22.12+ 和 Docker Compose。

```sh
git clone --recurse-submodules <主仓库地址>
cd cellapp
npm ci
npm --prefix skills ci
go mod download
docker compose -p cellapp-local-minio -f infra/compose.yaml up -d postgres proxy minio
docker compose -p cellapp-local-minio -f infra/compose.yaml run --rm minio-init
cp .env.example .env
```

本地 Compose 提供数据库、HTTPS 代理及持久化 MinIO。初始化命令必须成功后再启动服务，它会幂等创建私有 `cellapp` Bucket，重复运行保留已有对象。开发 S3 默认与 `.env.example` 一致，无需云端账号。默认使用 Supabase S3 Compose 所用的 Chainguard MinIO 镜像；详细配置、`dhi.io/minio:latest` / `dhi.m.daocloud.io/minio:latest` 备选镜像、初始化失败重试及已有外部存储升级见 [S3 存储配置](docs/s3-storage.md)。

本地开发使用 `AUTH_MODE=dev`，访问登录入口时直接进入唯一的固定开发账号，无需连接 GitHub。该模式只允许非生产环境和 localhost/loopback 控制域；生产环境或非本地域名配置 dev 模式会拒绝启动。它仍签发正常浏览器会话，因此设备授权和权限检查走与生产相同的后续流程。

生产设置 `AUTH_MODE=github`，并在 GitHub 的 **Settings → Developer settings → OAuth Apps** 创建 OAuth App。填写真实控制域作为 Homepage URL，Authorization callback URL 填写 `${CONTROL_ORIGIN}/auth/callback`，然后将 Client ID 和新生成的 Client secret 写入 `GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`。服务端使用 OAuth state 与 PKCE，并通过 GitHub API 返回的不可变数字用户 ID 绑定所有者；不会读取邮箱或仓库权限。

加载 `.env` 后执行（仅加载你自己维护的文件）：

```sh
set -a
. ./.env
set +a
npm run migrate
npm run build --workspace @cellapp/web
npm run dev
```

Docker 内代理连接宿主服务时，将开发 `LISTEN_ADDRESS` 设置为 `0.0.0.0:3000`，并用本机防火墙限制直连；生产监听仅允许可信反向代理访问。应用入口仍按精确 Host 检查。当前限流按直接连接地址计数，代理后的请求会共享来源配额；不要启用未验证的转发头信任。

本地入口为 `https://control.localhost:8443` 和 `https://<app-id>.apps.localhost:8443`。Caddy 使用本地 CA，浏览器和 CLI 都必须信任该 CA；不要关闭 TLS 验证。导出 CA：

```sh
mkdir -p .local
docker compose -p cellapp-local-minio -f infra/compose.yaml cp proxy:/data/caddy/pki/authorities/local/root.crt .local/root.crt
export NODE_EXTRA_CA_CERTS="$PWD/.local/root.crt"
```

已有数据库卷的配置升级见 [命名升级说明](docs/operations.md#命名升级)。

根据操作系统将此证书加入本地测试浏览器的受信任证书，或使用隔离测试浏览器配置。若系统不解析 `*.localhost`，为本次应用 host 配置本地 DNS。

## Web 控制面

`apps/web` 是主仓库的私有 npm workspace，根 `npm ci` 安装依赖。它提供真实应用列表/详情、分享密钥重置、删除、部署凭证撤销与设备授权。Go 在控制域同源提供页面及 `/api/console`，登录沿用现有浏览器会话。独立开发和界面检查不需要客户端子模块；未连接 Go 时显示服务不可达状态：

```sh
npm run dev:web
npm run typecheck --workspace @cellapp/web
npm run lint:web
npm run build --workspace @cellapp/web
npm run preview:web
npm test --workspace @cellapp/web
```

开发默认 `http://127.0.0.1:5173`，正式产物预览默认 `http://127.0.0.1:4173`，仅监听 loopback。前端产物为 `apps/web/dist/`，包含完整 HTML、JS、CSS、字体与品牌资源。前端测试自动构建并启动临时预览，使用本机 Chrome；非 macOS 或其他安装路径通过 `CHROME_PATH` 指定浏览器，报告及截图保存于 `test-results/web/`。

真实联调先构建前端再启动 Go，从 `https://control.localhost:8443` 访问。`WEB_ROOT` 开发默认 `apps/web/dist`；缺少可读取的 index/assets 时正常服务启动失败，迁移/清理等维护命令不依赖前端产物。完整构建生成 `dist/cellapp-server` 与 `dist/web/`，部署和回滚必须配套，生产显式指定静态目录。浏览器契约见 [Web API](docs/web-api.md)。

完整仓库 `typecheck`、`test`、`build` 也包含 Web 检查，测试包含前端 lint 和浏览器验收；缺少客户端子模块时仍返回初始化提示。前端工程、代码与设计约定见 [前端规范入口](docs/standards/README.md)，与服务端及 CLI 规范分别维护。

## 发布静态页面

```sh
npm run cli -- login --origin https://control.localhost:8443
npm run cli -- deploy --project /absolute/project --output dist --build 'npm run build' --spa --origin https://control.localhost:8443
npm run cli -- apps --origin https://control.localhost:8443
```

普通 HTML 目录可省略 `--build`。建议产物位于项目的专用子目录，避免把源码或配置文件当成网页。`cellapp.json` 保存应用关联和待完成部署标识，不保存秘密。部署凭证位于 `~/.config/cellapp/` 的私有文件。首次密钥只向所有者展示，丢失后可主动使用 `reset-key <app-id>`；重置会让旧会话失效。

更新时在同一项目运行 deploy。网络错误可以重试同一命令；如果发生版本冲突，先确认最新应用版本，再移除 `cellapp.json` 的 `pending` 字段以创建新部署。不要删除 `appId`，否则会创建另一个应用。

## 仓库与客户端版本

`skills/` 是独立 Git 子模块，公开客户端 TypeScript 包位于 `skills/packages/`，Web 前端位于主仓库 `apps/web`，主仓库固定子模块提交。已有检出先初始化：

```sh
git submodule update --init --recursive
```

拉取主仓库及已初始化客户端：

```sh
git pull --recurse-submodules
```

可主动执行 `git config submodule.recurse true` 设置本仓库的默认递归行为；新增或未初始化子模块仍需初始化。更新前检查两个仓库的本地修改；不要使用 `update --remote` 自动追随最新客户端。子仓库远程使用 SSH，需要对应 GitHub 读取权限与 SSH 认证。

主仓库 `npm run cli`、`typecheck`、`test`、`build` 调用子仓库客户端；完整测试与构建还包含 Go 服务端。客户端修改前若处于 detached HEAD，先创建开发分支。交付时先提交、推送子仓库并确认提交可获取，再提交主仓库固定版本，详见 [AGENTS.md](AGENTS.md) 和 [客户端规范](skills/AGENTS.md)。

## CLI 与 Skill 安装

客户端独立安装、构建和打包：

```sh
npm --prefix skills ci
npm --prefix skills run build
cd skills
npm pack --ignore-scripts
```

CLI 入口是子仓库中的 `dist/packages/cli/src/main.js`，需要保留其 `dist/packages/shared`。可在干净目录安装生成的 `cellapp-cli-0.1.0.tgz`，命令为 `cellapp`，仍提供 `ohmyapp` 别名。根目录不再打包 CLI。

将 `skills/skills/cellapp-deploy/` 复制到目标 Agent 的技能目录，并确保 `cellapp --help` 可用。安装 Skill 不会安装 CLI，不自动修改全局 Agent 配置。已有 `ohmyapp.json` 和旧凭证在首次使用时安全迁移并保留恢复副本；`OHMYAPP_ORIGIN` 仍支持，新增配置使用 `CELLAPP_ORIGIN`。独立客户端安装与兼容说明见 [客户端 README](skills/README.md)。

## 验证

```sh
npm run typecheck
npm test
npm run build
go vet ./apps/server/...
TEST_DATABASE_URL=postgres://cellapp:development@127.0.0.1:5432/cellapp go test -race -count=1 -v ./apps/server/internal/hosting
RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=postgres://cellapp:development@127.0.0.1:5432/cellapp go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting
```

未提供 `TEST_DATABASE_URL` 时，真实数据库集成测试会明确跳过。集成测试创建随机独立 schema，结束后清理，不操作其他 schema。跨服务浏览器验收要求真实 MinIO 私有 Bucket，使用 `S3_*` 指定隔离存储并先完成 MinIO 初始化；默认使用本机 macOS Chrome，可用 `CHROME_PATH` 指定其他平台 Chrome 可执行文件。macOS 跨服务浏览器测试为本轮临时 CA 建立临时钥匙串信任，并在结束时恢复搜索列表、移除信任与临时钥匙串，不关闭 TLS 校验；已有受信任的隔离测试 CA 时，可通过 `TEST_BROWSER_CA_CERT` 和 `TEST_BROWSER_CA_KEY` 指定证书及私钥，测试只用它签发临时 localhost 证书，不修改系统信任。macOS 禁止自动信任时也使用此方式。测试身份提供方和存储替身仅编译在测试中；生产服务没有认证绕过入口。内存对象存储测试替身不能代替真实 MinIO / S3 与匿名读取验收。

范围：在线 HTML/JS/CSS 与静态资源；不提供后端运行、数据库同步、公开免密、自定义域名或离线 PWA。更多运维配置见 [operations.md](docs/operations.md)。本次目录与子模块迁移验收见 [仓库协作规范验收](docs/repository-workflow-verification.md)。
