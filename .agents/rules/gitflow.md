# Gitflow 与发布规则

本文件适用于主仓库的开发、合并和 Cloud Run 发布。`main` 是开发集成分支；`deploy/*` 是发布分支。日常开发和合并阶段不推送任何分支。

## 开发分支与 worktree

1. 每项新工作默认从当前 `main` 创建 `codex/<任务名>` 分支和独立 worktree。先检查 `main`、skills 子模块与现有 worktree 的状态；有未提交工作时保护原样，不覆盖、不重置。
2. 开发、测试、规格归档和提交都在任务 worktree 完成。不得从 `deploy/*` 或其他任务分支派生新开发分支；发布分支上的问题也从 `main` 修复。
3. 完成并验证后，将任务分支合入本地 `main`。优先快进；如果 `main` 已前进，先在任务分支处理分歧并复验，再合入。不得为合入而强推、丢弃提交或覆盖其他 worktree 的改动。
4. 合入后核对 `main` 的提交、工作区状态和子模块引用；安全时归档已完成的 worktree。旧任务分支不需要重复合并。

## skills 子模块

涉及 skills 时，先从 skills 自己的 `main` 创建任务分支并提交，再在主仓任务分支更新固定的 gitlink，最后合入主仓 `main`。本地开发和本地合并可以保留尚未推送的子模块提交，但须明确记录远端克隆暂时无法获取该提交。推送主仓提交前须按授权推送 skills，并验证目标提交可从远端获取；服务发布与客户端发布的顺序见 [发布规则](deploy.md)。

## 发布分支与 Cloud Run

1. `deploy/*` 只承接发布，不承担日常开发。新的发布分支从 `main` 创建；准备发布时，检查待发布的 `main` 提交及本地子模块完整性，再把本地 `main` 合入目标 `deploy/*`。不得反向用 `deploy/*` 作为功能开发基线。发布分支若有独有修复，先将该修复整理回 `main`，再继续发布。
2. 当前阶段不因开发或合并而推送 `main`、`codex/*`、`deploy/*` 或 skills 分支。只有用户明确要求推送，或明确发起包含公开 Skill/npm 的发布时，才按 [发布规则](deploy.md) 推送所需的 skills 提交；主仓和 `deploy/*` 仍须单独授权推送。不能因代码合并而自动发布。
3. 发布按 [发布规则](deploy.md) 先判定客户端发布需求、再部署服务，最后按需发布 Skill 和 npm。服务使用 Google Cloud CLI，仓库入口是 `python3 scripts/deploy-cloud-run.py --config infra/cloudrun.json`；具体配置、验收与回滚按 [运维说明](../../docs/operations.md) 执行。不要从任务 worktree 或未合入的开发分支直接发布。
4. 发布后记录 `main`、`deploy/*` 和 skills 的提交、推送状态、Cloud Run 修订版本及真实环境验收结果；失败时按运维说明恢复上一可用修订版本，不把本地测试等同于线上验收。
