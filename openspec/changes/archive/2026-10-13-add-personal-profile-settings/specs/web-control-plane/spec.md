## ADDED Requirements

### Requirement: Personal profile settings

Web SHALL 提供当前登录账户的个人 Profile 页面，显示完整账户 ID 和当前登录模式，并允许设置或清除显示名称。资料读取与更新 MUST 以浏览器会话确定账户，不接受客户端指定账户 ID；显示名称更新 MUST 不改变账户身份、应用地址或访问权。

#### Scenario: Open profile and save a name
- **WHEN** 已登录用户打开 `/profile` 并提交合法显示名称
- **THEN** 页面显示真实账户信息，服务端只更新当前账户的名称，刷新页面后仍显示保存值

#### Scenario: Clear or reject a name
- **WHEN** 用户提交空字符串，或提交超过 80 个字符、首尾空白、控制字符或无效 JSON
- **THEN** 空字符串清除显示名称；无效输入返回明确错误且不修改已保存名称

#### Scenario: Authentication and origin boundaries
- **WHEN** 无会话或跨站来源请求资料更新
- **THEN** 请求被拒绝且不改变任何账户资料；现有 CLI Bearer 凭证不能替代浏览器会话

#### Scenario: Refresh and uncertain result
- **WHEN** 用户刷新 `/profile` 深链，或更新请求失败且结果不确定
- **THEN** 有效深链加载页面；失败不报告成功或自动重试写入，并提供重新读取当前资料的操作
