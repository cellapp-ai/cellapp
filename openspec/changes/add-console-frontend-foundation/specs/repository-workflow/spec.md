## MODIFIED Requirements

### Requirement: Single client source and supported commands

项目 SHALL 将公开 CLI/shared TypeScript 包统一维护于 `skills/packages/`，将控制台前端应用维护于主仓库 `apps/console`；客户端单元测试和 npm 发布产物由子仓库管理，控制台验证及跨服务验收由主仓库管理。主仓库 MUST 保留文档化的 CLI、类型检查、构建和测试入口，在完整检查与构建中纳入控制台，缺少子模块时返回可操作的失败信息；控制台独立开发与验证入口不得依赖子仓库初始化。

#### Scenario: Build and run from parent
- **WHEN** 开发者完成文档中的两个仓库依赖安装，并从主仓库执行常用开发命令
- **THEN** CLI、类型检查与客户端构建使用 `skills/` 中的唯一实现，完整类型检查覆盖控制台，完整构建包含 Go 服务端、公开客户端及控制台，测试入口包含客户端、服务端及控制台检查

#### Scenario: Client compatibility after consolidation
- **WHEN** 使用现有 `ohmyapp` 命令别名、旧项目状态或旧凭证进行客户端操作
- **THEN** 保留子仓库既有兼容策略、命令结果和秘密保护，新增前端工程不导致新建错误应用或丢失原有凭证

#### Scenario: Standalone console development
- **WHEN** 开发者安装主仓库依赖并独立启动或检查控制台，且 `skills/` 尚未初始化
- **THEN** 控制台独立入口仍可执行；完整仓库命令继续明确提示如何初始化缺失的客户端子模块

#### Scenario: Console failure propagates
- **WHEN** 完整仓库命令执行过程中控制台类型检查、前端 lint、测试或构建失败
- **THEN** 对应命令返回非零退出码并提供可定位的错误，不因客户端或服务端阶段成功而报告完整检查通过

### Requirement: Actionable repository guidance

主仓库与子仓库 SHALL 分别提供可独立理解的 `AGENTS.md`，说明架构、目录职责、扩展及依赖规则、代码规范、验证命令和设计 → 确认 → 开发 → 测试 → 归档 → commit 流程；规范 MUST 区分已授权范围内的自主实施与需要重新确认的范围变化。主仓库 SHALL 在 `docs/standards/` 提供 `README.md`、`frontend.md`、`frontend-code.md` 和 `design.md` 作为前端规范入口，明确适用于 `apps/console` 的前端工程、代码与设计，不替代服务端或 CLI 规范；内容 MUST 使用当前项目自身口吻并区分实现事实、必须要求、建议与待改进项，不写入外部参考工程的名称、路径或历史状态。

#### Scenario: Work within confirmed scope
- **WHEN** Agent 实施已确认设计中的任务
- **THEN** 依据对应仓库规范自主完成开发和相关验证，不反复请求相同范围授权，交付报告说明实际检查结果及提交状态

#### Scenario: Scope or contract changes
- **WHEN** 实施需要扩展产品行为、破坏契约或改变已确认架构
- **THEN** 先更新相关设计并确认受影响决策，再继续依赖该决策的实施

#### Scenario: Frontend contributor selects applicable guidance
- **WHEN** 开发者从根 `AGENTS.md` 或 README 进入前端规范
- **THEN** 可以找到前端工程、`frontend-code.md` 和前端设计规则，并明确辨别其适用范围与服务端、CLI 规则的边界

#### Scenario: Documentation accurately represents validation
- **WHEN** 前端规范描述检查工具、可访问性目标或待改进项
- **THEN** 内容与本项目实际配置对应，不将建议写成已执行检查，不用历史或其他工程验收证明本轮结果
