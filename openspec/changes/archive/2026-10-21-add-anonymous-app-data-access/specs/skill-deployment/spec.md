# Spec Delta

## ADDED Requirements

### Requirement: Agent-managed anonymous application settings
CLI 和 Skill SHALL 允许已授权所有者查看并设置自有公开应用的匿名页面、读取及新增记录权限，并在版本化表定义中声明独立的匿名查询与新增策略。客户端 SHALL 展示运维总开关造成的有效写入状态，说明匿名 `update`/`delete` 不受支持，并避免把公开配置当作授权凭证。

#### Scenario: Deploy an anonymous form
- **WHEN** 用户要求 Agent 为公开应用发布无需登录的表单
- **THEN** Skill 指导先确认平台总开关、显式设置应用匿名页面和写入、仅向目标表赋予匿名新增权限，再以未登录浏览器验证提交及未授权读写的拒绝

#### Scenario: Operator gate or table policy missing
- **WHEN** 应用允许匿名写入，但运维总开关关闭或表没有匿名新增权限
- **THEN** CLI 和 Skill 不报告匿名提交已可用，并给出对应的未满足条件

#### Scenario: Existing app and manifest
- **WHEN** 用户升级旧应用或继续使用不含匿名字段的既有表定义
- **THEN** 默认行为仍需登录，匿名表读写维持拒绝，旧客户端的既有命令继续有效

#### Scenario: Cross-owner attempt
- **WHEN** CLI 凭证所有者尝试修改他人的应用匿名设置或表策略
- **THEN** 服务端拒绝，客户端不绕过所有者鉴权
