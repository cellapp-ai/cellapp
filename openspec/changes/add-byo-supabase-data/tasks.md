# Tasks

## 1. 迁移执行器与数据列

- [x] 1.1 为 `apps/server/migrations` 增加 `schema_migrations` 版本跟踪：已有 `apps` 表登记版本 1，不改写 `001_initial.sql`；重复执行保持幂等
- [x] 1.2 新增 `002_app_data.sql`，为 `apps` 增加可空的 `data_provider`/`data_url`/`data_anon_key` 及完整性约束
- [x] 1.3 覆盖从仅有 001 的库升级、全新库和重复迁移

## 2. 控制 API 与网关

- [x] 2.1 实现所有者 `GET`/`PUT`/`DELETE /apps/{app}/data`，校验 HTTPS origin、supabase provider、拒绝 service role/`sb_secret`，外应用 404
- [x] 2.2 网关在有效访问会话下提供 `GET`/`HEAD /_hosting/data`；未绑定 404；不计入流量；其他 `/_hosting/` 仍 404
- [x] 2.3 增加 Go 集成测试：绑定/替换/清除、非法密钥、跨应用会话、未登录、控制台同源写请求

## 3. CLI、契约与 Skill

- [x] 3.1 扩展 `skills/contracts/http-v1.json` 与根契约，增加 data 操作；契约测试覆盖这些路径
- [x] 3.2 CLI 增加 `data` 命令；校验公开配置；不把 anon key 写入 `cellapp.json` 或日志
- [x] 3.3 更新 Skill：允许静态+外部 Supabase，拒绝后端/SSR/service role，说明共用数据与 `/_hosting/data` 用法
- [x] 3.4 更新客户端 README/AGENTS 中与“无数据库”冲突的表述

## 4. 控制台

- [x] 4.1 浏览器 API 允许应用 data 写请求体；详情返回可空 `data`
- [x] 4.2 应用详情提供绑定/更新/清除表单，含未绑定、失败和不确定结果
- [x] 4.3 更新 `docs/web-api.md` 与 `apps/web` 运行时校验

## 5. 文档与验证

- [x] 5.1 更新 README/AGENTS 产品边界，并新增所有者数据访问说明
- [x] 5.2 执行 typecheck、相关测试、`go vet`；有隔离数据库时跑 race 集成测试；记录通过/失败/跳过/缺口
