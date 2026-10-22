# Spec Delta

## ADDED Requirements

### Requirement: Agent-guided application API version selection
部署 Skill 和公开 Agent 指南 SHALL 说明应用运行时 API 使用一对一等价的 `major.minor` 与 `yymmdd` 别名、无 `v` 走最新版、默认稳定版只用于新构建，以及静态回滚如何恢复固定版本。Agent 构建使用官方 SDK 的应用时 SHALL 选择受支持契约并确认其 `major.minor` 别名已固化在静态产物中。

#### Scenario: Agent builds a new app
- **WHEN** Agent 使用官方 SDK 构建并发布新应用，项目未显式指定 API 版本
- **THEN** 构建采用平台当前默认稳定版，并报告最终固化的版本

#### Scenario: Agent investigates a versioned failure
- **WHEN** Agent 排查某应用的 API 错误
- **THEN** 指南引导它读取应用固化版本、错误码及公开 changelog，而不假设 SDK 包版本等于 API 版本

#### Scenario: Agent deploys an existing prebuilt app
- **WHEN** Agent 只发布既有预构建静态文件
- **THEN** 指南要求核对产物中的固定版本；缺失或不受支持时不得宣称 SDK 接口兼容已验证
