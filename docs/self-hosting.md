# 单机自部署 Cellapp

此路径在一台 Linux 主机上运行完整 Cellapp：Go 服务与配套 Web 控制台、内部 PostgREST、两个 PostgreSQL 数据库、默认私有 MinIO，以及 Caddy HTTPS 入口。`infra/compose.selfhost.yaml` 与本地开发的 `infra/compose.yaml` 分开。仅入口绑定主机的 80/443 端口；数据库、MinIO 与应用 HTTP 端口不映射到主机。

## 准备域名、证书和 OAuth

需要 Docker Engine 与 Compose v2、Python 3、OpenSSL、可用的 443 端口和可离机备份的磁盘。控制域和应用通配域必须处在**不同注册域**，例如 `control.example.com` 和 `apps.example.net`。将控制域的 A/AAAA 记录及 `*.apps.example.net` 的通配 DNS 指向服务器。准备受浏览器信任的控制域证书/私钥和 `*.apps.example.net` 通配证书/私钥；本版本不自动申请通配证书。私钥只让主机管理员读取。若使用反向代理或 CDN，必须保持原始 Host 与端到端 HTTPS。

在 GitHub **Settings → Developer settings → OAuth Apps** 创建应用。Homepage URL 填 `https://control.example.com`，Authorization callback URL 精确填 `https://control.example.com/auth/callback`。把 Client secret 单独写入仅管理员可读的文件，例如 `/secure/github-client-secret`，不要放进命令行参数。

## 初始化和启动

从递归检出的仓库根目录执行；把示例域名和证书路径替换为自己的值：

```sh
git submodule update --init --recursive
python3 scripts/selfhost.py setup \
  --control-host control.example.com \
  --apps-domain apps.example.net \
  --github-client-id YOUR_GITHUB_CLIENT_ID \
  --github-client-secret-file /secure/github-client-secret \
  --control-cert /secure/control.crt --control-key /secure/control.key \
  --apps-cert /secure/apps.crt --apps-key /secure/apps.key
docker compose -f infra/compose.selfhost.yaml build app
python3 scripts/selfhost.py start
```

`setup` 预检域名、证书覆盖范围与私钥配对、OAuth 输入，并将配置与随机独立秘密放在被 Git 忽略的 `infra/selfhost.local/`。目录权限为 0700，秘密文件为 0600（供非 root 应用读取的公开 CA 证书为 0644）；必须同数据库和对象一起备份。再次运行 `setup` 可省略参数并保留现有秘密；变更公网域名时需先改 GitHub OAuth 回调与 DNS/证书。`start` 依次等待数据库、初始化角色、建立私有 Bucket、运行控制库迁移、检查应用健康，最后启动入口。任一步失败会指出阶段；修复后再次运行 `python3 scripts/selfhost.py start`，不要删除卷。避免运行会输出秘密的 `docker compose config` 或公开 `runtime.env`。

## 验证

```sh
docker compose -f infra/compose.selfhost.yaml ps
curl -fsS https://control.example.com/health
curl -I https://control.example.com/dashboard/
```

`ps` 应显示 app 健康且入口运行；`/health` 返回成功。用浏览器检查真实可信证书、GitHub 登录、控制台、授权发布。全局安装 `npm install -g @cellapp/cli@latest`，以 `cellapp --help` 验证；登录和发布参数见[客户端文档](../skills/README.md)。发布静态测试项目后，检查 `https://<app-id>.apps.example.net`，同时检查未知 Host 不能拿到应用内容。MinIO API/控制台不应能从公网直连，Bucket 的匿名读取应被拒绝。这些公网 DNS、真实 GitHub 和公网证书检查需要运营者在自己的服务器完成；本地证书 fixture 不能替代。

## 更新与恢复

先备份 `infra/selfhost.local/`、PostgreSQL 两个数据库和 MinIO 卷，再拉取经审核的代码。构建新镜像并重新运行 `python3 scripts/selfhost.py start`；迁移失败时不启动新的公网入口。升级数据库或 MinIO 镜像前先确认版本兼容。回滚应用镜像也要先确认数据库迁移向后兼容，不能只恢复旧镜像覆盖新版结构。

建议在停写窗口做一致性卷备份，并将备份加密转移到另一台主机。以下命令从仓库根目录执行，保存 PostgreSQL 的**两个**数据库及角色、MinIO 全部对象和受保护配置；备份目录需有足够空间：

```sh
mkdir -m 700 -p backup/selfhost
docker compose -f infra/compose.selfhost.yaml stop ingress app postgres minio
tar -czf backup/selfhost/config.tgz infra/selfhost.local
docker run --rm -v cellapp-selfhost_postgres:/data:ro -v "$PWD/backup/selfhost:/backup" alpine:3.22 tar -C /data -cf /backup/postgres.tar .
docker run --rm -v cellapp-selfhost_minio:/data:ro -v "$PWD/backup/selfhost:/backup" alpine:3.22 tar -C /data -cf /backup/minio.tar .
python3 scripts/selfhost.py start
```

在**隔离的新主机**上恢复时，先复制相同源代码版本与全部备份，从仓库根目录确认目标卷为空，再执行：

```sh
tar -xzf backup/selfhost/config.tgz
docker volume create cellapp-selfhost_postgres
docker volume create cellapp-selfhost_minio
docker run --rm -v cellapp-selfhost_postgres:/data -v "$PWD/backup/selfhost:/backup:ro" alpine:3.22 tar -C /data -xf /backup/postgres.tar
docker run --rm -v cellapp-selfhost_minio:/data -v "$PWD/backup/selfhost:/backup:ro" alpine:3.22 tar -C /data -xf /backup/minio.tar
docker compose -f infra/compose.selfhost.yaml build app
python3 scripts/selfhost.py start
```

不要把备份解入正在运行的容器，也不要将旧卷与新卷混合。恢复演练必须核对控制用户、应用表和静态对象。回滚镜像前确认迁移兼容；不兼容时恢复同一时间点的全部备份。详细数据约束见[应用数据运维](app-data-operations.md)与 [S3 文档](s3-storage.md)。

若启动失败，查看 `docker compose -f infra/compose.selfhost.yaml ps` 及相应服务日志；修复证书、DNS、数据库或 MinIO 后重跑 `start`。不要执行 `docker compose down -v`，它会删除持久数据。真实公网验收、备份恢复和可用性监控由部署者负责。

本轮隔离环境的实际结果与未执行的公网检查见[自部署验收记录](self-hosting-verification.md)。
