# skill-deployment Spec Delta

## MODIFIED Requirements

### Requirement: Local static build
Skill 和 CLI SHALL 支持指定构建命令及输出目录，支持跳过构建的普通 HTML 目录；平台 MUST NOT 执行用户构建脚本或运行用户后端。Skill SHALL 明示浏览器 localStorage 等本地数据不跨设备同步。Skill SHALL 允许静态前端绑定所有者自带的外部 Supabase 项目，并 MUST 拒绝将 SSR、本机后端、需要服务端运行的产物或 service role 作为部署内容。

#### Scenario: Build static frontend
- **WHEN** 构建命令成功，输出目录包含有效 index.html 及静态资源
- **THEN** CLI 仅准备该目录的产物用于上传，不上传整个源码仓库

#### Scenario: Build fails or requires backend
- **WHEN** 构建失败，或指定输出被识别为需要服务端运行的产物
- **THEN** CLI 停止发布，报告错误或静态导出要求，现有线上版本不变

#### Scenario: Static frontend with external Supabase
- **WHEN** 所有者提供自有 Supabase URL 与 anon/public key，且产物仍是静态导出
- **THEN** Skill 使用 CLI 绑定该配置并继续或完成静态发布，不把外部数据库当作不可部署的用户后端，也不把密钥写入 `cellapp.json`
