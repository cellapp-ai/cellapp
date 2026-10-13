# Spec Delta

## MODIFIED Requirements

### Requirement: Delete and reclaim
系统 SHALL 允许所有者删除应用，立即阻止新的内容、数据访问和发布，并最终回收其对象、应用数据及用量；跨数据库与对象存储的清理 MUST 可重试，失败不得重新开放已删除应用。被删除应用的地址标识 MUST NOT 分配给其他应用。

#### Scenario: Delete live app
- **WHEN** 所有者删除已有应用
- **THEN** 现有会话不能再读取新页面、资源或数据响应，部署和表变更请求被拒绝，后台清理可重试并释放实际占用额度

#### Scenario: Abandoned deployment cleanup
- **WHEN** 未完成上传或未引用版本超过配置保留期限
- **THEN** 系统清理对应对象和预留用量，不删除当前有效版本或仍在使用的应用数据

#### Scenario: Data cleanup failure
- **WHEN** 删除应用后独立应用数据库暂时不可用
- **THEN** 应用仍保持不可访问，清理任务保留可重试状态，恢复后回收该应用表和数据且不影响其他应用
