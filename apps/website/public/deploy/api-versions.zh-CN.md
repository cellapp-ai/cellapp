# Cellapp 应用 API 版本

[English](./api-versions.md) · [版本记录](./api-changelog.md)

应用运行时 API 包括 `/_cellapp/me` 和 `/_cellapp/db/{table}` 下已支持的表操作。它与 SDK npm 包版本、`/_cellapp/db` 路径及 `cellapp.data.json` 表清单版本分别管理。

## 选择固定契约

每份契约有两个等价别名：语义号 `major.minor` 与日期号 `yymmdd`。例如 `1.1 = 261022`：

```text
GET /_cellapp/me?v=1.1
GET /_cellapp/me?v=261022
```

两个请求使用同一处理逻辑、响应和错误契约；一个日期别名只能对应一份契约。破坏兼容的语义变化递增 `major`，新增能力形成新的 `minor`；兼容修复可继续沿用已有契约。

不带 `v` 的请求使用服务端最新版，旧的直接调用也遵循此规则。平台的**默认稳定版**可以落后于最新版。新 SDK 构建若未显式选版本，构建工具会读取一次默认稳定版，并把它的 `major.minor` 号写入每个输出 HTML。之后调整默认值不会改变已构建的应用；恢复旧静态产物会恢复它固化的版本，已打开的页面继续使用加载时的版本。

SDK 的身份和表请求都带 `v`。不受支持、格式错误、重复或已停用的 `v` 返回 `400 unsupported_api_version`、请求别名与本指南链接，不会自动改用最新版。固定版本只选择对外 API 行为；登录、应用准入、表权限及行策略仍逐次检查。

排查问题时，先检查已发布 HTML 中的 `<meta name="cellapp-api-version">`，再用版本号和错误码查询[版本记录](./api-changelog.md)。不要从 SDK 包版本或静态发布 ID 推断 API 契约。应用所有者仍通过 Cellapp CLI 管理表与策略，浏览器代码不能使用所有者控制接口。
