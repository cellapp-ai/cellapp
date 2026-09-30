# Tasks

## 1. 保护现状与客户端归并

- [x] 1.1 记录两个仓库的分支、HEAD、远程和未提交状态，保存旧 `skills/deploy-static-app` 内容；验证已有子仓库历史和文件可恢复，不夹带无关工作。
- [x] 1.2 比对根 CLI/shared、单元测试与子仓库实现，补入未覆盖的行为回归；验证子仓库测试覆盖旧命令、项目/凭证迁移、产物校验与重试，确认删除清单。
- [x] 1.3 合并旧部署 Skill 的必要说明到 `skills/skills/cellapp-deploy/SKILL.md`，移除已确认重复的旧路径；验证 canonical Skill 检查通过且原内容未遗漏。

## 2. 两级 AGENTS 与公开边界

- [x] 2.1 创建主仓库 `AGENTS.md`，写明真实架构与目标目录、扩展及依赖规则、Go/TypeScript 规范、契约与秘密保护、完整流程和跨仓库交付；通过逐项审阅确认命令、路径和授权边界准确。
- [x] 2.2 创建 `skills/AGENTS.md`，覆盖独立客户端职责、兼容策略、代码规范、验证/打包和独立贡献流程；验证单独阅读即可理解，不依赖私有文档或泄露主仓库实现。
- [x] 2.3 在 `skills/public-boundary.json` 精确允许客户端 AGENTS，更新必要边界说明；验证 `npm --prefix skills run check:boundary` 和相关现有测试通过，其他禁止路径仍被拒绝。

## 3. 主仓库入口与测试迁移

- [x] 3.1 移除根目录已核对的重复 `packages/` 和客户端单元测试，保留子仓库完整覆盖；验证根目录不再存在 TypeScript 包，客户端测试通过。
- [x] 3.2 调整根 package.json、锁文件及编排入口，取消旧 workspaces/bin/CLI 打包清单，代理 CLI、typecheck、build 和 test，保留 Go 及浏览器依赖；验证独立安装、参数透传和子任务失败码传播。
- [x] 3.3 更新根 tsconfig 与 `tests/browser.e2e.ts` 的客户端引用、命令路径及 `cellapp.json` 读取，检查 Go 浏览器测试定位；验证主仓库类型检查和构建成功，联调使用唯一客户端。
- [x] 3.4 为编排入口缺少子模块、参数透传和失败传播添加必要验证；在临时环境确认缺少初始化时非零退出并给出初始化方法，不执行隐式仓库修改。

## 4. 已有仓库接入子模块及文档

- [x] 4.1 保留子仓库 origin 与 Git 历史，将主仓库旧 Skill 跟踪改为 `.gitmodules` 和 `skills` gitlink，按标准子模块布局处理 Git 目录；通过主索引 mode `160000`、`git submodule status`、子仓库历史和文件检查验证接入正确。
- [x] 4.2 更新根 README 的递归克隆、已有检出初始化、递归 pull、可选本地配置、两个仓库安装与 CLI 打包说明；核对文档命令、固定版本语义和 SSH 读取条件。
- [x] 4.3 更新相关架构源资料、运维/验证记录与子仓库贡献说明，修正旧变更归档状态并区分历史和本轮验收；搜索旧源码、产物、Skill 路径，确认剩余引用仅用于明确兼容或历史记录。

## 5. 综合验收与交付准备

- [x] 5.1 完成子仓库 typecheck、test、build、check:skill、check:boundary、check:package，并在干净目录验证打包安装后的 CLI；记录实际结果，确认客户端无需主仓库源码。
- [x] 5.2 完成主仓库 typecheck、test、build、Go vet，以及隔离 PostgreSQL 的 race 集成检查和浏览器端到端验收；记录成功、失败、跳过及环境缺口，确认协议和旧本地状态兼容。
- [x] 5.3 在临时检出验证递归克隆、已有检出初始化和两个 gitlink 版本间的递归 pull，检查未提交文件不会被丢弃；本地 fixture 与实际远程可获取性分别记录，验证检出 HEAD 等于主仓库固定版本。
- [x] 5.4 审查两仓库差异、暂存范围和文档一致性，执行 `openspec validate standardize-repository-workflow --strict`；确认所有必要验收满足，形成可用于规格同步、归档和提交的验证记录。

所有实施任务通过后，再调用规格同步与归档流程。终态交付按以下顺序进行：归档完成 → 子仓库 commit → 按授权推送并验证远程可获取 → 主仓库 commit（包含归档和最终 gitlink）。此收尾顺序不作为归档前的任务勾选，避免让归档依赖归档后的提交；任一环节未完成均明确报告实际状态，不宣称完整交付。
