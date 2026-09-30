# 七牛云 S3 配置

存储服务已选择七牛云 Kodo 的 S3 兼容接口。Bucket 由项目配置提供，AccessKey/SecretKey 只从服务进程的系统环境变量读取；当前未创建云资源。

以华东-浙江为例，在 `.env` 只设置非秘密配置：

```dotenv
S3_ENDPOINT=https://s3.cn-east-1.qiniucs.com
S3_REGION=cn-east-1
S3_BUCKET=控制台显示的S3空间名
```

在启动服务的系统环境或服务管理器中注入凭证，不要写入 `.env` 或仓库文件：

```sh
export QINIU_ACCESS_KEY='你的AccessKey'
export QINIU_SECRET_KEY='你的SecretKey'
```

其他区域必须使用对应的 Region 和 Endpoint。`S3_BUCKET` 使用空间概览中的 **S3 空间名**，不一定等于用户可见的普通空间名称。服务端采用 path-style 地址与 AWS Signature V4，不向访问者提供对象签名直链。对应规则见七牛官方[服务域名与 S3 空间名说明](https://developer.qiniu.com/kodo/4088/s3-access-domainname)和[兼容工具设置](https://developer.qiniu.com/kodo/4096/s3-compatible-sdk)。

将空间设为私有，检查任何已绑定的七牛下载域名/CDN 域名也不会公开提供对象。七牛 S3 Endpoint 本身不允许匿名访问，因此仅对 S3 地址测试匿名失败，不能单独证明其他下载域名也私有。不要把公开 CDN 地址配置为 S3_ENDPOINT。

服务端需要该空间的 HEAD Bucket、PUT/GET/HEAD/DELETE Object 权限；健康检查仅查询指定 Bucket，不需要列出账号的全部空间。实现采用有界单对象上传，避免不必要的分片会话；配置的单文件上限应保持在服务支持的单次上传范围内。接口范围见[七牛兼容 API](https://developer.qiniu.com/kodo/4087/compatible-s3-api)。

## 配置后的验收

确认当前服务进程能够继承 `QINIU_ACCESS_KEY` 和 `QINIU_SECRET_KEY`，加载 `.env` 后启动服务并访问控制域 `/health`。然后显式运行云存储冒烟测试：

```sh
RUN_S3_TESTS=1 go test -count=1 -run TestLiveS3 -v ./apps/server/internal/hosting
```

测试在 `cellapp-probes/<随机ID>` 写入一个很小的对象，验证读取、匿名拒绝及删除；不会创建或删除 Bucket，不读取业务对象。异常中断留下的该前缀对象可由所有者清理。测试可能产生少量云端请求与存储用量；只有显式设置开关才执行。

之后用真实账号完成一次 CLI 发布，验证 `/health`、资源访问、密钥重置和删除后的回收，并人工检查七牛原生下载域名的私有访问设置。这些步骤在凭证尚未配置时保持待验收。
