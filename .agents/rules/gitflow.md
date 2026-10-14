# Gitflow 与发布规则

本文件适用于主仓库的开发、合并和 Cloud Run 发布。`main` 是开发集成分支；`deploy/*` 是发布分支。完成并验证的 Git 提交默认推送；Git 推送不等于 Cloud Run 或 npm 发布。

## 远端与默认推送

- 主仓库 `main` 和已完成的 `codex/*` 默认推送到 `https://github.com/cellapp-ai/cellapp.git`，使用 Git HTTPS 和 GitHub CLI 凭证。任务分支完成验证后推送，合入后再推送 `main`；不推送未完成工作。
- 主仓库 `deploy/*` 只推送到独立的私有仓库 `https://github.com/cellapp-ai/cellapp-deploy.git`，保留同名分支，例如本地 `deploy/main` 对应远端 `deploy/main`。不得将新的 `deploy/*` 提交推送到公开主仓库；既有公开分支另行清理，不以强推或删除代替迁移。
- skills 仓库使用 `https://github.com/cellapp-ai/skills.git`，完成客户端改动后直接推送其 `main`；不建立 skills 的 `deploy/*` 分支。推送主仓中更新的 gitlink 前，先验证 skills 目标提交可从 GitHub 获取。
- 推送前核对远端地址、分支、提交、工作区状态和快进关系；遇到分歧、权限或网络错误时保留本地提交并报告，不强推。服务兼容性要求延后公开 Skill 时，遵循 [发布规则](deploy.md) 的顺序。
- 首次使用某个检出时，将主仓 `origin` 配为上述 HTTPS 地址，增加名为 `deployment` 的独立远端；skills 的 `origin` 配为其 HTTPS 地址。首次推送新分支时设置对应远端的同名上游，后续普通 `git push` 必须仍指向本节规定的仓库。

## 开发分支与 worktree

1. 每项新工作默认从当前 `main` 创建 `codex/<任务名>` 分支和独立 worktree。先检查 `main`、skills 子模块与现有 worktree 的状态；有未提交工作时保护原样，不覆盖、不重置。
2. 开发、测试、规格归档和提交都在任务 worktree 完成。不得从 `deploy/*` 或其他任务分支派生新开发分支；发布分支上的问题也从 `main` 修复。
3. 完成并验证后，将任务分支合入本地 `main`。优先快进；如果 `main` 已前进，先在任务分支处理分歧并复验，再合入。不得为合入而强推、丢弃提交或覆盖其他 worktree 的改动。
4. 合入后核对 `main` 的提交、工作区状态和子模块引用，按默认推送规则同步远端；安全时归档已完成的 worktree。旧任务分支不需要重复合并。

## skills 子模块

涉及 skills 时，先从 skills 自己的 `main` 创建任务分支并提交，再合入 skills `main`，在主仓任务分支更新固定的 gitlink，最后合入主仓 `main`。默认推送 skills `main` 并验证提交可获取，然后推送主仓；与未上线服务不兼容的客户端变更按 [发布规则](deploy.md) 延后公开 Skill，主仓含新 gitlink 的推送也相应等待。npm 发布仍是独立步骤。

## 发布分支与 Cloud Run

1. `deploy/*` 只承接发布，不承担日常开发。新的发布分支从 `main` 创建；准备发布时，检查待发布的 `main` 提交及本地子模块完整性，再把本地 `main` 合入目标 `deploy/*`。不得反向用 `deploy/*` 作为功能开发基线。发布分支若有独有修复，先将该修复整理回 `main`，再继续发布。
2. 按上述远端映射默认推送已完成的 `main`、所需的 `codex/*`、skills `main` 和已验收的 `deploy/*`。`deploy/*` 推送前核对 gitlink 已能从 skills 远端获取；推送 Git 分支不触发 Cloud Run 流量切换或 npm 发布，后两项仍须用户明确要求。
3. 发布按 [发布规则](deploy.md) 先判定客户端发布需求、再部署服务，最后按需发布 Skill 和 npm。服务使用 Google Cloud CLI，仓库入口是 `python3 scripts/deploy-cloud-run.py --config infra/cloudrun.json`；具体配置、验收与回滚按 [运维说明](../../docs/operations.md) 执行。不要从任务 worktree 或未合入的开发分支直接发布。
4. 发布后记录 `main`、`deploy/*` 和 skills 的提交、推送状态、Cloud Run 修订版本及真实环境验收结果；失败时按运维说明恢复上一可用修订版本，不把本地测试等同于线上验收。
