# 运维与上线验收

## 生产配置

从 `.env.example` 创建自己的配置，设置 `NODE_ENV=production` 和 `AUTH_MODE=github`，替换 GitHub OAuth、S3、数据库、SECRET 和域名。GitHub OAuth App 的 callback URL 必须精确设置为 `${CONTROL_ORIGIN}/auth/callback`。`AUTH_MODE=dev` 只供 localhost 非生产开发，生产或非本地控制域会拒绝启动。服务启动时检查必填配置和正数额度；SECRET 必须稳定保存，变更会使所有摘要凭证不可用。

控制域和应用域必须使用不同注册域，例如 `control.example.com` 与 `*.example-apps.net`；精确 Host 路由隔离应用。反向代理终止 TLS，保留原始 Host，不将应用内容与控制面共享缓存。后端端口仅暴露给代理，不能开放给公网。网关响应为 `private, no-store`。

对象 bucket 必须私有。运行匿名 GET 验收时应返回拒绝，不得返回对象内容；仅服务端具备读写权限。不要在客户端发放对象读取签名 URL。对象上传使用有界临时文件，配置足够的临时磁盘并限制并发入口，上传失败不会发布版本。

额度涵盖应用数、文件数、单文件/部署大小、全部暂存和保留版本占用、每应用 UTC 月流量。`UPLOAD_TTL_SECONDS` 和 `RETENTION_SECONDS` 控制回收期限。免费额度耗尽返回错误，不收费。具体数值必须在上线前结合预算配置。

## 迁移、清理和停用

```sh
./dist/ohmyapp-server -migrate
./dist/ohmyapp-server -cleanup
./dist/ohmyapp-server -metrics
./dist/ohmyapp-server -suspend APP_ID
```

维护命令使用服务端数据库权限，应仅向运营人员开放。服务运行时每分钟清理一次；失败记录 `cleanup_failed`，修复依赖后可重试。`-metrics` 输出应用总数和已预留存储；数据库 `usage` 表提供按应用/月份流量明细。停用即时阻止新分发，并提升密钥代数，使已有会话失效。

请求日志只记录请求 ID、方法和耗时，不记录 URL 参数、请求正文、Authorization、Cookie 或分享密钥。用返回的 `X-Request-ID` 关联问题；需要详细诊断时应增加安全的错误码，而不能记录原始请求。

## 备份和回滚

定期使用 PostgreSQL 的备份工具导出控制数据库，并备份私有对象 bucket；两者必须覆盖同一个可恢复时间范围。秘密配置单独备份，不放仓库。恢复演练确认 active_deployment 指向的对象完整、密钥摘要仍能用原 SECRET 验证。

升级前先备份并应用向后兼容迁移，再替换服务镜像/二进制。服务回滚使用上一版本，保留数据库和对象。不要用 `docker compose down -v` 作为回滚，这会删除持久数据。本期没有用户可操作的版本回滚界面。

## 上线验收

- 使用真实 GitHub OAuth 在浏览器授权 CLI，确认账号绑定使用 GitHub 数字用户 ID，拒绝和过期请求不能领取 token。
- 发布 HTML 和 SPA，分别用两个浏览器输入密钥；深层页面可访问，缺失资源为 404。
- 未授权访问 HTML、JS、图片、HEAD、条件请求和对象地址均不能读取内容。
- 更新失败仍返回旧版本；重置后旧密钥/旧 cookie 不可读取；删除后立即拒绝分发。
- 检查跨账号管理拒绝、跨应用存储隔离、Service Worker 注册被拒绝。
- 在小额度配置下测试并发创建、上传及流量超限，并验证清理回收。
- 验证备份恢复、运营停用、Secret 脱敏及代理端口隔离。

以上真实环境验收未完成前，不应把单元测试通过解释为已可公网运营。
