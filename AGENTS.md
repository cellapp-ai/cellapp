# 项目协作与代码规范

## 品牌命名

显示名称使用 `Cellapp`，允许全小写 `cellapp`，不得使用驼峰写法；技术标识使用 `cellapp`，环境变量保持既有全大写约定。页面、CLI、Skill、文档、图表及可访问名称均遵循 [品牌规范](docs/standards/brand.md)。

## 架构与目录

Cellapp 提供静态应用托管。Go 单服务负责控制 API、浏览器授权页面和应用网关；PostgreSQL 保存控制元数据，私有 S3 兼容存储保存产物，本地开发默认使用 MinIO。构建在用户本机执行，服务端不运行上传的程序。应用可绑定所有者自带的 Supabase 项目，由浏览器直连，平台不代理查询。

```text
Agent --> Skill --> CLI --> Control API --> PostgreSQL
 |
 v
 Private Storage
Browser --> App Gateway --> Access Check --> Active Release
                         |
                         +--> Public data config --> Owner Supabase
```

```text
cellapp/
|-- AGENTS.md
|-- .gitmodules
|-- apps/web/              # Web 控制面，主仓库 npm workspace
|-- apps/server/
|   |-- cmd/server/             # 启动、组装、进程生命周期
|   |-- internal/hosting/       # 授权、部署、网关、存储及 Go 测试
|   +-- migrations/             # 数据库结构与迁移
|-- skills/                    # 独立 Git 子模块
|   |-- AGENTS.md
|   |-- packages/cli/           # 唯一 CLI 实现
|   |-- packages/shared/        # 客户端共享校验与类型
|   |-- skills/cellapp-deploy/   # Agent Skill
|   |-- contracts/              # HTTP 与本地状态契约
|   |-- tests/                  # 客户端单元测试
|   +-- scripts/                # 公开边界、Skill 与包检查
|-- scripts/                   # 主仓库开发命令编排
|-- tests/                     # 跨服务浏览器验收与编排测试
|-- infra/                     # 本地 PostgreSQL、MinIO 与 HTTPS 代理
|-- docs/                      # 运维、架构与验收记录
+-- openspec/                  # 主规格、变更设计与归档
```

公开 CLI/shared TypeScript 包放在 `skills/packages/<name>/`，纳入子仓库工作区；控制台前端应用放在主仓库 `apps/web/`，纳入根 npm workspace。主仓库维护控制台、服务端、开发编排和跨服务验收，不重复维护 CLI。子仓库规范见 `skills/AGENTS.md`；独立贡献者无需主仓库私有资料。

前端任务阅读 [前端规范入口](docs/standards/README.md)、[前端工程规范](docs/standards/frontend.md)、[前端代码规范](docs/standards/frontend-code.md) 和 [前端设计规范](docs/standards/design.md)。这些规范专门适用于 `apps/web`，不替代下述服务端与客户端规范。

## 扩展与依赖

- 先按职责扩展现有模块；独立生命周期、明确依赖边界或实际替换需求出现后再设计拆包，不预建空目录。
- `cmd/server` 只负责配置、组装和生命周期；`hosting` 内按授权、应用、部署、网关、存储、维护职责组织。
- CLI 入口处理参数和展示，部署流程、HTTP 请求、本地状态分别维护；shared 不依赖 CLI 入口或服务端实现。
- Skill 使用 CLI，不复制授权协议或部署算法。服务端通过 HTTP 契约与客户端协作，不依赖客户端源码。
- 使用现有 Storage/Identity 接口扩展适配器；新增依赖或接口需说明实际用途及维护成本。
- HTTP 字段、错误码、CLI 输出、本地状态和环境变量是兼容边界；变化同步契约、测试与说明。
- 已执行数据库结构通过增量迁移升级，不改写旧迁移代替升级；增加下一项迁移前先设计迁移执行器的版本跟踪。
- 目录和入口调整同步本文件、构建、安装及运行说明。

## 开发流程

设计 → 确认 → 开发 → 测试 → 归档 → commit。

