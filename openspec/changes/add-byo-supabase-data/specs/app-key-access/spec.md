# app-key-access Spec Delta

## MODIFIED Requirements

### Requirement: Session-based resource protection
系统 MUST 在每次资源请求（HTML、脚本、图片、HEAD 以及平台 `GET`/`HEAD /_hosting/data`）先验证当前应用、会话未过期、密钥代次匹配且应用未删除，然后才提供内容或数据配置。

#### Scenario: Authorized HTML navigation
- **WHEN** 已授权浏览器请求应用内页面
- **THEN** 网关返回当前发布版本的对应 HTML，不要求再次输入密钥

#### Scenario: Unauthorized subresource
- **WHEN** 无有效会话的请求获取脚本、样式、图片或 `/_hosting/data`
- **THEN** 系统拒绝该响应，不把密钥输入页当作该资源内容，也不返回数据配置
