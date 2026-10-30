# Spec Delta

## Purpose

让个人用户安全地连接自己付费的公网 OpenAI 兼容模型服务。本变更供已发布应用中的使用者按连接授权调用，不向应用作者或其他用户泄露凭证。

## ADDED Requirements

### Requirement: Personal cloud model connection
系统 SHALL 允许已登录用户为自己的连接设置显示名称、公网 HTTPS API 基址、密钥，以及可选的测试模型 ID；未填写时 SHALL 尝试从供应商 `GET /models` 取得模型 ID，发现失败 SHALL 提示手动填写。首版 SHALL 仅接受约定的 OpenAI Chat Completions 协议；本机、私网与非 HTTPS 地址 MUST 被拒绝。连接的所有者与应用作者可以是不同用户。

#### Scenario: Configure a compatible provider
- **WHEN** 用户提交合法的公网 HTTPS 地址、个人密钥和模型 ID，或供应商模型列表可被成功获取
- **THEN** 系统创建仅该用户可管理的连接并返回不含密钥的连接信息

#### Scenario: Unsafe or unreachable target
- **WHEN** 用户配置本机或私网地址、非 HTTPS 地址、不可用的模型接口或不匹配的协议
- **THEN** 系统拒绝或标记测试失败，不向受限地址发送凭证，也不报告连接可用

### Requirement: Connection test and declared capabilities
系统 SHALL 以连接的测试模型执行有界测试，分别报告文本响应、流式响应及工具调用是否可用；未通过的能力 MUST 不被呈现为已支持。测试结果仅适用于该模型，同一连接下其他模型可发起调用，其实际能力以供应商响应为准。服务端 SHALL 保留可识别的上游错误类别，不把连接失败报告为模型成功响应。

#### Scenario: Partial compatibility
- **WHEN** 服务支持文本响应但不支持工具调用或流式输出
- **THEN** 测试模型仍可用于已验证能力，界面明确标记该测试模型不支持的能力，依赖该能力的调用失败并说明原因；连接下其他模型不沿用该测试结果

### Requirement: Secret lifecycle and ownership
系统 SHALL 将模型密钥作为可更换、可删除的个人秘密保管；读取连接、应用配置、静态产物、日志及错误响应 MUST NOT 返回密钥明文。只有连接所有者能更换或删除；删除或撤销后，新的模型请求 MUST 不能继续使用该连接。

#### Scenario: Another user reads a connection
- **WHEN** 应用作者、其他使用者或未登录请求读取某用户的连接
- **THEN** 系统不返回连接详情或密钥，也不允许修改它

#### Scenario: Rotate or remove a key
- **WHEN** 所有者更换密钥或删除连接
- **THEN** 后续请求使用新密钥或被拒绝；应用仍保留自身发布状态而不取得旧密钥
