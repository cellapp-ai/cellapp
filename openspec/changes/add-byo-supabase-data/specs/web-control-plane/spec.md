# web-control-plane Spec Delta

## ADDED Requirements

### Requirement: Owner data backend settings

Web SHALL 在自有应用详情中展示当前数据后端绑定，并允许所有者提交或清除 Supabase URL 与 anon/public key。MUST 区分未绑定、已绑定、保存中、失败和结果不确定；MUST NOT 在应用列表中展示 anon key，MUST NOT 把 service role 当作可保存值，MUST NOT 在失败后自动重试写请求。

#### Scenario: Bind from the application page
- **WHEN** 所有者在自有应用详情提交合法的 Supabase URL 与 anon/public key 且服务端成功
- **THEN** 详情显示已绑定的 URL，并保留表单供再次更新；密钥不进入 URL 或浏览器持久存储

#### Scenario: Clear an existing binding
- **WHEN** 所有者确认清除绑定且服务端成功
- **THEN** 详情恢复为未绑定，访客随后读取 `/_hosting/data` 得到未配置结果

#### Scenario: Invalid or uncertain data update
- **WHEN** 输入不合法、请求失败，或写请求完成情况不明
- **THEN** 界面说明错误或建议刷新查看当前状态，不自动重试，也不把失败显示为已保存
