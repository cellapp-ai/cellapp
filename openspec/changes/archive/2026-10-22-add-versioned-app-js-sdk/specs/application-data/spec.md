# Spec Delta

## ADDED Requirements

### Requirement: Versioned application table contract
应用域 `/_cellapp/db/{table}` SHALL 接受与应用身份接口一致的可选运行时 API 版本选择。SDK SHALL 将其内部 Supabase 表客户端生成的请求转换到该公开路径；浏览器不需要知道内部的 `/rest/v1` 路径。显式版本固定对外支持的表操作及错误语义。任何版本 MUST 继续逐次执行当前应用、用户、匿名开关、表和行权限检查。旧 `/rest/v1/{table}` SHALL 不再作为公开表接口，也不得回退为静态页面。

#### Scenario: Pinned table request
- **WHEN** 已获准用户通过 SDK 请求 `/_cellapp/db/notes?v=1.1` 或通过等价日期别名请求 `/_cellapp/db/notes?v=261022`，并执行该契约支持的表操作
- **THEN** 请求按该契约处理，并仍受当前表策略和行策略限制

#### Scenario: Anonymous or revoked request under an older contract
- **WHEN** 匿名访问开关被关闭或用户访问权被撤销，而客户端仍使用受支持的旧 API 版本
- **THEN** 后续数据请求按当前权限拒绝，不恢复历史授权
