## Purpose

为 Cellapp 提供独立控制台前端基础，使开发者能够启动、构建与预览符合品牌、响应式和可访问性要求的界面，并为后续真实控制台功能提供清楚的交付与权限接入边界。

## ADDED Requirements

### Requirement: Independently runnable frontend foundation

项目 SHALL 提供位于 `apps/console` 的前端应用及文档化的开发、检查、构建和预览入口；安装和构建结果 MUST 可由已提交的依赖声明及锁文件复现，构建产物不得包含秘密。

#### Scenario: Developer starts the console
- **WHEN** 开发者完成主仓库文档规定的依赖安装并执行控制台开发入口
- **THEN** 可以在 loopback 地址访问具有 Cellapp 品牌、明确标题和正文的基础界面，不要求公开客户端子模块初始化或后端运行

#### Scenario: Preview a production build
- **WHEN** 开发者构建控制台并通过文档化入口预览生成目录
- **THEN** 页面、脚本、样式、字体与品牌资产可以正常加载，字体不依赖运行时外部字体服务，失败构建返回非零退出码

### Requirement: Usable responsive and accessible shell

控制台基础界面 SHALL 提供真实目的地的导航、可见键盘焦点、跳过正文入口和明确的语义结构；在 320、390、768、1440 px 宽度下 MUST 保持主要内容可读且无页面整体横向溢出，非必要动效 MUST 尊重减少动态效果偏好。

#### Scenario: Keyboard navigation
- **WHEN** 用户仅使用键盘浏览基础界面和操作移动菜单
- **THEN** 可以到达主要导航与正文，焦点可见，移动菜单的展开状态与可访问名称准确且可关闭

#### Scenario: Narrow viewport and long content
- **WHEN** 用户在规定的小屏宽度查看页面，包括较长文字或链接
- **THEN** 内容能换行或在适用的局部容器内滚动，不通过隐藏主要内容消除整体溢出

#### Scenario: Reduced motion or unavailable scripts
- **WHEN** 用户启用减少动态效果或浏览器无法执行 JavaScript
- **THEN** 非必要动态效果停用，无法执行脚本时仍获得有意义的基础说明，不呈现虚假的可操作管理界面

### Requirement: Truthful foundation capability boundary

基础控制台 SHALL 准确表达当前可用能力；MUST NOT 展示伪造的应用、部署、额度或管理成功结果，也不得为尚未实现的管理行为索取或持久化部署凭证。其交付说明 MUST 区分独立前端预览与真实控制域接入。

#### Scenario: User opens the foundation page
- **WHEN** 用户打开本期基础页面
- **THEN** 页面操作仅指向实际存在的内容或交互，不请求应用管理接口，不显示已登录、已发布或已撤销等未发生的业务状态

#### Scenario: Developer evaluates control plane readiness
- **WHEN** 开发者阅读控制台运行与交付说明
- **THEN** 可以明确得知本期提供独立前端工程，真实登录、管理 API 和控制域托管仍需后续设计与验收
