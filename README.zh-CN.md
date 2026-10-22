# Cellapp，个人 AI 应用工作台

[English](README.md) | [简体中文](README.zh-CN.md)

和 Agent 一起，把想法做成应用，并发布、管理、分享。

Agent 在你的电脑上完成构建，通过部署 Skill 发布到 Cellapp；你也可以直接使用 `cellapp` CLI。发布后，应用拥有固定的 HTTPS 地址，可以持续更新、设置访问权限，并按需使用独立的应用数据表。Cellapp 负责交付应用、身份验证和数据访问，不运行上传的程序。

## 快速开始

### 使用 Agent（推荐）

把这句话发给正在开发项目的编码 Agent：

> 使用 `npx skills add cellapp-ai/skills` 安装 Cellapp 部署 Skill，然后按 Skill 指引在本机构建并发布这个项目。需要共享数据时使用应用数据表，最后把发布后的网址发给我。

Agent 会安装 Skill、按需获取官方 CLI，并完成构建和发布。首次使用时，你只需在浏览器中批准 Cellapp 授权。如果 Agent 不支持 Skill，可以改为让它阅读[公开的 Agent 部署指引](https://cellapp.ai/deploy/AGENTS.zh-CN.md)。

### 手动使用 CLI

如果想自己在终端操作，请全局安装 CLI 并验证。需要 Node.js 22.12 或更高版本及 npm。

```sh
npm install -g @cellapp/cli@latest
cellapp --help
```

使用 `cellapp login` 连接平台，再用 `cellapp deploy --help` 查看发布参数。构建产物和授权文件保留在本机，发布凭证不会写入网页产物。完整命令和应用数据工作流见 [客户端文档](skills/README.md)。

## 做应用，无需先搭服务端

Cellapp 提供账户登录、访问控制和每个应用独立的数据表。Agent 可以在本机完成构建与发布；应用在浏览器中运行，通过受控接口读写共享数据。开发者可以专注于界面和交互，无需为了发布、用户访问或数据存储自行搭建服务端。

如果应用需要持续运行的后台任务或自定义服务端逻辑，仍需另行部署相应服务。

## 自己部署

[单机 Docker Compose 自部署指南](docs/self-hosting.md)涵盖完整平台：Go 服务、控制台、独立应用数据 API、PostgreSQL、默认私有 MinIO 和 HTTPS 入口。需要准备两个不同注册域的控制域名与应用通配域名、相应可信证书，以及 GitHub OAuth 应用。现有 Cloud Run 部署路径见[运维文档](docs/operations.md)。

## 参与开发

[开发指南](docs/development.md)包含仓库检出、本地依赖、构建、测试和贡献约定。公开 CLI 与 Skill 位于独立的 `skills` Git 子模块；Web 控制台和 Go 服务位于主仓库。

## 许可证

主仓库原创代码与文档使用 [Apache License 2.0](LICENSE)，版权声明见 [NOTICE](NOTICE)。独立 `skills` 子模块使用其自身的 [MIT License](skills/LICENSE)。
