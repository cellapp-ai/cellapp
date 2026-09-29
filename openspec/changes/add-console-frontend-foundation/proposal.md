# Proposal

## Why

Cellapp 当前没有独立控制台前端工程，也缺少明确适用于前端的工程、代码与设计规范。建立 `apps/console` 和前端规范入口，为后续控制台功能提供可构建、可验证且符合品牌的基础。

## What Changes

- 新增 `docs/standards/README.md`、`frontend.md`、`frontend-code.md` 和 `design.md`，分别提供前端规范入口、工程规范、代码规范和设计规范；明确适用范围为 `apps/console`，服务端规则继续由现有文档维护。
- 使用当前项目自身口吻维护规范，不记载外部参考工程的名称、路径、迁移来源或工作状态；区分实际实现、必须要求、建议及待改进项。
- 新增 React / TypeScript strict / Vite / Tailwind 控制台基础工程，包含可运行的单页框架、本地字体、品牌资产及响应式导航，不提供模拟业务数据或管理成功提示。
- 沿用 npm，纳入主仓库开发、类型检查、前端 lint、测试和构建入口；保留 CLI 唯一实现与子模块边界。
- 调整仓库规范中的 TypeScript 目录约束：独立 CLI/shared 包仍在 `skills/packages/`，控制台应用位于主仓库 `apps/console`。
- 本期仅交付前端工程基础；登录、应用管理、凭证管理、浏览器会话 API、Go 静态托管及 CSP 调整另行设计。

## Capabilities

### New Capabilities

- `console-frontend`: 独立控制台前端的启动、构建、品牌界面、响应式与可访问性基础。

### Modified Capabilities

- `repository-workflow`: 区分控制台与公开 CLI/shared 的源码位置，在主仓库验证和构建入口纳入控制台，并提供适用范围清楚的前端规范入口。

## Impact

- 新增 `apps/console/` 和 `docs/standards/`；修改根目录 `AGENTS.md`、`README.md`、`package.json`、`package-lock.json`、`scripts/project.mjs` 及相关编排测试。
- 引入前端所需 React、Vite、Tailwind、本地字体及检查工具；基础组件按实际使用引入，不复制未使用的组件库。
- 控制台作为根 npm workspace 管理，依赖由主仓库锁文件固定；`skills/` 保持独立依赖和发布边界。
- 不改变已有 HTTP、CLI、本地状态或数据库契约，不修改子仓库源码与 gitlink。实施时保护当前两仓库已有未提交工作。
