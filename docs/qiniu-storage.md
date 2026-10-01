# 七牛存储历史入口

此路径保留供旧验收记录链接使用。当前开发默认使用本地 MinIO，供应商无关的配置、初始化及验收见 [S3 存储配置](s3-storage.md)。

已有七牛 S3 环境保留原 Endpoint、Region、Bucket；旧 `QINIU_ACCESS_KEY` / `QINIU_SECRET_KEY` 在新版服务中不再生效，升级前须将两项密钥迁至通用 S3 变量。具体迁移、失败行为和回滚方式见新文档。历史七牛验收不代表当前 MinIO 验收，也不触发云对象自动迁移。
