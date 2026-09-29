# 仓库协作规范验收

变更：`standardize-repository-workflow`。日期：2026-09-29。

## 实施结果

- 主仓库与公开客户端各自提供 AGENTS，明确架构、目录、扩展、代码规范及设计、确认、开发、测试、归档、commit 流程。
- TypeScript 包唯一维护于 `skills/packages/`；主仓库常用入口调用子仓库客户端，浏览器联调保留在主仓库。
- 已有客户端 Git 历史保留，主仓库通过 `.gitmodules` 和 mode 160000 gitlink 接入 `skills/`，子仓库 Git 目录由标准子模块机制管理。
- 旧部署 Skill 经比对确认说明已由 canonical Skill 覆盖；根重复 CLI/shared 及单元测试已移除，恢复副本保存在忽略的 `.local/repository-workflow-backup/`。

## 已通过

- 两个仓库各自 `npm ci --ignore-scripts --offline` 安装；主仓库 typecheck、build、test 与 Go vet。
- 客户端 11 项测试全部通过，包含旧状态/凭证迁移、产物校验、HTTP 契约及响应丢失重试；Skill、公开边界、包内容检查通过。
- 主仓库命令编排测试通过：未初始化提示、含空格参数透传、失败退出码及未知命令拒绝。真实主仓库 CLI 帮助入口通过。
- 客户端 tarball 在干净临时目录独立安装，cellapp/ohmyapp 两个入口及配套 Skill 内容验证通过，不依赖主仓库源码。
- 隔离 PostgreSQL schema 的 Go race 集成测试通过；单独启用真实 Chrome 的端到端验收通过，覆盖设备授权、发布、来源隔离、资源保护、重试、密钥重置、删除和退出。
- 临时本地仓库验证递归克隆、已有检出初始化、两个固定客户端版本间的递归 pull；子仓库存在更晚提交时仍使用主仓库指定版本。本地修改与目标版本冲突时更新失败且修改保留。
- 实际远程 main 为 `284711556c5ef17a2dfd9d0674c496e5b10b1f3d`，与当前主索引 gitlink 一致；临时父仓库使用实际 `.gitmodules` 和 gitlink，通过单次 HTTPS 传输覆盖完成真实客户端递归克隆并验证 HEAD。
- 两仓库差异空白检查及 OpenSpec strict 变更校验通过。

## 环境与交付状态

默认 GitHub SSH 22 端口连接超时；实际远程验收仅在临时命令中将同一仓库 SSH 地址映射为 HTTPS，未更改用户配置或 `.gitmodules`。通过 SSH 日常拉取仍需要可用网络和认证，HTTPS 的成功不能宣称 SSH 已通过。

常规测试默认跳过需要显式环境开关的测试；本轮已另外启用测试数据库和浏览器完成相关验收。没有重新执行真实七牛存储验收，本变更不修改存储适配器。

17 项实施任务全部完成，主规格已同步，并于 2026-09-29 归档至 `openspec/changes/archive/2026-09-29-standardize-repository-workflow/`。仍保留未提交工作，尚未 commit 或推送；最终客户端改动提交并可从远程获取后，主仓库再更新最终 gitlink。本记录中的远程验证针对当前已有提交，不能代替未来新增提交的可获取性检查。

## 重跑

```sh
git submodule update --init --recursive
npm ci
npm --prefix skills ci
npm run typecheck
npm test
npm run build
go vet ./apps/server/...
npm --prefix skills run check:skill
npm --prefix skills run check:boundary
npm --prefix skills run check:package
TEST_DATABASE_URL=<isolated-test-dsn> go test -race -count=1 ./apps/server/...
RUN_BROWSER_TESTS=1 TEST_DATABASE_URL=<isolated-test-dsn> go test -race -count=1 -run TestBrowserEndToEnd ./apps/server/internal/hosting
openspec validate --specs --strict
```

最后一项校验归档后的主规格。
