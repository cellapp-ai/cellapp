# Spec Delta

## ADDED Requirements

### Requirement: Versioned browser model client
官方 JavaScript SDK SHALL 提供 `models.current()`：列出当前应用已获授权的模型，读取当前浏览器对此应用的本地默认模型；未选择时由 SDK 弹窗引导使用者授权供应商连接和选择模型。SDK SHALL 只在应用来源本地保存选择偏好，不通过独立接口向 Cellapp 服务端同步默认模型；模型调用请求仍须携带所选 `model`。SDK SHALL 提供使用该模型的文本与流式调用，沿用浏览器应用会话，不接受供应商地址或密钥，也不得返回或持久化密钥。未授权、能力不支持、限速、上游错误和流中断 SHALL 提供可区分的结果；SDK 弹窗不替代服务端逐次鉴权。

#### Scenario: Authorized frontend agent
- **WHEN** 已获准用户在已发布应用中使用 SDK 调用自己授权的兼容模型连接
- **THEN** SDK 请求当前应用的版本化模型接口并返回文本或流式结果，不把控制域凭证和供应商密钥交给应用

#### Scenario: Missing authorization or incomplete stream
- **WHEN** 用户尚未授权该应用，或模型流在结束前中断
- **THEN** SDK 报告相应状态，不把未完成结果当作完整成功响应
