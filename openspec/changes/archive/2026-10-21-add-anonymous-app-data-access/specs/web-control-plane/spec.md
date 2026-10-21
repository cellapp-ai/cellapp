# Spec Delta

## ADDED Requirements

### Requirement: Owner controls for anonymous application access
Web SHALL 向已登录所有者展示自有应用的匿名页面、数据读取和新增记录设置，以及平台匿名写入总开关造成的有效状态。所有者 SHALL 可分别更改应用设置；扩大匿名访问前 SHALL 明示影响并要求确认。写入必须使用所有者会话、逐应用鉴权和同源检查，界面只在服务端确认后报告成功。

#### Scenario: Owner enables anonymous form
- **WHEN** 所有者将自有公开应用的匿名页面及新增记录打开并确认，而运维总开关允许写入
- **THEN** 界面展示设置和有效状态，同时提示目标表仍需显式匿名新增权限

#### Scenario: Operator gate is off
- **WHEN** 应用写入设置为开，但运维关闭匿名提交总开关
- **THEN** 界面明确显示“应用已允许、平台当前禁止”的有效状态，不把应用设置误报为可提交

#### Scenario: Invalid or uncertain change
- **WHEN** 非所有者、跨来源请求或网络中断时尝试修改设置
- **THEN** 服务端拒绝未授权请求；结果不确定时界面不报告成功，并允许重新读取实际设置

#### Scenario: Restricted application
- **WHEN** 所有者查看受限应用或将允许匿名访问的公开应用改为受限
- **THEN** 匿名设置不可在受限状态下开启；切换后匿名访问立即停止，界面显示当前有效状态
