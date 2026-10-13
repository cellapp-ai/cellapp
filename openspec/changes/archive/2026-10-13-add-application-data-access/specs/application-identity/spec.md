# Spec Delta

## Purpose

让所有应用访客复用 Cellapp 平台登录，并向静态前端提供仅在当前应用内稳定的用户身份，使应用无需自建用户系统且不能据此关联用户在其他应用中的活动。

## ADDED Requirements

### Requirement: Platform login before application entry
系统 SHALL 要求访客先完成 Cellapp 平台登录，再决定其能否进入公开或受限应用。跨控制域与应用域的登录返回 MUST 仅指向发起登录的应用安全地址，不得把控制域登录凭证交给应用脚本。

#### Scenario: First visit and return
- **WHEN** 未登录用户访问应用深层页面并完成平台登录
- **THEN** 系统返回原应用中的安全路径，再按该应用访问规则决定是否进入

#### Scenario: Invalid return target
- **WHEN** 登录回调携带外部、其他应用或已过期的返回目标
- **THEN** 系统拒绝该目标，不向目标地址传递登录凭证

#### Scenario: Expired or logged-out identity
- **WHEN** 平台用户会话失效或用户退出登录
- **THEN** 后续应用页面与数据请求要求重新登录，不能继续依靠旧应用会话取得访问权

### Requirement: Application-scoped user identity
系统 SHALL 为同一平台用户在同一应用内提供稳定的应用用户 ID，并在不同应用使用不同 ID。应用接口 MUST NOT 暴露平台全局用户 ID、登录提供方主体标识、控制域 Cookie 或其他应用的用户 ID。

#### Scenario: Repeat visit to one application
- **WHEN** 同一用户在不同设备登录并访问同一应用
- **THEN** 应用获得相同的应用用户 ID，可据此访问属于该用户的记录

#### Scenario: Visit two applications
- **WHEN** 同一用户分别进入应用 A 和 B
- **THEN** 两个应用收到不同的用户 ID，无法从该字段直接关联该用户

### Requirement: Current user for static frontends
系统 SHALL 向已登录且获准进入应用的前端提供当前应用用户的最小身份与访问状态，并明确区分未登录、待批准、已拒绝和已获准；额外个人资料 MUST 另行取得用户授权后才可暴露。

#### Scenario: Read current user
- **WHEN** 获准用户从应用来源读取当前用户信息
- **THEN** 响应包含该应用范围的稳定用户 ID 和真实访问状态，不包含全局身份或私密凭证

#### Scenario: Unapproved user asks for identity
- **WHEN** 已登录但未获准进入受限应用的用户尝试读取应用用户资料
- **THEN** 不返回可供应用脚本使用的身份数据，仅提供平台访问申请或密钥兑换流程
