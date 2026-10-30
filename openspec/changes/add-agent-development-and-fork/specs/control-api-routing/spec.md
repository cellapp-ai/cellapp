# Spec Delta

## ADDED Requirements

### Requirement: Remote Agent route and credential isolation
设备绑定、心跳、项目授权、开发任务与 Agent 包管理 SHALL 仅在控制域的明确接口开放。浏览器操作 SHALL 使用所有者会话与逐资源鉴权，设备操作 SHALL 使用独立、可撤销的设备凭证；部署 CLI Bearer、MCP 令牌、分享密钥和应用会话 MUST NOT 代替设备凭证。应用域及未知 API 路径 MUST NOT 回退到静态页面。

#### Scenario: Wrong host or credential family
- **WHEN** 应用脚本或部署凭证请求设备任务与包接口，或设备凭证请求浏览器管理接口
- **THEN** 系统拒绝并不泄露其他所有者的设备、项目、任务或副本数据

#### Scenario: Revoked device credential
- **WHEN** 已撤销设备继续发送心跳、领取任务或安装 Agent 包
- **THEN** 请求失败且不产生新的有效执行或安装

#### Scenario: Unknown Agent route
- **WHEN** 客户端请求未声明的设备或任务 API 路径
- **THEN** 返回明确 API 失败，不返回控制台或应用静态 HTML
