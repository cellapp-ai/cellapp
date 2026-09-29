# Proposal

## Why

项目已使用 Cellapp 品牌，但服务端页面、模块名、构建产物、默认配置与图表仍混用旧名称；发布图和验收命令也与实现及规范不一致。

## What Changes

- 品牌显示名称为 Cellapp，允许 cellapp；禁止驼峰写法，规则记录于品牌规范并同步主仓库与公开客户端。

- 统一当前页面、User-Agent、npm/Go 模块、服务端产物、图表文件与标题、测试标识为 Cellapp/cellapp。
- 新数据库默认名和角色使用 cellapp；已有卷通过明确升级步骤处理，不自动修改运行中数据库。
- 开发登录使用 cellapp:development，将旧身份原位改名且保留 owner ID；双身份冲突时明确拒绝自动合并。
- 修正发布图为授权后构建，上传表述为逐文件上传，并统一浏览器验收的 race 开关。
- 保留旧客户端命令、配置、凭证兼容和历史归档中的旧名称。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `repository-workflow`: 当前命名与文档一致，明确历史与兼容例外及升级要求。

## Impact

修改主仓库服务端、配置、构建编排、测试和文档；同步公开客户端品牌文案，不改变客户端协议或 gitlink，不新增依赖或数据库结构迁移。现有未提交工作保留。用户“全部修改”授权本轮修正，默认保留此前明确标注的兼容和历史记录。
