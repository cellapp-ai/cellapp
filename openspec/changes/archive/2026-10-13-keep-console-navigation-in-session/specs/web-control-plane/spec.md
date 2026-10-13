## ADDED Requirements

### Requirement: Navigation within an authenticated console session

Web SHALL 在已经确认的所有者会话内切换管理页面时保留当前页面框架，不因每次普通内部链接点击重新加载文档或重新检查会话。内部目的地 MUST 保持真实链接和可直接访问的 URL；会话过期时仍按现有授权规则清除私有状态并进入登录。

#### Scenario: Move between management pages
- **WHEN** 已登录用户点击侧栏、应用列表或详情返回链接
- **THEN** 地址栏和当前页面更新，但现有会话检查不重复，且焦点进入新页面正文

#### Scenario: Browser history and direct entry
- **WHEN** 用户使用前进后退、刷新或直接打开有效管理深链
- **THEN** 显示与 URL 对应的页面，刷新和直接进入只执行一次初始会话检查；新标签页及修饰键点击保留浏览器原生行为

#### Scenario: Temporary state and session expiry
- **WHEN** 用户离开应用详情、切换到其他应用，或业务请求确认会话失效
- **THEN** 本次临时访问密钥及确认状态不带到另一页面；会话失效仍返回登录入口

#### Scenario: Initial session check
- **WHEN** 浏览器首次加载管理页面，尚未确认会话状态
- **THEN** 页面显示中性加载提示；只有确认无有效会话时才显示登录入口
