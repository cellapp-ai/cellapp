# Design

## Context

Cellapp 只托管静态产物：本地构建、完整发布、网关按分享密钥会话读文件。控制面 PostgreSQL 存元数据，MinIO/S3 存对象。网关 CSP 不限制 `connect-src`，浏览器理论上已能直连外部 API，但平台没有应用数据配置，Skill 将数据库与后端一并拒绝。本期按已确认方向：所有者自带 Supabase（BYO），同一应用访客共用一份数据。

## Goals / Non-Goals

**Goals:** 为应用保存一份公开的 Supabase 连接配置；已授权访客可读；所有者可通过 CLI 与控制台绑定/清除；拒绝 service role；静态托管边界保持不变。

**Non-Goals:** 不代建 Supabase 项目、不代理 API、不把分享密钥换成 Supabase JWT、不做应用内多用户、不注入构建期环境变量、不收紧现有 CSP（避免误伤已直连其他 API 的页面）。

## Decisions

### 1. 配置存在应用元数据，不写入发布产物

在 `apps` 表保存 `data_provider` / `data_url` / `data_anon_key`。三者同为 NULL 或同为完整值。访客 `GET /_hosting/data` 在会话通过后返回 JSON；未绑定返回 `404 data_not_configured`。这样换项目不必重新发布，配置也不占用产物额度。

相对把 `cellapp-data.json` 打进静态包：后者与版本绑定，且会把密钥写入可下载文件清单。元数据更适合“先试 Supabase”。

```text
Owner CLI/Web -> PUT /apps/{id}/data -> PostgreSQL apps.data_*
Visitor JS    -> GET /_hosting/data  -> same-origin JSON
              -> browser             -> owner's Supabase (RLS)
```

### 2. 只接受公开凭据

`provider` 仅 `supabase`。URL 必须是无用户信息的 HTTPS origin。anon key 拒绝 JWT `role=service_role` 以及 `sb_secret` 前缀。dataset 固定为 `shared`，客户端不能改。anon key 对能打开应用的人可见，等同于打进前端包；文档与 Skill 必须写明：分享密钥挡住的是页面，不是绕过页面后的数据库调用。

`cellapp.json` 仍不含密钥。CLI `data` 只把值发给控制 API。

### 3. 增量迁移先有版本表

当前 `migrations.Run` 每次执行 `001_initial.sql`。按仓库约定，增加 `002` 前先引入 `schema_migrations`：已有 `apps` 表的库登记版本 1 而不改写 001；新库按序应用 001、002。002 只加可空数据列和完整性约束。

### 4. Skill 与控制台

Skill 继续拒绝 SSR/后端/service role，但应询问是否绑定已有 Supabase，并指导页面 `fetch('/_hosting/data', {credentials:'same-origin'})` 后用官方客户端连接。控制台在应用详情展示绑定表单；清除需确认；写失败不自动重试。列表不返回 anon key。

## Risks / Trade-offs

- 持有 anon key 的人可在应用域外调用 Supabase。这是 BYO + 共享数据集的固有面，用文档和禁止 service role 约束，不做假安全。
- 不收紧 CSP，恶意或误用页面仍可连接任意主机；本期优先兼容，把连接白名单留给后续。
- 配置与发布解耦：删除应用仍立刻停止配置读取；旧前端若把密钥写死在产物里，清除绑定不能收回已发布文件中的值。

## Migration Plan

1. 部署带版本跟踪的迁移执行器与 `002_app_data.sql`。
2. 发布控制 API、网关、CLI、Skill、控制台。
3. 回滚：停止新二进制后数据列可留空；清除绑定将三列置 NULL。不回退 001。

## Open Questions

无。供应商扩展、JWT 桥和平台代建不在本期。
