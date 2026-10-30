# Spec Delta

## ADDED Requirements

### Requirement: Personal agent route isolation
首版个人模型连接与应用授权的管理接口 SHALL 仅在控制域提供，并按当前登录用户鉴权；应用模型调用 SHALL 仅在当前应用域的 `/_cellapp/*` 路径提供，按应用访客会话鉴权。设备配对、设备心跳、项目远程授权与远程任务接口在本变更中 MUST NOT 作为可用接口暴露；其后续授权与路由由 `add-agent-development-and-fork` 单独定义。现有 CLI Bearer、MCP 令牌、分享密钥和应用脚本 MUST NOT 跨接口取得其他身份的权限；未知路径 MUST 返回 API 错误，不得回退到静态页面。

#### Scenario: Wrong host or credential family
- **WHEN** 应用脚本访问个人连接管理接口，或设备凭证请求浏览器管理 API
- **THEN** 系统拒绝请求，不泄露个人连接、设备和任务数据

#### Scenario: First-release device route
- **WHEN** 首版客户端请求设备配对、心跳、项目远程授权或任务接口
- **THEN** 控制域返回可识别的不可用或未知接口错误，不创建可用设备身份或接受开发任务

#### Scenario: Unknown agent API route
- **WHEN** 请求与新接口相似但未声明的控制域或应用域路径
- **THEN** 返回可识别的 API 失败，不返回官网或应用的 HTML 入口
