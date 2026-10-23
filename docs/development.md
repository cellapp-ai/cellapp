# 开发 Cellapp

本指南用于本地开发。完整平台的单机生产部署见[自部署指南](self-hosting.md)，Cloud Run 路径见[运维文档](operations.md)。

## 检出与依赖

需要 Go 1.24+、Node.js 22.12+、Docker Compose。官网另需 Corepack/pnpm。首次检出使用：

```sh
git clone --recurse-submodules <主仓库地址>
cd cellapp
npm ci
npm --prefix skills ci
go mod download
```

已有检出先运行 `git submodule update --init --recursive`。更新时先检查主仓库与子模块未提交修改，再用 `git pull --recurse-submodules`；不要使用 `update --remote` 隐式更换固定客户端版本。子模块修改前若处于 detached HEAD，先创建开发分支。协作和提交规则见 [AGENTS.md](../AGENTS.md) 与[客户端规范](../skills/AGENTS.md)。

## 本地服务

```sh
docker compose -f infra/compose.yaml up -d postgres postgrest proxy rustfs
docker compose -f infra/compose.yaml run --rm rustfs-init
cp .env.example .env
set -a
. ./.env
set +a
npm run migrate
npm run build --workspace @cellapp/web
npm run dev
```

开发模式 `AUTH_MODE=dev` 只允许非生产 localhost 控制域。开发入口是 `https://control.localhost:8443` 和 `https://<app-id>.apps.localhost:8443`；Docker 代理连接宿主 Go 服务时，`LISTEN_ADDRESS` 使用 `.env.example` 中的开发值。本地 Caddy CA 必须加入测试浏览器信任；CLI 可用以下方式获得该证书：

```sh
mkdir -p .local
docker compose -f infra/compose.yaml cp proxy:/data/caddy/pki/authorities/local/root.crt .local/root.crt
export NODE_EXTRA_CA_CERTS="$PWD/.local/root.crt"
```

RustFS 初始化幂等，私有 Bucket 持久化。旧 MinIO 卷不会被新环境复用或自动迁移。配置、失败重试及外部存储见 [S3 文档](s3-storage.md)；应用数据角色与现有卷升级见[应用数据运维](app-data-operations.md)。

## 构建与检查

```sh
npm run cli -- --help
npm run dev:web
npm run typecheck
npm test
npm run build
go vet ./apps/server/...
corepack pnpm --dir apps/website install --frozen-lockfile
corepack pnpm --dir apps/website run lint
corepack pnpm --dir apps/website run build
```

前端工程、代码与设计规则见[前端规范入口](standards/README.md)。官网独立于根 npm workspace，使用自己的 [协作规范](../apps/website/AGENTS.md)。服务端真实数据库和浏览器验收需要独立测试资源；环境变量与命令见根 [AGENTS.md](../AGENTS.md)。接口见[控制 API](control-api.md) 与 [Web API](web-api.md)。

开发构建和 Git 推送不会自动发布 Cloud Run、npm 或官网。按[发布规则](../.agents/rules/deploy.md)分别验收。
