# S3 存储配置

服务使用私有 S3 兼容 Bucket 保存发布产物，开发默认使用本地 MinIO。访客仅通过应用网关读取文件，不获得对象读取签名直链。适配器使用 path-style 和 Signature V4，支持指定 Bucket 的 HEAD 及对象 PUT/GET/HEAD/DELETE；生产 Endpoint 必须为 HTTPS。

## 本地 MinIO

```sh
docker compose -f infra/compose.yaml up -d postgres proxy minio
docker compose -f infra/compose.yaml run --rm minio-init
```

第二条命令必须以零状态退出，出现 `Private local S3 bucket initialized.` 后才算存储就绪。初始化有界等待服务、幂等创建 Bucket 并关闭匿名访问；重复运行保留对象。首次运行需要能够访问固定版本镜像所在的镜像仓库。

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
docker compose -f infra/compose.yaml ps -a
docker compose -f infra/compose.yaml logs minio
docker compose -f infra/compose.yaml run --rm minio-init
```

修改宿主端口需同时修改 Compose 映射和服务 `S3_ENDPOINT`；初始化容器仍使用内部 `http://minio:9000`。修改本地 Bucket/凭证时同步 MinIO、初始化服务和 Go 服务环境。不停止无关进程或删除旧卷来释放端口。

## 外部存储与生产

显式提供 `S3_ENDPOINT`、`S3_REGION`、`S3_BUCKET`，将真实 `S3_ACCESS_KEY_ID`、`S3_SECRET_ACCESS_KEY` 从服务进程环境或秘密管理器注入，不写入仓库、产物或日志。生产缺少必填配置会拒绝启动，不使用本地回退。Bucket 必须事先存在，服务不会自动创建生产 Bucket；健康检查仅探测指定 Bucket，无需列出全部 Bucket。

Bucket 和任何独立下载/CDN 入口均须私有。仅对 S3 地址测试匿名 GET 拒绝不能证明另一下载域名也私有。

## 旧环境升级与回滚

已有外部后端保留原 `S3_ENDPOINT`、`S3_REGION`、`S3_BUCKET` 和完整凭证；不自动修改个人 `.env`、搬迁对象或删除云资源。默认切换用于新的本地环境，不迁移旧 Bucket 数据。

旧 `QINIU_ACCESS_KEY`、`QINIU_SECRET_KEY` 仍作为成对兼容入口。完整通用凭证组优先；仅当两个通用变量都为空时才回退旧组，所选组缺一项会报错，不混用密钥。确认旧配置可读后可把两项同时迁至通用变量。回滚旧服务需恢复对应旧变量和原存储参数，保留数据库、Bucket 和对象卷。主动切换存储数据需另行安排数据库引用和对象的一致迁移，不把空 Bucket 当作迁移结果。

## 真实存储验收

只在独立测试 Bucket 和隔离实例运行，设置与该实例对应的 S3 环境：

```sh
RUN_S3_TESTS=1 go test -race -count=1 -run TestLiveS3 -v ./apps/server/internal/hosting
```

测试包含 Bucket 健康检查、随机对象写入、签名读取及内容核对、匿名 GET 拒绝、删除后不可读和探针清理。服务 `/health` 同时检查数据库与存储。完整 CLI 发布、访客鉴权及删除验收还需要隔离数据库、配套 Web 产物和受信任 HTTPS 入口。内存或协议存储替身不能代替真实 MinIO 验收。
