# skill-deployment Specification

## Purpose

让用户在 AI 对话中通过部署 Skill 完成本机项目构建与静态文件上传，得到可访问的应用地址，并在重复发布和失败时获得明确、可恢复的结果。

## Requirements

### Requirement: Local static build
Skill 和 CLI SHALL 支持指定构建命令及输出目录，支持跳过构建的普通 HTML 目录；平台 MUST NOT 执行用户构建脚本或运行用户后端。Skill SHALL 区分浏览器本地数据与通过 Cellapp 应用数据库持久化的数据，并说明只有获准的数据操作可跨设备使用。

#### Scenario: Build static frontend
- **WHEN** 构建命令成功，输出目录包含有效 index.html 及静态资源
- **THEN** CLI 仅准备该目录的产物用于上传，不上传整个源码仓库

#### Scenario: Build fails or requires backend
- **WHEN** 构建失败，或指定输出被识别为需要服务端运行的产物
- **THEN** CLI 停止发布，报告错误或静态导出要求，现有线上版本不变

### Requirement: Artifact validation
客户端和服务端 SHALL 拒绝目录越界、符号链接、保留平台路径、明显敏感文件以及超过已配置限制的产物，并校验上传文件与清单一致。

#### Scenario: Invalid or incomplete artifact
- **WHEN** 产物含越界路径、.env、凭证文件、超限内容、缺失文件或不匹配的摘要
- **THEN** 系统拒绝该产物发布并返回可定位的原因，不将部分文件作为新版本上线

### Requirement: Repeatable deployment result
CLI SHALL 保存不含秘密的应用关联配置，后续更新同一应用；系统 SHALL 支持幂等创建和部署，且仅在发布确认后报告成功。应用表变更的结果 MUST 与前端发布结果分别报告，不以其中一个成功掩盖另一个失败。

#### Scenario: Initial publish
- **WHEN** 新应用完整发布成功
- **THEN** CLI 返回固定 HTTPS 地址、部署标识及该应用初始访问密钥，并保存不含密钥的应用关联配置

#### Scenario: Update existing app
- **WHEN** 已关联项目重新部署成功
- **THEN** 系统更新该应用版本，保持地址与访问密钥及已有用户授权不变；CLI 不要求重新展示已存储密钥明文

#### Scenario: Retry after network interruption
- **WHEN** 客户端对相同清单使用相同幂等请求标识重试
- **THEN** 系统返回或继续同一操作，不重复创建应用或部署；同标识不同内容返回冲突

### Requirement: Skill-guided application data workflow
Skill 和 CLI SHALL 提供所有者授权的表变更、已应用版本查看、数据权限配置及验证入口，并指导 Agent 将前端接到应用专属数据地址和当前用户接口。Skill MUST 说明数据权限、公开配置与秘密凭证的区别，不能让前端携带数据库管理凭证。

#### Scenario: Build a data-backed static app
- **WHEN** 用户要求 Agent 为自有应用增加持久化表和前端读写
- **THEN** Skill 指导检查现有表版本、提交受限的版本化变更、设置最小权限、验证不同用户访问，再发布静态前端

#### Scenario: Migration or authorization fails
- **WHEN** 表变更或数据权限设置失败
- **THEN** CLI 返回具体失败及当前状态；Skill 不报告数据已可用，也不在失败后继续声称整项应用更新完成

#### Scenario: Cross-owner management attempt
- **WHEN** CLI 凭证所有者尝试操作别人的应用数据或访问权
- **THEN** 服务端拒绝，Skill 不绕过授权或请求数据库管理员凭证

### Requirement: Declared client compatibility
Skill SHALL 给出可运行的 `supabase-js` 表级 CRUD 用法和版本/操作支持范围，并明确平台登录、当前用户与应用访问权由 Cellapp 提供；未支持的 Supabase Auth、Realtime、Storage、RPC 和直连能力 MUST 标注为不可用。

#### Scenario: Agent selects a client API
- **WHEN** Agent 根据 Skill 为静态前端生成数据调用
- **THEN** 使用受支持的应用数据地址和公开配置，不假设完整 Supabase 产品协议可用

### Requirement: Recoverable initial application creation
The CLI SHALL preserve a stable creation request and its initial access key outside project files until the first release succeeds. A provisioning failure after application creation MUST leave a reusable application association and MUST NOT cause a retry to create another application.

#### Scenario: Provisioning fails after creation
- **WHEN** application creation commits but data provisioning reports failure
- **THEN** the CLI records the returned application ID, retains the key in private credential storage, reports failure, and a retry targets the same application

#### Scenario: Creation response is lost
- **WHEN** the application creation response is lost before the CLI receives it
- **THEN** the next attempt uses the original request ID and key, allowing the server to return the same application

#### Scenario: Older project lacks association
- **WHEN** the owner selects an application they own and supplies an output directory to link an unassociated project
- **THEN** the CLI writes a non-secret project association without creating another application

### Requirement: Daily CLI update notice

The CLI SHALL attempt to check the public npm registry for a newer stable `@cellapp/cli` release on its first invocation of each local calendar day. It SHALL notify on stderr only when the latest version is newer, without changing command stdout or exit status. The request MUST NOT include Cellapp credentials or project data. Registry and local state failures MUST NOT prevent the requested command.

#### Scenario: New release available

- **WHEN** the first CLI invocation of the day receives a valid newer stable version
- **THEN** the CLI prints the version and upgrade command to stderr and runs the requested command normally

#### Scenario: Same day or offline

- **WHEN** the CLI runs again on the same local day, or npm cannot be reached within the bounded timeout
- **THEN** it does not repeatedly query npm that day and continues the requested command without an update-check error

### Requirement: Agent-led CLI setup

The public deployment instructions and Skill SHALL guide an Agent to check the local CLI, install the published `@cellapp/cli` package when absent and verify it before deployment. Installation failure SHALL be reported with the actionable blocker rather than treated as deployment success.

#### Scenario: CLI is missing

- **WHEN** an Agent follows the guide in a project with supported Node.js and no `cellapp` command
- **THEN** it installs the CLI, verifies the command and continues deployment without asking the user to perform setup manually

#### Scenario: Setup cannot complete

- **WHEN** Node.js is unsupported or npm installation fails
- **THEN** the Agent stops before deploying and reports the specific environment issue

### Requirement: Public Agent deployment guide

The public website SHALL expose a login-free `https://cellapp.ai/deploy/AGENTS.md` Markdown guide that describes the current static deployment flow and its limits. It SHALL direct the Agent to use the CLI rather than implement the upload protocol itself.

#### Scenario: Agent reads instructions

- **WHEN** an unauthenticated Agent requests the guide
- **THEN** the service returns Markdown covering inspection, setup, browser authorization, publishing, verification and failure recovery
