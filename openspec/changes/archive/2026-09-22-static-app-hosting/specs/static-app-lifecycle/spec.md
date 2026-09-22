# Spec Delta

## Purpose

管理免费静态应用从创建、完整发布、持续更新到删除的生命周期，保持访问地址稳定，并以明确的额度和错误行为控制存储、上传和分发资源使用。

## ADDED Requirements

### Requirement: Atomic releases at a stable address
系统 SHALL 为应用提供稳定 HTTPS 地址，仅将完整且验证通过的部署发布为当前版本；失败和过期的并发发布 MUST NOT 替换已生效版本。

#### Scenario: Failed update
- **WHEN** 新版本上传中断、校验失败或发布失败
- **THEN** 原版本继续服务，未发布对象不对访客开放

#### Scenario: Concurrent updates
- **WHEN** 两个更新基于相同旧版本发布且其中一个先成功
- **THEN** 后一个返回版本冲突，不静默覆盖先发布成功的版本

### Requirement: Static routing
系统 SHALL 按正确内容类型提供文件，并支持显式启用的 SPA 页面回退；缺失静态资源 MUST NOT 被伪装为成功 HTML 响应。

#### Scenario: SPA deep link
- **WHEN** 已授权访客以 HTML 导航请求启用 SPA 的应用内不存在文件的页面路径
- **THEN** 系统返回当前版本 index.html；未启用 SPA 的同类请求返回 404

#### Scenario: Missing asset
- **WHEN** 已授权访客请求不存在的脚本或图片资源
- **THEN** 系统返回 404，不返回 SPA 入口

### Requirement: Free bounded hosting
系统 SHALL 免费提供本期托管，并执行可配置的应用数、文件数、单文件大小、部署大小、总存储及流量限制；并发请求 MUST NOT 绕过额度，超限 MUST NOT 触发自动收费。

#### Scenario: Upload or creation quota exceeded
- **WHEN** 创建或上传超过可用额度，包括并发预留用量
- **THEN** 系统拒绝操作并返回具体限制与恢复办法，已发布应用保持不变

#### Scenario: Traffic exhausted
- **WHEN** 应用达到当前计费无关的流量统计周期上限
- **THEN** 内容分发返回 429 和额度恢复时间，所有者仍能访问控制面，下一周期恢复可用额度

### Requirement: Delete and reclaim
系统 SHALL 允许所有者删除应用，立即阻止新的内容访问和发布，并最终回收其对象及用量；被删除应用的地址标识 MUST NOT 分配给其他应用。

#### Scenario: Delete live app
- **WHEN** 所有者删除已有应用
- **THEN** 现有会话不能再读取新响应，部署请求被拒绝，后台清理可重试且释放实际占用额度

#### Scenario: Abandoned deployment cleanup
- **WHEN** 未完成上传或未引用版本超过配置保留期限
- **THEN** 系统清理对应对象和预留用量，不删除当前有效版本
