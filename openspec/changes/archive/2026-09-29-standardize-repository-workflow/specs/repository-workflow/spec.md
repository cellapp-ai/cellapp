# Spec Delta

## Purpose

为主仓库与独立客户端仓库提供可复现的协作方式，使开发者能够获取配套版本、使用明确的源码位置和验证入口，并依照一致的设计、确认、开发、测试、归档及提交流程交付跨仓库变更。

## ADDED Requirements

### Requirement: Reproducible client checkout
主仓库 SHALL 将 `skills/` 作为独立 Git 子模块并固定具体提交，提供递归克隆、已有检出初始化和递归拉取说明；递归拉取 MUST 使用主仓库记录的客户端版本，不自动追随子仓库分支最新版本。

#### Scenario: Clone with client
- **WHEN** 开发者按文档递归克隆主仓库且具有两个远程仓库的读取权限
- **THEN** `skills/` 已初始化且 HEAD 等于主仓库记录的子模块提交，可独立执行客户端安装与检查

#### Scenario: Pull a recorded client update
- **WHEN** 主仓库更新子模块版本，开发者在干净且已初始化的检出中执行递归拉取
- **THEN** 客户端更新到主仓库记录的提交，不被子仓库额外的新提交影响

#### Scenario: Existing checkout and local work
- **WHEN** 开发者初始化已有检出或更新存在本地修改的子仓库
- **THEN** 文档提供初始化与状态检查方法，流程不丢弃或覆盖未提交工作；不能安全更新时明确报告冲突

### Requirement: Single client source and supported commands
项目 SHALL 将所有 TypeScript 包统一维护于 `skills/packages/`，客户端单元测试和 npm 发布产物由子仓库管理，跨服务验收由主仓库管理；主仓库 MUST 保留文档化的 CLI、类型检查、构建和测试入口，缺少子模块时返回可操作的失败信息。

#### Scenario: Build and run from parent
- **WHEN** 开发者完成文档中的两个仓库依赖安装，并从主仓库执行常用开发命令
- **THEN** CLI、类型检查与客户端构建使用 `skills/` 中的唯一实现，完整构建包含 Go 服务端，测试入口包含客户端与服务端检查

#### Scenario: Client compatibility after consolidation
- **WHEN** 使用现有 `ohmyapp` 命令别名、旧项目状态或旧凭证进行客户端操作
- **THEN** 保留子仓库既有兼容策略、命令结果和秘密保护，目录迁移不导致新建错误应用或丢失原有凭证

### Requirement: Actionable repository guidance
主仓库与子仓库 SHALL 分别提供可独立理解的 `AGENTS.md`，说明架构、目录职责、扩展及依赖规则、代码规范、验证命令和设计 → 确认 → 开发 → 测试 → 归档 → commit 流程；规范 MUST 区分已授权范围内的自主实施与需要重新确认的范围变化。

#### Scenario: Work within confirmed scope
- **WHEN** Agent 实施已确认设计中的任务
- **THEN** 依据对应仓库规范自主完成开发和相关验证，不反复请求相同范围授权，交付报告说明实际检查结果及提交状态

#### Scenario: Scope or contract changes
- **WHEN** 实施需要扩展产品行为、破坏契约或改变已确认架构
- **THEN** 先更新相关设计并确认受影响决策，再继续依赖该决策的实施

### Requirement: Compatible public client boundary
公开客户端仓库 SHALL 可以脱离主仓库独立安装、检查、构建和打包；其公开文件允许清单 MUST 接受客户端 `AGENTS.md`，继续排除服务端实现、基础设施、私有运维材料和秘密。

#### Scenario: Standalone client validation
- **WHEN** 开发者独立检出客户端并执行文档化检查与打包
- **THEN** 不依赖主仓库源码即可完成，公开边界检查接受 `AGENTS.md`，包包含 CLI/shared 与部署 Skill

### Requirement: Recoverable cross repository delivery
跨仓库交付 SHALL 在验证和归档完成后先提交子仓库，确认子仓库提交已可从远程获取，再提交主仓库的 gitlink 及关联变更；MUST 显式区分通过、跳过、未执行和失败，不以存在任务勾选代替验收，不夹带无关本地工作。

#### Scenario: Successful delivery
- **WHEN** 跨仓库变更完成相关验证、规格同步和归档
- **THEN** 子仓库提交可从远程获取，主仓库固定该提交，干净递归检出可以复现配套代码，交付记录与实际归档状态一致

#### Scenario: Incomplete verification or unavailable child commit
- **WHEN** 必要验收失败、尚未归档或目标子仓库提交无法从远程获取
- **THEN** 报告未完成事项并保留工作，不宣称完整交付，不提交指向不可获取子仓库提交的主仓库交付版本
