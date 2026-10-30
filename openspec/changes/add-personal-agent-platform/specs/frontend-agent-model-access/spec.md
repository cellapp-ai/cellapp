# Spec Delta

## Purpose

让已发布的纯前端 Agent 应用使用当前访客自己授权的云端模型连接，并保持 Cellapp 的应用来源、访客身份和个人密钥隔离，使应用无需持有模型供应商凭证。

## ADDED Requirements

### Requirement: Per-user per-app model grant
系统 SHALL 要求已登录且获准进入应用的使用者按供应商连接明确授权应用。同一使用者与应用可拥有多个有效连接授权，每个授权 SHALL 独立撤销；应用作者或应用脚本 MUST NOT 替使用者确认授权、读取密钥或给其他应用继承授权。已授权连接下的模型 ID 可由应用请求指定，因此 SDK 的本地默认模型不是权限边界。

#### Scenario: First model use
- **WHEN** 应用首次请求模型调用且当前使用者尚未授权连接
- **THEN** 调用被拒绝，并向使用者提供显示应用身份、模型连接及预计数据流向的授权入口

#### Scenario: Same user visits another app
- **WHEN** 使用者已授权应用 A，但应用 B 请求使用同一连接
- **THEN** B 仍需单独授权，不能复用 A 的授权

#### Scenario: User authorizes another provider
- **WHEN** 使用者已授权应用 A 使用连接 X，又授权其使用连接 Y
- **THEN** A 的模型列表包含 X 与 Y 的模型；撤销 X 后仅 Y 可继续调用

### Requirement: Bounded application model calls
应用域 SHALL 以 `/_cellapp/v1` 作为模型 API base URL，提供 OpenAI 形态的 `GET /models` 与 `POST /chat/completions`；后续 Responses 路径 SHALL 为复数 `/responses`。`/models` SHALL 返回已获授权供应商连接的模型并集，以不冲突的稳定 ID 与供应商名称供前端选择；供应商模型列表获取失败时 SHALL 保留手工配置模型 ID。调用请求 SHALL 包含模型 ID，并只允许该 ID 所属连接已被当前使用者授权给当前应用；MUST NOT 接受应用提供的供应商地址或密钥。服务端使用当前应用来源和访客会话逐次检查应用状态、准入、连接授权及撤销状态；缺少授权时 SHALL 返回可识别错误和控制域授权页地址。首版支持 Chat Completions 文本、流式输出和工具调用子集；连接测试结果仅约束测试模型，对其他模型由供应商响应判定能力。系统 SHALL 限制请求大小、输出上限、并发、时长和请求频率；工具的实际执行仍由应用或本地客户端按自身权限完成。

#### Scenario: Authorized streaming call
- **WHEN** 已获准用户对已授权连接发起该连接支持的流式调用
- **THEN** 系统以受限的流式结果返回模型输出，并将上游失败或断流报告为失败，不把部分输出当作完整成功

#### Scenario: Revoked or blocked request
- **WHEN** 用户撤销应用授权、应用访问权失效、连接被删除或请求超出限制
- **THEN** 后续请求被拒绝并返回可识别错误，不向模型供应商发送新调用

### Requirement: User control and usage visibility
系统 SHALL 允许用户查看哪些应用可使用自己的连接、撤销单个应用授权，并查看按应用和连接归属的调用用量及失败状态。用量信息 SHALL 区分供应商返回的 token 数据与 Cellapp 的请求计数，不把估算金额或缺失用量报告为精确账单。

#### Scenario: Revoke one app
- **WHEN** 用户撤销应用 A 对连接 X 的授权
- **THEN** A 的后续调用失败，其他应用已获授权的调用不受影响

#### Scenario: Missing upstream usage
- **WHEN** 上游流在返回最终用量前中断或供应商不返回 token 统计
- **THEN** 系统仍记录请求及失败状态，并将 token 用量标记为未知
