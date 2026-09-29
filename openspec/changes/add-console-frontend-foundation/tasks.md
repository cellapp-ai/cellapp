# Tasks

## 1. 前端规范与仓库边界

- [x] 1.1 新增 `docs/standards/README.md` 和 `frontend.md`，写明前端适用范围、规则状态、目录职责、npm workspace、开发和交付要求；通过逐项核对实际配置及相对链接验证，不将规划或未执行检查写成现状。
- [x] 1.2 新增 `docs/standards/frontend-code.md`，覆盖 strict 类型、输入校验、React 组件与副作用、CSS token、语义交互、错误恢复及资源边界；确认不创建通用 `code.md`，内容不混入服务端实现规则。
- [x] 1.3 新增 `docs/standards/design.md`，记录品牌资产、配色、字体、控制台布局、响应式、状态反馈及可访问性目标；核对颜色、资源路径和验收项，全文明确为前端设计规范。
- [x] 1.4 局部更新根 `AGENTS.md` 的目录与 TypeScript 包约束，补充前端规范阅读入口；检查公开 CLI/shared 仍位于 `skills/packages/`、服务端规则保留且链接均可解析。

## 2. 控制台工程基础

- [x] 2.1 新增私有 `@cellapp/console` workspace 及 React、Vite、Tailwind、本地字体和必要检查工具依赖，增量更新根锁文件；确认工具与 Node.js 22.12+ 基线兼容，根 `npm ci` 可复现安装且控制台无独立锁文件、`skills/` 未纳入 workspace。
- [x] 2.2 建立 `apps/console` HTML、React 入口、Vite、Tailwind 和独立应用/配置 TypeScript 环境，覆盖全部源码并启用 hooks lint；通过控制台 `typecheck` 和 `lint` 验证 DOM/bundler/JSX 与配置检查，不修改根 NodeNext 检查职责。
- [x] 2.3 配置控制台 `dev`、`typecheck`、`lint`、`test`、`build`、`preview` 命令及 loopback 默认监听；在无需 Go 或客户端子模块的条件下验证独立开发、构建和预览入口可用。

## 3. 品牌界面与基础交互

- [x] 3.1 添加必要 Cellapp 品牌原始资产、favicon 与本地字体，建立无格式冲突的语义颜色和间距 token；检查正式产物资源正常加载、Logo 比例正确且无外部字体请求。
- [x] 3.2 实现带主标题、正文、品牌、真实页内导航、跳过正文入口及移动菜单的基础页面；验证各链接到达真实内容、菜单状态准确且可关闭，不出现模拟业务数据、管理按钮或 API 请求。
- [x] 3.3 实现小屏布局、长内容换行、可见焦点、减少动态效果与无脚本说明；在 320、390、768、1440 px 以及实际断点两侧人工检查布局、键盘操作和触控范围。

## 4. 编排与浏览器验收

- [x] 4.1 增加根 `dev:console`、`preview:console`、`lint:console`，在 `scripts/project.mjs` 的完整类型检查、测试和构建流程纳入控制台；通过原 CLI 参数/退出码测试和新增编排场景验证转发、顺序、失败退出及子模块缺失提示兼容。
- [x] 4.2 添加正式构建产物的前端浏览器验收，测试脚本自动管理临时 loopback 预览生命周期；检查基础渲染、资源加载、导航、菜单、键盘焦点、规定宽度、axe、减少动态效果及无脚本说明，缺少 Chrome 时明确失败并给出配置方法。
- [x] 4.3 更新根 README 的前端安装、独立命令、产物位置和规范链接；逐项执行所列命令并核对文档，明确本期为前端基础而非已接入真实控制面。
- [x] 4.4 执行完整 `npm run typecheck`、`npm test`、`npm run build` 及本轮桌面/移动视觉验收；记录通过、失败、跳过与未执行结果，核对未混入秘密或外部参考工程标识、未改变原 Go/CLI 行为及子模块版本。

## 5. 规格与交付

- [x] 5.1 在全部必要验收满足后复核任务状态、运行说明和前端规范，与规格场景逐项对照；通过 `openspec validate add-console-frontend-foundation --strict` 验证规划格式并记录实际验收证据，缺少验收时保持相关任务未完成。
- [ ] 5.2 同步 `console-frontend` 和 `repository-workflow` 主规格并使用归档技能归档，再按归档后自动提交规则审查、仅暂存并提交本次文件；核对归档目录、主规格和提交差异，不夹带已有未提交工作，报告主仓库提交状态及子仓库未改动状态，推送遵循已有授权。

交付阻塞：主规格同步已完成；归档与提交尚未执行。控制台集成依赖当前未提交的仓库迁移（根编排脚本、根协作规范与子模块入口），这些前置文件在现有 HEAD 中不存在，不能夹带其他变更或将局部前端变更提交成不可复现版本。实现与本轮验收已记录于 `docs/console-frontend-verification.md`，前置迁移提交后继续本项。
