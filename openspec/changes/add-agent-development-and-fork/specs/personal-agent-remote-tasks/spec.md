# Spec Delta

## Purpose

让用户从其他设备安全地调用自己电脑上已授权的成熟编码 Agent 完成通用开发任务，同时保护本机项目、未提交工作和当前发布，并明确断线与撤销后的任务结果。

## ADDED Requirements

### Requirement: Owner-bound local device
系统 SHALL 让用户经控制域浏览器确认后绑定本机客户端，设备凭证 MUST 与部署 CLI、MCP 及应用会话隔离。设备在线状态 SHALL 来自近期真实心跳，用户 SHALL 能撤销设备；撤销后不得领取新任务。

#### Scenario: Pair and revoke
- **WHEN** 用户批准本机设备后又撤销它
- **THEN** 设备仅在批准后进入本账户，撤销后心跳与任务接口失效，其他账户不能接管

### Requirement: Explicit project and executor authorization
用户 SHALL 在设备上选择项目与受支持的成熟编码 CLI，并明确授予文件、命令、构建和发布范围。服务端 MUST NOT 保存本机绝对路径；扩大权限 SHALL 在该设备上重新确认。仅选择 `cellapp set agent` 首选项不构成任务授权。

#### Scenario: Register and expand project
- **WHEN** 用户登记项目后请求扩大写入或发布权限
- **THEN** 新权限在设备本地确认前不可用，服务端仅持有不透明项目 ID 和权限摘要

#### Scenario: Unsupported executor
- **WHEN** 所选 Agent 没有可验证的无界面执行入口或未登录
- **THEN** 系统明确拒绝该执行器，不改用其他 Agent 或静默安装

### Requirement: Remote development task lifecycle
已授权用户 SHALL 能从其他设备提交、观察和取消开发任务；在线设备主动领取，离线设备的新任务 SHALL 直接拒绝。任务状态、序号和结果 SHALL 可恢复，重复投递 MUST NOT 导致同一有效任务重复执行。

#### Scenario: Offline submission
- **WHEN** 用户对离线或已撤销设备提交开发任务
- **THEN** 平台拒绝，不留下会在设备重连后自动执行的排队任务

#### Scenario: Reconnect after interruption
- **WHEN** 设备领取任务后断线并重连，或同一消息被重复投递
- **THEN** 用户看见可解释的最新状态与结果，至多一次有效执行，不把不确定结果报告为成功

### Requirement: Isolated local work and bounded output
任务 SHALL 在独立工作目录运行并记录本地任务日志、源码快照与使用的授权范围。MUST 保留原项目检出中的未提交内容；并发任务、取消和重试 MUST NOT 覆盖原工作或越过项目授权。日志和结果不得包含凭证。

#### Scenario: Dirty project and concurrent tasks
- **WHEN** 原检出已有未提交修改且两个任务同时请求同一项目
- **THEN** 任务在受控独立目录执行或明确拒绝冲突，原检出及秘密保持不变

### Requirement: Private preview and confirmed publication
任务成功构建后，系统 SHALL 提供有期限、需授权的私有静态预览；预览不得取得正式应用 Cookie 或更改正式发布。发布 SHALL 由用户确认预览摘要及基准发布版本后进行；冲突、构建失败、预览过期或设备撤销不得替换现行发布。

#### Scenario: Review and publish
- **WHEN** 用户查看预览并确认发布，且基准版本仍相同
- **THEN** 复用原子静态发布流程；只有完整成功后报告新发布

#### Scenario: Conflict or failed build
- **WHEN** 构建失败、预览过期或确认时基准版本已变化
- **THEN** 当前发布保持不变，界面说明需要重新构建或审阅
