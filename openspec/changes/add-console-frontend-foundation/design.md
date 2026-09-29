# Design

## Context

动机见 `proposal.md`。主仓库当前以 npm 管理开发编排，Node.js 基线为 22.12+，根类型检查采用 NodeNext，仅覆盖跨服务测试；`scripts/project.mjs` 将 CLI/shared 检查转发到独立 `skills/` 子仓库，再执行 Go 检查与构建。

已有 `repository-workflow` 和根 `AGENTS.md` 要求所有 TypeScript 包位于 `skills/packages/`，新增控制台需要显式调整该约束。两仓库存在未提交的协作流程改动，实施必须以当前工作区为基础进行局部修改。

Go 当前提供服务器渲染的首页、设备授权与凭证管理页。应用管理接口使用部署 Bearer 凭证，控制域 CSP 禁止脚本；因此前端工程的建立不能被描述为已经接入生产控制面。

## Goals / Non-Goals

**Goals:**

- 为前端建立独立的规范、编译环境与验证边界，避免把浏览器工程规则施加到 Go 服务端或公开 CLI。
- 提供开发者能启动、构建和预览的控制台基础，使品牌与无障碍约定可以通过真实界面验证。
- 在保持 CLI/shared 唯一实现及发布边界的同时，使根检查能发现控制台失败。

**Non-Goals:**

- 本次不将前端挂载到 Go 控制域，不修改 Cookie、授权流程、管理 API、CSP 或数据库。
- 不创建应用列表、额度或发布记录的假数据，不提供不可用的删除、重置密钥或授权按钮。
- 不引入支付、博客、内容管理、多主题、营销页面静态预渲染或单文件 HTML 交付机制。

## Decisions

### 1. 前端规范单独命名并明确边界

新增以下文档，全部使用 Cellapp 当前项目视角撰写：

| 路径 | 标题与职责 |
| --- | --- |
| `docs/standards/README.md` | 前端规范入口：适用范围、阅读顺序、规则状态及维护方式 |
| `docs/standards/frontend.md` | 前端工程规范：目录、依赖、命令、构建产物及验收 |
| `docs/standards/frontend-code.md` | 前端代码规范：类型、组件、状态、副作用、CSS、输入边界与错误处理 |
| `docs/standards/design.md` | 前端设计规范：品牌、颜色、字体、布局、交互及可访问性 |

所有文档开头明确适用于 `apps/console` 的浏览器源码、前端配置、资源及前端验证；服务端与 CLI 的规则继续引用各自现有规范。根 `AGENTS.md` 提供阅读链接并调整目录约束，不复制完整前端规则。

长期规范区分“现状”“必须 / 不得”“建议 / 优先”“待改进”。不写入外部参考工程的名称、路径、历史变更、已执行验收或发布状态。`design.md` 的文件名按已讨论方案保留，标题和范围明确为前端设计；代码规范必须使用 `frontend-code.md`，不创建 `code.md`。

备选方案是使用通用 `project.md` / `code.md` 并把规则放到根协作规范；其名称与范围容易混淆服务端及前端，因此采用显式前端入口。

### 2. 控制台归主仓库 npm workspace 管理

根 `package.json` 仅将 `apps/console` 纳入 workspace，应用包命名为私有的 `@cellapp/console`。根 `npm ci` 安装控制台依赖，根 `package-lock.json` 固定配套版本，控制台不再创建独立锁文件。`skills/` 不加入该 workspace，继续独立安装与发布。

沿用 React 18、TypeScript 5.9、Vite、Tailwind CSS 3.4 的前端基线；实施时确认具体依赖及 peer dependency 与 Node.js 22.12 基线兼容并写入锁文件。使用 React hooks 检查的 lint 工具；优先 Oxlint，根据实际规则配置验证覆盖。Radix / shadcn 基础组件只在实际交互需要时加入，不复制完整模板依赖和未使用 UI 文件。

备选方案是为控制台单独使用 pnpm 或独立 npm 锁文件。根 workspace 可以复用现有安装入口并减少重复依赖管理，适合当前一个控制台应用的规模。

### 3. 浏览器工程使用独立 TypeScript 配置

推荐初始目录：

```text
apps/console/
|-- package.json
|-- index.html
|-- vite.config.ts
|-- tsconfig.json
|-- tsconfig.app.json
|-- tsconfig.node.json
|-- public/logos/
|-- src/
|   |-- main.tsx
|   |-- App.tsx
|   |-- components/
|   +-- styles/
+-- tests/
```

页面增加时再创建 `pages/`；hook 和工具出现真实复用后再扩展目录。基础工程不预建 API 客户端或空模块。

浏览器配置使用 DOM、ESM、bundler 解析和 React JSX，保持 strict、未使用检查及 `noUncheckedIndexedAccess`；应用检查覆盖整个 `src/`，配置与前端测试也纳入各自适用的检查入口。`@/` 映射在 TypeScript 和 Vite 一致维护。根 NodeNext 配置保持跨服务测试职责。

备选方案是扩充根 NodeNext 配置检查所有浏览器代码，但会混合 Node 与浏览器环境，独立配置更便于维护边界。