1. **设计**：先读相关规格、代码、测试和工作区状态。明确目标、范围、模块/契约影响、验收场景、兼容性、失败恢复与验证方法。业务、架构和协议变化使用 OpenSpec；纯文档修正可用简短说明和事实校验完成流程。
2. **确认**：确认设计范围及验收标准后实施。明确授权持续有效，范围内细节、修复和验证自主推进；扩大产品行为、破坏兼容或改变已确认架构时先更新设计并确认受影响决策。
3. **开发**：按任务实施，保持改动聚焦。发现设计假设失效时先修正设计，不混入无关重构。
4. **测试**：执行与改动有关的成功、失败、权限和并发检查。通过、失败、跳过和未执行分别记录；测试替身不代表真实服务验收。
5. **归档**：必要验收满足、任务状态准确后同步主规格并归档。更新相关文档，检查路径、命令、名称和归档状态；不将历史验证当成本轮结果。
6. **commit**：审查并仅暂存本次文件，使用 feat/fix/refactor/test/docs/chore 等明确提交类型。跨仓库先提交子仓库，按已获授权推送并验证目标提交可获取，再提交主仓库最终 gitlink、归档和配套改动。未满足条件则保留工作并报告未完成项，不宣称完整交付。npm 发布单独处理。

- **归档后自动提交**：每次成功执行 `openspec-archive-change` 后，Agent 必须自动审查、暂存并 commit 该变更涉及的代码、规格、文档和归档文件，无需再次请求 commit 确认。跨仓库变更沿用上述提交顺序与远程可获取性要求，不夹带无关工作；没有可提交差异时说明已是干净状态，提交受阻时保留工作并报告原因。此规则授权自动 commit，推送仍按已有授权执行。

## 代码规范

- Go 使用 gofmt，TypeScript 保持 strict 与 ESM。命名表达业务，函数职责聚焦，注释解释原因与约束。
- 外部输入进行运行时校验；处理错误，不吞掉失败或失败后报告成功。
- SQL 参数化；版本、额度和状态竞争使用事务及必要约束/锁，发布保持原子性，重试保持幂等性。
- 外部请求设置超时，尊重取消；日志记录可定位的请求信息而不泄露秘密。
- 保持所有者/访客授权区分、来源隔离、资源逐次鉴权与私有存储边界。
- 凭证、分享密钥、Cookie 不进入项目状态、发布产物或日志。测试仅使用独立测试资源，不修改生产数据。

## 开发与验证入口

```sh
git submodule update --init --recursive
npm ci
npm --prefix skills ci
npm run cli -- --help
npm run typecheck
npm test
npm run build
go vet ./apps/server/...
```

服务端事务、权限、额度或并发修改执行隔离 PostgreSQL 集成和 race 检查；浏览器授权、Cookie、来源、资源路由或客户端联调变化执行浏览器验收：

```sh
TEST_DATABASE_URL=<isolated-test-dsn> go test -race -count=1 ./apps/server/...
RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=<isolated-test-dsn> go test -race -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting
```

客户端独立检查见子仓库 AGENTS。文档修改检查事实、链接、命令即可，不增加镜像实现的形式化测试。缺少必要环境时记录缺口，不以跳过代替通过。

## 默认使用 Worktree 开发

- 涉及文件修改的任务默认在独立 Git worktree 中执行；只读调查可在当前检出中进行。
- 开始前检查主仓库、skills 子仓库及已有 worktree 状态，优先复用适合当前任务且没有其他工作占用的 worktree。
- 没有合适 worktree 时，自动创建，无需再次确认。Codex 桌面端优先使用内置 worktree 工具。
- 修改前创建或切换到 `codex/<任务名>` 开发分支，后续编辑、安装依赖、测试和提交均在该 worktree 中完成。
- 初始化 skills 子模块并保持主仓库固定的提交；不覆盖原检出中的未提交工作。
- 任务完成后报告 worktree 路径、分支及两个仓库的提交/推送状态；通过内置归档工具清理不再需要的 worktree。

## 子模块与工作区保护

首次克隆使用 `git clone --recurse-submodules <parent-url>`；已有检出先初始化。日常执行 `git pull --recurse-submodules`，按主仓库固定提交更新。可主动运行仓库本地 `git config submodule.recurse true`，使普通 pull 更新已初始化子模块；新子模块仍需初始化。不要用 `update --remote` 隐式追随最新版本。

更新前检查两个仓库状态，保护未提交文件。子模块 detached HEAD 下先切开发分支再修改。禁止擅自 reset、覆盖工作、删除 Git 历史或无差别暂存。公开客户端不包含服务端、基础设施、私有运维资料和秘密。

交付说明结果、实际验证、未完成事项及两个仓库的提交/推送状态。
