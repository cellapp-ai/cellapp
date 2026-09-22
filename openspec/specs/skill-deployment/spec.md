# skill-deployment Specification

## Purpose

让用户在 AI 对话中通过部署 Skill 完成本机项目构建与静态文件上传，得到可访问的应用地址，并在重复发布和失败时获得明确、可恢复的结果。

## Requirements

### Requirement: Local static build
Skill 和 CLI SHALL 支持指定构建命令及输出目录，支持跳过构建的普通 HTML 目录；平台 MUST NOT 执行用户构建脚本或运行用户后端。Skill SHALL 明示浏览器本地数据不跨设备同步。

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
CLI SHALL 保存不含秘密的应用关联配置，后续更新同一应用；系统 SHALL 支持幂等创建和部署，且仅在发布确认后报告成功。

#### Scenario: Initial publish
- **WHEN** 新应用完整发布成功
- **THEN** CLI 返回固定 HTTPS 地址、部署标识及该应用初始分享密钥，并保存不含密钥的应用关联配置

#### Scenario: Update existing app
- **WHEN** 已关联项目重新部署成功
- **THEN** 系统更新该应用版本，保持地址与分享密钥不变；CLI 不要求重新展示已存储密钥明文

#### Scenario: Retry after network interruption
- **WHEN** 客户端对相同清单使用相同幂等请求标识重试
- **THEN** 系统返回或继续同一操作，不重复创建应用或部署；同标识不同内容返回冲突
