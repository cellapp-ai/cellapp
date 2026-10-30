# Proposal

## Why

Cellapp 已能托管静态应用，也能让外部 Agent 通过 CLI 或 MCP 管理应用，但用户还缺少一条利用自己已有的编码 Agent 建立 Cellapp 本地开发环境的清晰路径；纯前端 Agent 也无法安全地调用使用者自己的云端模型。先建立个人模型连接和无需额外 Agent 运行时的本机引导，再逐步扩展到跨设备使用与 Agent 分发。

## What Changes

- 提供个人模型连接：已登录用户配置公网 HTTPS 的 OpenAI 兼容 API 地址、模型 ID 和个人密钥；Cellapp 受控入口按当前用户和应用授权转发模型调用，不向应用脚本交付密钥。首版以 Chat Completions 的文本、流式输出和工具调用为验收范围，具体能力在连接测试中确认。
- 允许已发布的纯前端 Agent 应用经应用域调用使用者授权的模型连接；未授权、撤销、超限或服务不可用时给出明确失败。
- 首版本机引导由用户已有的编码 Agent 阅读 Cellapp 的公开设置指令或 Skill，并使用 CLI 完成检查、登录与项目准备；没有编码 Agent 的用户可按同一指引手动操作。`cellapp doctor` 提供无需模型的只读环境诊断，涵盖 CLI 运行条件、Cellapp 连通与登录状态、受支持的本机 Agent CLI，以及显式指定的项目准备情况；无法确认的状态应明确标示。用户或其本地 Agent 可通过 `cellapp set agent <名称>` 选择已支持的编码 CLI，例如 `cellapp set agent codex`，不要求 Cellapp 另装一个 Boot Agent。
- 首版不为本机引导要求设备配对、在线心跳或项目远程授权；这些能力与远程任务一起留待后续阶段。已实现但不属于新版首版范围的试验性接口和运行时须在实施时清理或关闭，不作为首版验收成果。
- 后续阶段再提供个人设备的远程开发任务、进度与结果、私有预览和确认发布。用户从手机或其他电脑使用自己的在线设备；Agent、源码和构建仍在本地执行。远程编码的执行边界在该阶段实施前单独确定。
- 后续阶段再提供 Agent 定义/源码的版本化分享与个人副本。使用者在自己的设备安装副本，独立配置模型连接和本地权限；作者的密钥、私有数据及会话不随副本分发。纯前端 Agent 可直接使用已发布版本。
- 保留现有静态发布、应用访问和数据授权语义。现有 MCP 可作为 Agent 调用 Cellapp 的可选工具，不承担个人设备任务投递。
- 首版不支持远程编码、设备任务投递、本机模型接口、Anthropic 原生协议、平台售卖模型额度或云端运行用户代码。

## Capabilities

### New Capabilities

- `personal-model-connections`：个人 OpenAI 兼容云端接口的配置、秘密保管、连接测试和调用权限。
- `frontend-agent-model-access`：静态应用内的使用者授权模型调用、流式响应与资源限制。
- `personal-agent-remote-tasks`：首版提供现有 Agent 引导、`cellapp doctor` 和本机 Agent CLI 选择；设备绑定、项目远程授权、远程任务、预览与发布确认留待后续阶段。
- `agent-distribution`：后续阶段提供 Agent 包的版本化分享、个人副本、安装和更新授权。

### Modified Capabilities

- `control-api-routing`：首版为个人模型连接增加控制域路径，并为应用模型调用明确应用域隔离；设备与远程任务路径只在后续阶段开放。
- `application-js-sdk`：为已发布的纯前端 Agent 提供固定 API 契约版本下的模型调用方法。

## Impact

- 首版涉及主仓库 Go 控制 API、应用网关、控制数据库增量迁移、浏览器控制台及本机引导验收；`skills` 子仓库扩展 `cellapp doctor`、`cellapp set agent`、公开设置指令/Skill 与契约，保持 CLI/Skill 独立可用。
- 首版新增模型流式请求、秘密加密与轮换及本机环境检查，不依赖 mini-SWE-agent、Python/uv 或用于引导的模型调用。设备绑定、设备任务投递、私有预览及 Agent 包生命周期留待后续阶段；已有试验性实现的收敛与兼容处理纳入实施。部署路由与运维说明按阶段同步。
- 不改变现有应用静态产物、CLI 部署、MCP 授权及数据 API 的兼容契约。
