# 本轮实现与验证记录

变更：`static-app-hosting`。服务端按用户要求采用 Go；对象存储使用七牛云 S3，密钥仅从系统环境注入。OpenSpec 任务进度为 30/30，未归档。

## 已验证

- Go 服务端与 TypeScript CLI 构建成功；TypeScript 类型检查、Go vet、5 项 Node 测试通过。
- 19 项 Go 测试通过（包含真实 PostgreSQL 集成和真实 Chrome 测试）；并发相关运行启用 `-race`。
- 真实 PostgreSQL：独立 schema 迁移、唯一约束、事务回滚、应用/存储/流量额度、发布冲突、上传中断恢复、幂等重试、凭证到期/撤销、密钥重置及清理重试。
- 受控 GitHub OAuth 服务：验证授权码和 PKCE、GitHub `/user` 数字 ID 绑定及无效身份拒绝；开发认证验证固定本地账号且 OAuth callback 关闭；控制面测试覆盖 state、浏览器会话和退出。
- 真实 Chrome：设备授权、CLI 发布、两个浏览器会话、两个应用来源隔离、脚本加载、SPA 与资源 404、跨来源读取拒绝、Service Worker 拒绝、发布响应丢失重试、密钥重置、删除、浏览器凭证撤销与 CLI 退出。
- S3 适配器：在本机受控 HTTPS 协议服务上验证 Signature V4、path-style、指定 Bucket 探测、对象 PUT/HEAD/GET/DELETE。此项不是七牛云验收。
- 真实七牛 Bucket `acell-test`：使用 `cn-south-1` S3 Endpoint 验证健康检查、签名 PUT/GET、内容一致、匿名请求无法读取内容、DELETE 及删除后不可读；随机探针已清理。
- CLI 安装包在项目临时目录完成安装，`ohmyapp --help` 可运行；部署 Skill 格式校验通过；OpenSpec strict 校验通过。

浏览器流程使用真实浏览器、真实 PostgreSQL、测试专用身份提供方及内存存储替身；测试替身不进入生产二进制。测试浏览器使用隔离配置，允许测试服务的自签名证书；CLI 使用测试 CA 验证 TLS，没有关闭 CLI TLS 校验。

## 尚未执行的生产操作

本地完整服务已经启动并通过真实七牛验收。没有创建或删除 Bucket，也没有发布公网服务；生产 GitHub OAuth App、公网域名和 TLS 仍由上线环境提供。七牛配置与可重复冒烟测试见 [qiniu-storage.md](qiniu-storage.md)，运行方法见 [README](../README.md)。

## 重跑

```sh
npm run typecheck
npm run build
node --import tsx --test tests/*.test.ts
TEST_DATABASE_URL=postgres://ohmyapp:development@127.0.0.1:5432/ohmyapp go test -race -count=1 -v ./apps/server/...
RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=postgres://ohmyapp:development@127.0.0.1:5432/ohmyapp go test -race -count=1 -run TestBrowserEndToEnd -v ./apps/server/internal/hosting
go vet ./apps/server/...
openspec validate static-app-hosting --strict
```

测试创建并删除自己的随机数据库 schema。测试所用数据库服务保持本机运行，不清除用户数据卷。
