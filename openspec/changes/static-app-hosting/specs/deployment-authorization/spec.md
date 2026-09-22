# Spec Delta

## Purpose

让应用所有者在浏览器中确认来自本机部署工具的授权请求，随后通过可撤销的独立凭证管理自己的应用，避免将访问者分享权限和应用修改权限混用。

## ADDED Requirements

### Requirement: Browser approved device authorization
系统 SHALL 通过所有者登录和明确确认，将有期限的设备授权请求绑定到所有者；仅持有该请求设备秘密的客户端能领取一次部署凭证。

#### Scenario: First deployment authorization
- **WHEN** 未登录 CLI 发起授权，用户在浏览器登录并确认匹配的请求
- **THEN** CLI 按约定间隔轮询后领取绑定该所有者的部署凭证，并继续部署

#### Scenario: Rejected or expired request
- **WHEN** 授权被拒绝或已过期
- **THEN** 系统返回对应终止状态，不签发凭证，CLI 提示重新发起授权

#### Scenario: Code replay and polling abuse
- **WHEN** 调用者仅持有浏览器确认码、重复领取凭证或过于频繁地轮询
- **THEN** 系统分别拒绝非法领取、拒绝重放或返回减速指示，不产生额外凭证

### Requirement: Credential lifecycle and confidentiality
CLI SHALL 安全保存部署凭证并在有效期内复用；系统 SHALL 支持所有者撤销凭证及 CLI 退出登录，且 MUST NOT 将部署秘密写入项目文件、上传产物或日志。

#### Scenario: Reuse and revoke
- **WHEN** 用户使用有效凭证部署，然后通过浏览器撤销该凭证或在 CLI 退出登录
- **THEN** 首次部署无需重复授权，撤销后的管理请求被拒绝；CLI 退出同时移除本机凭证

#### Scenario: Expired credential
- **WHEN** CLI 使用已过期凭证
- **THEN** 系统拒绝管理操作，CLI 引导重新设备授权

### Requirement: Owner scoped management
系统 SHALL 对创建、列出、上传、发布、删除和密钥重置执行所有者授权；分享密钥 MUST NOT 授予管理权限。

#### Scenario: Cross owner operation
- **WHEN** 用户尝试更新、删除或重置另一所有者的应用，或用分享密钥调用管理接口
- **THEN** 请求被拒绝，应用状态和内容保持不变

#### Scenario: App listing
- **WHEN** 所有者请求应用列表
- **THEN** 系统仅返回该所有者的应用及其部署状态，不返回密钥明文
