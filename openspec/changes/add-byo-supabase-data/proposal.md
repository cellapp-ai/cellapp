# Proposal

## Why

托管应用目前只能分发静态文件。访客数据留在单台浏览器，跨设备无法共用。所有者已经能把前端直连外部 BaaS，但 Skill 把“带数据库的项目”一律拒绝，平台也不保存或下发公开配置。需要在不运行用户后端、不代理查询的前提下，让静态应用绑定所有者自带的 Supabase 项目，使同一应用的分享密钥访客共用一份数据。

## What Changes

- 所有者可为应用绑定自有 Supabase 项目的 HTTPS URL 与 anon/public key；绑定独立于发布版本，可在不重新部署的情况下更新或清除。
- 已授权访客从应用域读取这份公开配置，由浏览器直连 Supabase；Cellapp 不执行 SQL、不转发 Rest/Realtime、不代建项目、不持有 service role。
- 同一应用的访客共用该项目中的数据，不引入应用内账号或把分享密钥映射为 Supabase JWT。
- CLI、控制台和 Skill 支持绑定/查看/清除；Skill 仍拒绝 SSR、本机后端和 service role，但允许静态前端使用外部 Supabase。
- 增量迁移执行器开始跟踪已应用版本，再增加应用数据列；不改写 `001_initial.sql`。

## Capabilities

### New Capabilities

- `app-data-access`: 所有者自带 Supabase 绑定、公开配置下发、访客共用数据集，以及拒绝私有密钥与平台代建。

### Modified Capabilities

- `skill-deployment`: 区分“不可部署的用户后端”与“静态前端连接外部数据”；明示 localStorage 仍不同步，外部数据按绑定项目共享。
- `web-control-plane`: 应用详情提供绑定、更新和清除自有 Supabase 配置的真实状态。
- `app-key-access`: 平台数据配置端点与静态资源一样先验证分享密钥会话。

## Impact

影响 `apps/server` 迁移与网关/控制 API、`apps/web` 应用详情、`skills` CLI/Skill/HTTP 契约，以及 README、Web API 与数据访问说明。不改变发布原子性、分享密钥、私有对象存储或控制面 PostgreSQL 的职责。anon key 对已授权访客可见，文档必须说明数据库暴露面等于该密钥的 RLS。
