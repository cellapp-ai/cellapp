# Spec Delta

## ADDED Requirements

### Requirement: Consistent current Cellapp naming

项目 SHALL 在当前品牌页面、构建入口、默认新环境配置和当前图表使用 Cellapp/cellapp，MUST NOT 使用内部字母大写的驼峰写法；文档中的发布顺序和验收命令 MUST 与实际实现及验证规范一致。旧客户端兼容入口和历史记录 SHALL 保留且明确其用途。已有数据库升级 MUST 保留数据，开发身份改名 MUST 保留账号及资源关联，存在双身份冲突时 MUST 拒绝自动合并。

#### Scenario: New development environment
- **WHEN** 开发者按当前文档构建和配置新环境
- **THEN** 服务二进制与默认数据库使用 cellapp 命名，页面与图表展示 Cellapp，验收命令包含规定的 race 检查

#### Scenario: Existing development identity
- **WHEN** 原开发账号存在且没有冲突的新账号时登录
- **THEN** 账号改用当前名称并保留 ID、应用、凭证和浏览器会话关联，并发登录不会产生另一个账号

#### Scenario: Conflicting development identities
- **WHEN** 新旧开发身份对应两个不同账号
- **THEN** 登录明确失败，账号和资源不被自动合并或删除

#### Scenario: Legacy client and historical records
- **WHEN** 使用旧客户端入口或查看旧版本记录
- **THEN** 旧客户端兼容继续有效，历史记录保留当时的名称与验收事实