### 4. 单页框架使用真实可用的基础交互

本期使用单一 `/` 页面，包含品牌、标题、简短控制台基础说明、跳过正文入口和响应式导航。导航指向实际存在的页内区域；移动菜单可展开、关闭并具有同步的辅助技术状态。不为未来页面生成假链接或禁用管理按钮。

界面先沿用简短英文文案，工程规范为中文；没有业务 API 请求，不引入浏览器凭证或本地存储状态。缺少 JavaScript 时显示有意义的说明，不要求整个未来控制台离线可操作。正式产物为常规静态 HTML、JS、CSS、字体及品牌资源，通过 Vite preview 验收；开发服务器与预览仅绑定 loopback，不部署到控制域。

单页不需要额外路由库、SSR、Parcel 或单文件打包。未来真实页面、深链刷新与会话接入在后续变更中设计，避免本期默许 SPA fallback 覆盖已有 API 路径。

### 5. 品牌与通用设计规则适配管理界面

使用 `apps/console/public/logos/` 中现有 Cellapp 品牌 SVG 和必要 PNG，保持原始比例、深浅背景适配及可访问名称；favicon 使用方形标。资源及其说明不携带参考工程信息。

定义明确命名的语义颜色 token，统一 CSS 与 Tailwind 引用，避免同名 token 混用 HEX 与 HSL：背景 `#090d18`、面板 `#111827`、主要文字 `#f5f7f2`、次级文字 `#9ba5b7`、边框 `#293142`、主强调 `#b6ff5c`、辅助强调 `#8199ff`。焦点和选中状态同时具备可感知语义，不能只使用颜色。

Space Grotesk 用于正文与标题，DM Mono 用于代码及短工程标签，字体通过本地依赖打包，不依赖运行时外部字体服务；Logo 使用矢量轮廓。新增中文内容时检查实际系统字体回退，不宣称 Latin 字体覆盖中文字形。

控制台标题采用适合管理界面的尺寸与行高，不套用营销 Hero 大标题、110 px 区块间距或倾斜终端装饰。保留克制圆角、4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 px 间距尺度、可见焦点、减少动态效果及内容优先布局。以 WCAG AA 为目标，新控件优先采用至少 44 px 触控范围，具体对比度在实际渲染后验证。

备选方案是原样复制页面 CSS 与样式补丁；这会继承覆盖冲突和营销布局，因此仅转写通用设计约定，建立控制台自身 token。

### 6. 根编排保留原入口并增加前端入口

控制台包提供 `dev`、`typecheck`、`lint`、`test`、`build`、`preview`。根新增 `dev:console`、`preview:console`、`lint:console`；原 `npm run dev` 仍启动 Go，原 `npm run cli -- ...` 参数与退出码行为保持兼容。

根 `typecheck`、`test`、`build` 通过现有 `scripts/project.mjs` 顺序纳入控制台命令：类型检查覆盖控制台；测试执行前端 lint 及基础浏览器验收；构建生成 `apps/console/dist/` 并保留现有 Go/CLI 产物位置。任何阶段失败均返回非零退出码。控制台 workspace 的独立命令不要求 `skills/` 已初始化；根完整检查保留现有子模块缺失提示。

浏览器测试复用主仓库已有 Playwright/本机 Chrome 能力，在临时 loopback 预览上检查正式构建；测试脚本负责启动和关闭预览，不依赖手工预启动。新增 axe 依赖用于前端无障碍自动检查。缺少浏览器时报可操作失败，不能静默跳过或从网络临时安装工具。

## Risks / Trade-offs

- [当前工作区已有广泛未提交改动] → 只在必要文件局部更新，不重建锁文件或规范来覆盖既有工作；提交仅包含本次已审查改动。
- [新前端包突破旧 TypeScript 目录约束] → 同步修改 `repository-workflow` 对应要求与根 `AGENTS.md`，准确区分控制台应用和公开 CLI/shared 包。
- [工具版本或 peer dependency 不兼容] → 在实施时检验本机 Node 基线、依赖声明及实际安装结果，锁定兼容组合；升级范围只覆盖本次所需工具。
- [品牌一致但控制台布局尚无业务页面验证] → 本期仅验收基础框架；后续页面在同一规范下补充实际加载、空、失败和恢复状态。
- [前端构建成功被误解为生产控制台已接入] → README 明确开发与静态预览用途，真实控制域接入必须独立设计和浏览器权限验收。
- [规范描述被误认为已执行检查] → 文档写清现状与验收目标，交付另记本轮通过、失败、跳过与未执行结果。

## Migration Plan

1. 在当前工作区增量添加前端规范和 workspace，保持子仓库及已有 API 行为。
2. 添加控制台基础界面、资源、检查与根编排，更新运行说明和规范入口。
3. 执行文档核对、干净安装复现、类型/lint/构建和正式产物浏览器验收，检查原 CLI 与编排兼容。
4. 验收满足后按项目流程同步规格、归档并提交；本期不发布站点、不推送未获授权的提交。

回退时仅撤销本次前端新增文件与相关局部配置，不改数据库或子模块提交。根锁文件需要与 workspace 修改一起回退，保留本次之前已有的工作。
