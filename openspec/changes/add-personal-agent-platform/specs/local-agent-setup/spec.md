# Spec Delta

## Purpose

让用户用已有编码 Agent 或手动指引建立 Cellapp 本地开发环境，以只读诊断和本机首选项辅助准备，同时保持跨设备开发任务在本变更中关闭。

## ADDED Requirements

### Requirement: Existing-agent local setup
首版 SHALL 提供可由用户已有编码 Agent 阅读的公开设置指令或 Skill，指导其使用 Cellapp CLI 检查环境、完成浏览器登录和准备项目；没有可用编码 Agent 的用户 SHALL 能按同一指引手动操作。本机设置 MUST NOT 要求个人模型连接、内置 Boot Agent、设备配对或常驻进程。

#### Scenario: Set up with an existing agent
- **WHEN** 用户让已可用的本地编码 Agent 按 Cellapp 指令准备项目
- **THEN** 该 Agent 可使用公开 CLI 完成检查和经用户授权的操作，不需要启动 Cellapp 自带 Agent 或配置用于引导的模型

#### Scenario: Set up without a coding agent
- **WHEN** 用户没有已安装的编码 Agent
- **THEN** 公开指引提供可手动执行的本机检查、浏览器登录与项目准备步骤，不把缺少 Agent 当作 Cellapp CLI 的启动前提

### Requirement: Read-only Cellapp doctor
首版 SHALL 提供 `cellapp doctor`，在无模型、未登录或网络不可用时仍完成本机只读检查。报告 SHALL 覆盖 CLI 运行条件、本地配置、受支持 Agent CLI 的安装与可确认登录状态，以及显式指定项目的准备情况；网络和登录检查失败 MUST 不阻断本地结果。每项 SHALL 区分通过、需处理和无法确认，并给出下一步；MUST NOT 自动安装、登录、构建、读取其他 Agent 的凭证文件或输出秘密。

#### Scenario: Diagnose while offline or signed out
- **WHEN** 用户在离线或尚未登录 Cellapp 的电脑运行 `cellapp doctor`
- **THEN** 命令仍返回本机检查结果，将无法验证的联网状态标为无法确认或需登录，不声称远端服务可用

#### Scenario: Inspect a selected project
- **WHEN** 用户显式传入 `--project` 指定本机目录
- **THEN** 命令只读报告该目录、项目关联、构建脚本与静态产物线索，不执行构建或扫描无关目录

#### Scenario: Agent login cannot be verified
- **WHEN** 受支持 Agent CLI 已安装，但其官方状态命令无法可靠确认登录
- **THEN** 报告已发现的 CLI 与版本，将登录状态标为未知，不读取其凭证文件或推断用户已登录

### Requirement: Explicit local agent selection
首版 SHALL 允许用户或其本地 Agent 通过 `cellapp set agent <名称>` 选择受支持且已安装的编码 CLI。命令 SHALL 在本机保存选择，并在 `cellapp doctor` 中展示；MUST NOT 接受任意 shell 命令、自动安装或登录该 Agent、执行编码任务，或将选择视为远程任务及项目权限授权。

#### Scenario: Select an installed agent
- **WHEN** 用户运行 `cellapp set agent codex` 且受支持的 Codex CLI 已安装
- **THEN** 命令将其记录为本机首选项，后续诊断能显示该选择，而不启动 Codex 或向 Cellapp 同步默认 Agent

#### Scenario: Unknown or unavailable executable
- **WHEN** 名称不受支持、对应 CLI 未安装，或参数包含附加命令
- **THEN** 命令拒绝并保持原有首选项，不执行输入内容

### Requirement: Remote coding unavailable in the first release
首版 MUST NOT 开放设备配对、心跳、云端项目远程授权或 Boot 模型会话作为本机引导入口；平台 MUST 拒绝从手机或其他电脑提交的开发任务，不排队或在未来设备上线后自动执行。本机选择了 Agent CLI 也不代表远程编码已获授权。

#### Scenario: Submit a remote development task
- **WHEN** 用户在首版尝试提交远程开发任务
- **THEN** 平台明确返回该能力尚不可用，本机 CLI 不领取或执行该任务
