# 发布规则：服务、Skill、npm

本规则在用户明确要求发布时执行；编写、合并代码或更新本文件都不触发发布。发布分支和 worktree 的来源遵循 [Gitflow 规则](gitflow.md)。顺序固定为：**发布前判定 → Cloud Run 服务 → GitHub Skill → npm 包**。前一阶段失败或验收未通过时停止后续阶段，并报告已发生的外部变更。

## 1. 发布前判定与准备

1. 先确定待发布的本地 `main` 和它固定的 skills 提交，检查两个仓库及现有发布 worktree 状态。记录主仓提交、skills 提交、上一可用 Cloud Run revision 与本次目标版本。不得从开发 worktree 直接发布。
2. 比较目标 skills 提交与上次公开的 `cellapp-ai/skills` 提交，检查 `skills/cellapp-deploy/`、公开契约、CLI/shared 源码及 npm 打包文件。再查询 npm 上的 `@cellapp/cli` 版本并检查本地打包内容，分别给出 **Skill 是否要发布**、**npm 是否要发布** 及依据；不能仅凭主仓 gitlink 变化就认定两者都要发布。首次发布时将尚不存在的公开版本视为需要发布。
3. Skill 从 GitHub 的 `cellapp-ai/skills` 仓库安装；npm 根包 `@cellapp/cli` 同时打包 CLI 和 `skills/cellapp-deploy`。Skill 内容有变化时，计划更新公开 GitHub Skill，且 npm 包内容也视为有变化；只有 CLI/shared 或打包契约变化时，计划发布 npm，并检查 Skill 文档是否需要同步。若两者均无变化，服务发布后跳过对应公开发布阶段。
4. 在服务部署前完成所有客户端代码、版本和验证准备：需要 npm 发布时，在 skills 开发分支更新公开根包版本及相关锁文件，确认版本未被 npm 使用，并依次合入 skills `main`、更新主仓 gitlink、合入主仓 `main`；运行 skills 仓库的 typecheck、test、build、Skill、边界与包检查，以及 `npm pack --dry-run --json`，核对包内 CLI 与 Skill。服务与旧 CLI/Skill 的兼容性、失败恢复和真实浏览器验收方案也须事先确认。准备不完整则停止，不先发布服务。
5. 当前 Cloud Run 构建上下文排除 `skills/`，因此服务阶段使用已固定的本地 skills 提交即可，公开 GitHub Skill 留到服务验收后发布。如果构建方式改为从远端获取 skills，先调整发布设计，不能悄悄颠倒顺序或构建不同的版本。

## 2. 先发布 Cloud Run 服务

准备完成后，把本地 `main` 合入目标 `deploy/*` 并确认检出干净、gitlink 指向已验证的 skills 提交。从该发布分支运行 `python3 scripts/deploy-cloud-run.py --config infra/cloudrun.json`。该仓库入口使用 Google Cloud CLI 完成预检、构建、迁移、候选 revision 检查及流量切换。记录镜像 digest、迁移 job、revision 和实际切流结果；按 [运维说明](../../docs/operations.md) 验收控制台登录、CLI 授权、已发布访客应用、数据 API、未知 Host 与未授权访问。仅在服务和所需的旧客户端兼容验收通过后，继续公开客户端。服务失败时停止发布 Skill/npm；按实际影响决定是否用 gcloud 将流量恢复到上一可用 revision，不回滚或删除数据库与对象。

## 3. 再发布 GitHub Skill

若 Skill 或 npm 包需要更新，从与主仓 gitlink 一致的 skills `main` 提交推送到 `cellapp-ai/skills` 的 `main`，核对远端 HEAD 是目标提交。Skill 内容变化时，还要从公开仓库检查 `cellapp-deploy` 的安装路径和内容；仅 npm 内容变化时，本阶段同步公开源码，但不把未变化的 Skill 记作新发布。不得推送未审查的 skills 工作区修改，不强推有分歧的远端。GitHub 发布失败时停止 npm 发布，保留已上线服务并报告部分发布状态。

## 4. 最后发布 npm

仅在判定包内容变化且服务、所需 Skill 阶段均通过后，从同一个已验证的 skills 提交发布 `@cellapp/cli`。发布前再次检查 npm 登录身份、目标版本不存在及打包清单；在 skills 仓库根目录执行 `npm publish --access public`。发布后查询 npm registry 的版本与包内容，在干净临时目录安装并验证 `cellapp --help`、兼容命令和打包的 Skill 文件。已发布的 npm 版本不可覆盖；失败后核对 registry 实际状态，再决定重试或准备新版本，不盲目重复发布。

## 5. 结果记录

记录每阶段的目标与实际提交、是否跳过及理由、Cloud Run 镜像/revision、Skill 远端提交、npm 版本、验证结果和未完成事项。只有所有计划中的阶段及真实环境验收通过，才报告完整发布成功。推送主仓前核对其 gitlink 已能从 skills 远端获取；当前仍不自动推送主仓或 `deploy/*`。
