# 前端工程规范

适用范围：`apps/web` 的前端应用、配置、资源与前端验证。服务端和公开 CLI 的规则见 [规范入口](README.md) 中各自的协作文档。

## 应用边界

**现状**：Web 是所有者控制面，提供登录、真实应用列表/详情、密钥重置/删除、部署凭证撤销和设备授权。Go 同源提供静态产物与 `/api/console`，浏览器使用所有者会话，CLI Bearer 接口保持独立。它使用 React 18、TypeScript strict、Vite 和 Tailwind CSS 3.4；界面文案为英文，工程规范为中文。构建和发布继续在本机通过 Skill/CLI 完成。

必须让界面操作与真实能力一致，不伪造应用、额度、部署结果或登录状态。新增远程请求、页面路由、会话、上传、删除及控制域托管时，先定义需求、契约、授权边界、失败恢复及验收，再实施。构建期前端环境变量属于公开值；秘密、Cookie、部署凭证和分享密钥不得进入源码、静态产物或日志。

## 工程与依赖

控制台是主仓库的私有 npm workspace `@cellapp/web`，根 `npm ci` 安装其依赖，根 `package-lock.json` 固定版本。不得增加控制台独立锁文件或混用其他包管理器；构建不得临时安装依赖。新增依赖需说明实际用途及维护、体积成本，不因模板中存在某个库就加入全部组件。

Node.js 基线为 22.12+；工具升级须验证声明的 Node 与 peer dependency 约束。公开 CLI/shared 仍由独立 `skills/` 子仓库管理，其安装使用 `npm --prefix skills ci`，不纳入主 workspace，也不从前端复制 CLI 算法或导入服务端实现。

| 路径 | 职责 |
| --- | --- |
| `apps/web/index.html` | metadata、favicon、挂载入口及无脚本说明 |
| `apps/web/src/main.tsx` | 本地字体、样式载入与 React 挂载 |
| `apps/web/src/App.tsx` | 管理路由、应用框架和实际管理页面 |
| `apps/web/src/api.ts` | 浏览器 API 请求、外部数据校验及会话失效处理 |
| `apps/web/src/styles/` | 语义 token、界面布局与状态样式 |
| `apps/web/public/logos/` | 品牌原始 SVG 和必要导出资产 |
| `apps/web/tsconfig*.json` | 应用、工具配置及前端测试类型检查 |
| `apps/web/tests/` | 正式产物浏览器验收及预览生命周期管理 |
| `scripts/project.mjs` | 主仓库客户端、服务端、前端检查编排 |
| `apps/web/dist/` | 生成的静态产物，不手工修改 |

出现多个页面时再建立 `pages/`，出现实际复用时再抽取 hooks 或工具；不得预建空模块。页面框架、业务组件与通用 UI 原语按职责扩展，避免把业务数据放入通用基础组件目录。

## 开发、构建与预览

在主仓库根目录执行：

```sh
npm ci
npm run dev:web
npm run typecheck --workspace @cellapp/web
npm run lint:web
npm run build --workspace @cellapp/web
npm run preview:web
npm test --workspace @cellapp/web
```

开发默认 `http://127.0.0.1:5173`，预览默认 `http://127.0.0.1:4173`；占用时以终端实际地址为准。可透传 `-- --port <端口>`；开发与预览默认仅监听 loopback。

独立前端命令不要求客户端子模块初始化；界面测试使用明确标注的 API 夹具，未连接 Go 的独立预览显示服务不可用，不能证明登录与管理已经验收。真实业务使用 Go、隔离数据库和 HTTPS 控制域联调。完整仓库检查要求按根 README 安装客户端，并使用 `npm run typecheck`、`npm test`、`npm run build`。根 `npm run dev` 继续启动服务端。

`build` 先检查类型，再清理并生成 `apps/web/dist/`，包含 HTML、JS、CSS、字体及品牌资产。预览和验收使用整个目录，不只复制 `index.html`。Go 仅为明确的 UI 路由返回入口，未知 API、静态资源和路径返回 404。开发先构建 Web，再启动根 `npm run dev`；`WEB_ROOT` 默认 `apps/web/dist`。完整构建提供 `dist/cellapp-server` 与 `dist/web/`，生产显式指定静态目录并配套部署/回滚。详情深链、未登录和会话过期必须验证，不以 Vite 开发回退证明生产路由能力。浏览器契约见 [Web API](../web-api.md)。

## 验证与交付

| 变更 | 必要验证 |
| --- | --- |
| 仅前端规范和说明 | 事实、路径、相对链接、命令及状态表述 |
| 前端源码、配置、资源和依赖 | 前端 typecheck、lint、build |
| 前端界面、交互和构建输出 | 在正式产物上运行浏览器验收，并检查桌面和小屏视觉 |
| 根编排 | 缺少子模块提示、CLI 参数与退出码、前端失败传播、完整检查 |

前端测试自动构建并管理 loopback 预览进程，复用本机 Chrome，macOS 默认路径为 `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`；其他环境通过 `CHROME_PATH` 指定可执行文件。缺少浏览器会报错，不静默跳过或在测试中下载浏览器。报告与截图写入根 `test-results/web/`。

检查包括页面与资源加载、真实导航、移动菜单、键盘焦点、320 / 390 / 768 / 1440 px 与 800 / 801 px 布局、长内容、axe、减少动态效果和无脚本说明。axe 通过不等于完整无障碍认证；人工检查焦点顺序、缩放、视觉层级和控件可辨识度。

交付必须记录本轮通过、失败、跳过与未执行检查，不能引用历史验收当作本轮结果。文档、规范与实际实现保持一致；任务完成后按根项目流程同步规格、归档和提交，不夹带无关工作，推送遵循已获授权。

**待改进**：多浏览器兼容性和视觉快照差异门禁尚未建立；真实 GitHub 与对象存储验收按实际环境分别记录，不以身份/存储替身代替。
