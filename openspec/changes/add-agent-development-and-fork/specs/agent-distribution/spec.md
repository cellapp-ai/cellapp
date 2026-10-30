# Spec Delta

## Purpose

让作者分享可核验的 Agent 定义与源码版本，让使用者在自己的设备 fork 并安装独立副本，同时保持双方的模型、项目、会话、数据和本机执行权限彼此隔离。

## ADDED Requirements

### Requirement: Immutable agent package
作者 SHALL 能发布带不可变版本、内容摘要、运行入口、依赖和所需能力声明的 Agent 包。系统 SHALL 校验文件大小、路径和明显敏感文件；MUST NOT 将作者密钥、设备凭证、会话、私有数据或本机状态打入包。发布失败不得影响旧版本。

#### Scenario: Publish valid version
- **WHEN** 作者发布合法包
- **THEN** 使用者可在 fork 前看到版本和能力声明，并校验取得的内容

#### Scenario: Sensitive or invalid package
- **WHEN** 包含凭证文件、越界路径、符号链接或超限内容
- **THEN** 发布失败，旧版本继续可用，违规内容不交付给他人

### Requirement: Independent fork and installation
使用者 SHALL 能将作者分享的版本 fork 为自己拥有的副本，并在自己已绑定的设备安装。副本 SHALL 保留来源及基准版本，但模型连接、项目路径、工具权限、会话和后续修改由使用者独立配置；作者不能读取或改变这些个人状态。

#### Scenario: Fork another user's agent
- **WHEN** 使用者 fork 并安装作者公开版本
- **THEN** 仅取得允许分发的内容，并需自行配置模型与本机权限，不继承作者的秘密或项目

#### Scenario: Install fails
- **WHEN** 设备离线、能力不足或安装失败
- **THEN** 原有副本和已安装版本保持可用，界面显示可恢复的失败状态

### Requirement: Explicit permission-aware updates
作者发布新版本或删除原件 MUST NOT 静默修改已安装副本。使用者 SHALL 能查看差异并主动选择更新；新增或扩大的模型、文件、命令与网络权限 SHALL 在自己的设备上重新确认。拒绝或失败时旧版继续可用。

#### Scenario: Update requests new access
- **WHEN** 新版本要求先前未批准的项目目录或命令权限
- **THEN** 更新暂停等待该设备确认，拒绝后旧版与个人配置不变

#### Scenario: Author removes source
- **WHEN** 作者更新或删除分享源
- **THEN** 使用者现有副本不被静默替换或删除
