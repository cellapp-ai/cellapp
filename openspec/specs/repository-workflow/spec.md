# repository-workflow Specification

## Purpose

为主仓库与独立客户端仓库提供可复现的协作方式，使开发者能够获取配套版本、使用明确的源码位置和验证入口，并依照一致的设计、确认、开发、测试、归档及提交流程交付跨仓库变更。

## Requirements

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

项目 SHALL 将公开 CLI/shared TypeScript 包统一维护于 `skills/packages/`，将 Web 前端应用维护于主仓库 `apps/web`；客户端单元测试和 npm 发布产物由子仓库管理，Web 验证及跨服务验收由主仓库管理。主仓库 MUST 保留文档化的 CLI、类型检查、构建和测试入口，在完整检查与构建中纳入 Web，缺少子模块时返回可操作的失败信息；`dev:web`、`preview:web`、`lint:web` 及前端独立检查不得依赖子仓库初始化。原 `apps/console` 及对应开发脚本不再作为当前工程入口。

#### Scenario: Build and run from parent
- **WHEN** 开发者完成文档中的两个仓库依赖安装，并从主仓库执行常用开发命令
- **THEN** CLI、类型检查与客户端构建使用 `skills/` 中的唯一实现，完整类型检查覆盖 Web，完整构建包含 Go 服务端、公开客户端及配套 Web 静态产物，测试入口包含客户端、服务端及 Web 检查

#### Scenario: Client compatibility after consolidation
- **WHEN** 使用现有 `ohmyapp` 命令别名、旧项目状态或旧凭证进行客户端操作
- **THEN** 保留子仓库既有兼容策略、命令结果和秘密保护，Web 工程不导致新建错误应用或丢失原有凭证

#### Scenario: Standalone console development
- **WHEN** 开发者安装主仓库依赖并独立启动或检查 Web，且 `skills/` 尚未初始化
- **THEN** Web 独立入口仍可执行，真实管理需连接控制服务且不可达状态明确；完整仓库命令继续提示如何初始化缺失的客户端子模块

#### Scenario: Console failure propagates
- **WHEN** 完整仓库命令执行过程中 Web 类型检查、前端 lint、测试或构建失败
- **THEN** 对应命令返回非零退出码并提供可定位的错误，不因客户端或服务端阶段成功而报告完整检查通过

### Requirement: Actionable repository guidance

主仓库与子仓库 SHALL 分别提供可独立理解的 `AGENTS.md`，说明架构、目录职责、扩展及依赖规则、代码规范、验证命令和设计 → 确认 → 开发 → 测试 → 归档 → commit 流程；规范 MUST 区分已授权范围内的自主实施与需要重新确认的范围变化。主仓库 SHALL 在 `docs/standards/` 提供 `README.md`、`frontend.md`、`frontend-code.md` 和 `design.md` 作为前端规范入口，明确适用于 `apps/web` 的前端工程、代码与设计，不替代服务端或 CLI 规范；内容 MUST 使用当前项目自身口吻并区分实现事实、必须要求、建议与待改进项，不写入外部参考工程的名称、路径或历史状态。运行说明 MUST 包含同源控制服务、HTTPS 会话、静态产物配置及配套部署方法。

#### Scenario: Work within confirmed scope
- **WHEN** Agent 实施已确认设计中的任务
- **THEN** 依据对应仓库规范自主完成开发和相关验证，不反复请求相同范围授权，交付报告说明实际检查结果及提交状态

#### Scenario: Scope or contract changes
- **WHEN** 实施需要扩展产品行为、破坏契约或改变已确认架构
- **THEN** 先更新相关设计并确认受影响决策，再继续依赖该决策的实施

#### Scenario: Frontend contributor selects applicable guidance
- **WHEN** 开发者从根 `AGENTS.md` 或 README 进入前端规范
- **THEN** 可以找到适用于 `apps/web` 的前端工程、`frontend-code.md` 和前端设计规则，并明确辨别其与服务端、CLI 规则的边界

#### Scenario: Documentation accurately represents validation
- **WHEN** 前端规范或交付记录描述工具、可访问性目标、真实控制面验收或待改进项
- **THEN** 内容与实际配置和本轮结果对应，区分预览、替身测试及真实服务验收，不将建议写成已执行检查，不用历史或其他工程验收证明本轮结果

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

### Requirement: Consistent current Cellapp naming

项目 SHALL 在当前品牌页面、构建入口、默认新环境配置和当前图表使用 Cellapp/cellapp，MUST NOT 使用内部字母大写的驼峰写法；文档中的发布顺序和验收命令 MUST 与实际实现及验证规范一致。旧客户端兼容入口和历史记录 SHALL 保留且明确其用途。已有数据库升级 MUST 保留数据，开发身份改名 MUST 保留账号及资源关联，存在双身份冲突时 MUST 拒绝自动合并。

#### Scenario: New development environment
- **WHEN** 开发者按当前文档构建和配置新环境
- **THEN** 服务二进制与默认数据库使用 cellapp 命名，页面与图表展示 Cellapp，验收命令包含规定的 race 检查

#### Scenario: Existing development identity
- **WHEN** 原开发账号存在且没有冲突的新账号时登录
- **THEN** 账号改用当前名称并保留 ID、应用、凭证和浏览器会话关联，并发登录不会产生另一个账号

#### Scenario: Conflicting development identities
- **WHEN** 新旧开发身份对应两个不同账号
- **THEN** 登录明确失败，账号和资源不被自动合并或删除

#### Scenario: Legacy client and historical records
- **WHEN** 使用旧客户端入口或查看旧版本记录
- **THEN** 旧客户端兼容继续有效，历史记录保留当时的名称与验收事实
