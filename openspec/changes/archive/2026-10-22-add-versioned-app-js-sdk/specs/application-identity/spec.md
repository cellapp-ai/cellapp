# Spec Delta

## ADDED Requirements

### Requirement: Versioned current application user contract
应用域 `GET /_cellapp/me` SHALL 按可选的运行时 API 版本返回当前应用用户信息。显式版本仅选择该接口的响应和错误契约；每次请求 MUST 继续验证当前会话与应用准入，不得暴露平台全局身份或其他应用的用户 ID。

#### Scenario: Pinned current user
- **WHEN** 已获准用户请求 `/_cellapp/me?v=1.1` 或其等价别名 `/_cellapp/me?v=261022`
- **THEN** 接口按该版本契约返回本应用稳定用户 ID 和访问状态

#### Scenario: Access revoked under old API version
- **WHEN** 用户的应用访问权已被撤销，但前端仍请求一个受支持的旧 API 版本
- **THEN** 接口拒绝访问，不因版本固定而恢复旧授权
