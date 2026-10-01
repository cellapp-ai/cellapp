# S3 存储配置

服务使用私有 S3 兼容 Bucket 保存发布产物，开发默认使用本地 MinIO。访客仅通过应用网关读取文件，不获得对象读取签名直链。适配器使用 path-style 和 Signature V4，支持指定 Bucket 的 HEAD 及对象 PUT/GET/HEAD/DELETE；生产 Endpoint 必须为 HTTPS。

## 本地 MinIO

以下命令使用独立 Compose 项目名 `cellapp-local-minio`，避免与已有 Supabase 或其他项目的容器、卷混用。

```sh
docker compose -p cellapp-local-minio -f infra/compose.yaml up -d postgres proxy minio
docker compose -p cellapp-local-minio -f infra/compose.yaml run --rm minio-init
```

第二条命令必须以零状态退出，出现 `Private local S3 bucket initialized.` 后才算存储就绪。初始化有界等待服务、幂等创建 Bucket 并关闭匿名访问；重复运行保留对象。首次运行需要能够访问镜像仓库。服务镜像默认采用 [Supabase S3 Compose](https://github.com/supabase/supabase/blob/master/docker/docker-compose.s3.yml) 所用的 `cgr.dev/chainguard/minio:latest`，初始化客户端为 `cgr.dev/chainguard/minio-client:latest-dev`。本轮在独立 Compose 项目完成私有 Bucket、读写和重启持久化验证。也可尝试 Docker Hardened Images 的 `dhi.io/minio:latest` 或其 DaoCloud 国内镜像 `dhi.m.daocloud.io/minio:latest`；这是另一发行渠道，应先确认与当前 Compose 命令、卷权限兼容。原站可能需要 Docker 登录，DaoCloud 镜像为源镜像的映射，`latest` 可能短暂滞后。使用镜像覆盖时，两条 Compose 命令保持同一设置：

```sh
export CELLAPP_MINIO_IMAGE=dhi.m.daocloud.io/minio:latest
docker pull "$CELLAPP_MINIO_IMAGE"
docker compose -p cellapp-local-minio -f infra/compose.yaml up -d postgres proxy minio
docker compose -p cellapp-local-minio -f infra/compose.yaml run --rm minio-init
```

恢复默认镜像可取消设置 `CELLAPP_MINIO_IMAGE`。不要对包含已有对象的卷直接切换未经验证的镜像；先在独立 Compose 项目和测试卷验证，记录镜像 digest 与实际版本。镜像地址及映射关系见 [Docker Hardened Images](https://docs.docker.com/dhi/) 与 [DaoCloud 镜像说明](https://github.com/DaoCloud/public-image-mirror/blob/main/README.md)。

宿主 Go 服务默认值与 `.env.example` 一致：

```dotenv
S3_ENDPOINT=http://127.0.0.1:9000
S3_REGION=us-east-1
S3_BUCKET=cellapp
S3_ACCESS_KEY_ID=cellapp-local
S3_SECRET_ACCESS_KEY=cellapp-local-development-only
```

这些是公开的本地开发凭证，真实外部存储需独立注入秘密。API 与管理控制台分别绑定本机 `9000`、`9001`，管理入口为 `http://127.0.0.1:9001`，数据保存在命名卷。停止和重启不删除对象，不要用 `down -v` 处理初始化失败。

初始化失败先检查状态、镜像下载、端口占用、Bucket/凭证配置，修复后重试：

```sh
docker compose -p cellapp-local-minio -f infra/compose.yaml ps -a
docker compose -p cellapp-local-minio -f infra/compose.yaml logs minio
docker compose -p cellapp-local-minio -f infra/compose.yaml run --rm minio-init
```

修改宿主端口需同时修改 Compose 映射和服务 `S3_ENDPOINT`；初始化容器仍使用内部 `http://minio:9000`。修改本地 Bucket/凭证时同步 MinIO、初始化服务和 Go 服务环境。不停止无关进程或删除旧卷来释放端口。

## SSH 远程开发时访问控制台

Compose 的 `9000`、`9001` 端口只绑定在运行 Docker 的主机 `127.0.0.1`。通过 SSH 开发时，本地浏览器中的 `localhost` 指向自己的电脑，需要在**本地电脑**打开另一个终端，用与开发连接相同的 SSH 主机名建立转发：

```sh
ssh -N -L 19001:127.0.0.1:9001 -L 19000:127.0.0.1:9000 <现有SSH主机别名>
```

保持该终端运行，在本地浏览器打开 `http://127.0.0.1:19001/`。本地使用 `19001/19000` 避免与已有端口冲突；远端仍只监听 loopback。若平时使用 SSH 配置中的 `ProxyJump` 或非标准端口，继续使用同一个主机别名即可，不需要开放 MinIO 公网端口。

## 外部存储与生产

显式提供 `S3_ENDPOINT`、`S3_REGION`、`S3_BUCKET`，将真实 `S3_ACCESS_KEY_ID`、`S3_SECRET_ACCESS_KEY` 从服务进程环境或秘密管理器注入，不写入仓库、产物或日志。生产缺少必填配置会拒绝启动，不使用本地回退。Bucket 必须事先存在，服务不会自动创建生产 Bucket；健康检查仅探测指定 Bucket，无需列出全部 Bucket。

Bucket 和任何独立下载/CDN 入口均须私有。仅对 S3 地址测试匿名 GET 拒绝不能证明另一下载域名也私有。

## 旧环境升级与回滚

已有外部后端保留原 `S3_ENDPOINT`、`S3_REGION`、`S3_BUCKET` 和完整凭证；不自动修改个人 `.env`、搬迁对象或删除云资源。默认切换用于新的本地环境，不迁移旧 Bucket 数据。

旧 `QINIU_ACCESS_KEY`、`QINIU_SECRET_KEY` 不再被服务读取。升级已有外部环境前，必须把两项真实密钥同时迁至 `S3_ACCESS_KEY_ID`、`S3_SECRET_ACCESS_KEY`，并保留原 Endpoint、Region、Bucket；只设置旧变量时生产服务拒绝启动。回滚旧服务时恢复它所需的原变量，保留数据库、Bucket 和对象卷。主动切换存储数据需另行安排数据库引用和对象的一致迁移，不把空 Bucket 当作迁移结果。

## 真实存储验收

只在独立测试 Bucket 和隔离实例运行，设置与该实例对应的 S3 环境：

```sh
RUN_S3_TESTS=1 go test -race -count=1 -run TestLiveS3 -v ./apps/server/internal/hosting
```

测试包含 Bucket 健康检查、随机对象写入、签名读取及内容核对、匿名 GET 拒绝、删除后不可读和探针清理。服务 `/health` 同时检查数据库与存储。完整浏览器与 CLI 发布、访客鉴权及删除验收使用真实 MinIO，还需要隔离数据库、配套 Web 产物和受信任 HTTPS 入口：`RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=<isolated-test-dsn> S3_ENDPOINT=<isolated-minio-url> S3_BUCKET=<isolated-bucket> go test -race -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting`。内存或协议存储替身不能代替真实 MinIO 验收。
