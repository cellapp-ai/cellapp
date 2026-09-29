# Proposal

## Why

主仓库缺少可执行的架构与协作规范，根目录和 `skills/packages/` 重复维护 TypeScript 客户端。`skills/` 已有独立 Git 历史和远程地址，但尚未登记为主仓库子模块，无法通过主仓库记录并复现客户端版本。

## What Changes

- 创建主仓库 `AGENTS.md` 和子仓库 `skills/AGENTS.md`，明确模块职责、目录扩展规则、代码规范和设计 → 确认 → 开发 → 测试 → 归档 → commit 流程。
- 将所有 TypeScript 包统一到 `skills/packages/`，复用已有公开客户端及其兼容迁移能力，移除根目录重复实现；客户端单元测试归子仓库，跨服务浏览器验收保留在主仓库。
- 将已有 `skills/` 仓库接入 Git submodule，主仓库以 gitlink 固定客户端提交，并提供递归克隆、初始化、拉取及更新版本说明。
- **BREAKING（仓库开发接口）**：根目录 `packages/` 和 `dist/packages/` 不再作为客户端源码及发布入口；CLI 包改由 `skills/` 安装和打包。保留根目录常用开发命令作为代理入口，更新测试引用、文档和安装路径。
- 明确跨仓库交付顺序：验证并归档后提交、推送子仓库，再提交主仓库的子模块版本；保护未提交文件，维护公开源代码边界。

## Capabilities

### New Capabilities

- `repository-workflow`: 可复现的主仓库与客户端子模块协作，包括源码归属、递归获取、开发入口、规范和跨仓库交付。

### Modified Capabilities

无。现有 `skill-deployment` 的本机构建、产物校验和可重试部署行为继续保持；本次不改变托管、访问或授权需求。

## Impact

- 主仓库：`AGENTS.md`、`.gitmodules`、gitlink、根目录 `packages/`、`package.json`/锁文件、`tsconfig.json`、`tests/`、README 和相关运行/验收说明。
- 子仓库：`AGENTS.md`、公开边界允许清单及相关测试/说明；根据差异比对补齐客户端回归覆盖，继续独立构建和发布。
- 保留 Go 服务端、HTTP 协议、数据库、存储配置及现有客户端命令/输出兼容性；不重命名 Go module，不新增托管能力，不执行 npm 发布。
- 实施需验证子仓库目标提交可由其远程获取；当前仅观察到本地历史和远程配置，未验证远程可达或是否已包含最新提交。
