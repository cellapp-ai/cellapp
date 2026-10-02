# app-data-access Spec Delta

## Purpose

让所有者把自有 Supabase 项目绑定到静态应用，使已授权访客在浏览器中共用一份数据，同时保持平台不运行用户后端、不代理查询。

## ADDED Requirements

### Requirement: Owner-managed public Supabase binding

系统 SHALL 允许应用所有者绑定、读取和清除该应用的公开数据后端配置。本期 provider MUST 为 `supabase`；配置 MUST 包含 HTTPS origin 与 anon/public key，且三者同时存在或同时为空。系统 MUST NOT 接受 service role 或 `sb_secret` 凭据，MUST NOT 将配置写入 `cellapp.json` 或部署产物。

#### Scenario: Bind a Supabase project

- **WHEN** 所有者对自有应用提交 `provider=supabase`、合法 HTTPS origin 与 anon/public key
- **THEN** 系统保存配置并在后续读取中返回 `provider`、`url`、`anonKey` 和 `dataset=shared`

#### Scenario: Replace or clear the binding

- **WHEN** 所有者提交新的合法配置，或明确清除绑定
- **THEN** 新配置立即替换旧值，或三列恢复为空；已发布静态文件保持不变

#### Scenario: Reject private credentials or invalid input

- **WHEN** 请求使用非 https origin、含用户信息的 URL、非 supabase provider、service role JWT 或 `sb_secret` 前缀
- **THEN** 系统拒绝保存，现有绑定不变，并返回可定位的 `invalid_data` 错误

#### Scenario: Foreign or missing app

- **WHEN** 请求针对其他所有者的应用、已删除应用或不存在的应用
- **THEN** 系统返回 `404 app_not_found`，不泄露配置

### Requirement: Session-gated public config for visitors

应用网关 SHALL 在分享密钥会话有效时向该应用 origin 提供 `GET`/`HEAD /_hosting/data`。未绑定 MUST 返回 `404 data_not_configured`。该响应 MUST NOT 计入应用流量额度。平台 MUST NOT 代理 Supabase 请求或签发 service role。

#### Scenario: Authorized visitor reads config

- **WHEN** 持有当前有效访问会话的访客请求 `/_hosting/data`
- **THEN** 若已绑定则返回公开 JSON；若未绑定则返回 `404 data_not_configured`

#### Scenario: Unauthenticated or other-app session

- **WHEN** 无会话、过期会话、密钥已重置，或其他应用的会话请求 `/_hosting/data`
- **THEN** 系统不返回配置；文档导航仍可进入密钥输入，非导航返回 401

### Requirement: Shared dataset without in-app accounts

绑定的数据集 SHALL 由该应用全部有效访客共用。系统 MUST NOT 把访客映射为独立 Supabase 用户，MUST NOT 用分享密钥签发 Supabase JWT。Skill 与说明 MUST 写明：分享密钥保护页面，anon key 对能打开应用的人可见，数据暴露面由该项目的 RLS 决定。

#### Scenario: Two visitors of the same app

- **WHEN** 两个已授权访客从同一应用读取数据配置
- **THEN** 二者获得同一 `url` 与 `anonKey`，可访问同一 Supabase 项目中的数据
