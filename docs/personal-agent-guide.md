# 个人 Agent 平台：当前版本怎么用

本文说明当前开发工作区已实现的能力。代码尚未提交、发布到线上服务或发布新版 npm 包；以下本机命令应在本工作区执行，不能把 `npm install -g @cellapp/cli@latest` 当作本版验证方式。Windows 实机和隔离候选环境验收尚待完成，见[验证记录](personal-agent-verification.md)。

目前有两条使用路径：在 Cellapp 静态应用里调用使用者自己的云模型；在自己的电脑上使用已有编码 Agent 或手动建立环境。模型连接需要兼容 OpenAI Chat Completions 的**云供应商 HTTPS 接口**；本版不接 Ollama、LM Studio，也不提供 Cellapp 模型额度。

## 一、给应用接入模型

1. 在控制台打开“Model connections”（`/dashboard/model-connections`），填写连接名称、供应商的 HTTPS API base URL（例如 `https://provider.example/v1`）和该供应商的 API key。模型 ID 可先留空，由 Cellapp 请求供应商的 `/models`；如果获取失败，可删除该连接并重新创建，手动填写供应商给出的模型 ID。密钥由服务端保存，不能写入应用前端或项目文件。
2. 保存后点击“Test capabilities”，查看文本、流式响应和工具调用的检测结果。检测结果针对页面显示的测试模型；换钥后应重新检测。应用需要哪种能力，就先确认对应结果。
3. 用包含模型功能的当前 SDK 构建静态应用，部署时指定应用 API `1.2`，例如 `cellapp deploy --project /absolute/project --output dist --build 'npm run build' --api-version 1.2`。已经固定为 `1.1` 的旧页面不会因为单独更新 SDK 而获得模型 API。当前工作区可用 `npm run cli -- deploy ...` 运行源码版 CLI；正式用户应等服务、CLI 和 SDK 配套发布后再用已发布包。
4. 使用者打开已登录、且自己有权访问的应用。每位使用者需配置自己的供应商连接；应用作者的连接不会自动分享给访客。第一次调用 `cellapp.models.current()` 时，SDK 会显示模型选择界面；如需授权，按界面进入控制台确认**当前应用可以使用哪个供应商连接**。回到应用后选择模型。一个应用可以获准使用多个连接；连接授权后，应用代码可以调用该连接下的任意模型，不受选择界面显示的默认模型限制。模型费用由使用者与其供应商结算。
5. 控制台的“Model connections”页可查看应用授权和近 30 天请求数，按连接撤销授权。撤销后，该应用后续模型请求会失败；删除连接也会停止使用它的新请求。用量中的 token 仅统计供应商实际返回的数字，不是账单。

应用内最小示例：

```ts
import {createCellappClient} from '@cellapp/sdk';

const cellapp = createCellappClient();
const chosen = await cellapp.models.current(); // 未选择时由 SDK 显示选择/授权界面
console.log(chosen.provider, chosen.name);

const reply = await cellapp.chat({
  messages: [{role: 'user', content: '请用一句话介绍这个应用'}],
});
console.log(reply.choices[0]?.message.content);
```

如果应用要让使用者切换模型，调用 `await cellapp.models.select()`。选择偏好只留在该浏览器本地，不同步到 Cellapp 账号；换浏览器需要重新选择。模型请求仍会把选中的模型 ID 发给 Cellapp 服务端路由，并由服务端逐次检查应用及连接授权，因此“本地选择”不等于“模型调用完全不经过 Cellapp”。

应用也可以使用同源的 OpenAI 形态接口：`GET /_cellapp/v1/models` 和 `POST /_cellapp/v1/chat/completions`。前者返回当前使用者已授权给该应用的模型并集，模型 ID 形如 `cellapp:<连接 ID>:<供应商模型 ID>`；后者目前支持已记录的文本、流式和工具调用子集，不承诺与 OpenAI 所有字段一比一兼容。`/_cellapp/v1/responses` 尚未提供。完整 SDK 用法见[客户端说明](../skills/packages/sdk/README.md)，字段和错误码见[Web API](web-api.md)。

## 二、使用已有 Agent 建立本机环境

前提是 Node.js 22.12+、npm 和当前工作区的客户端依赖。以下命令在**仓库根目录**执行；`npm run cli --` 运行此工作区的源码版 CLI。CLI 默认连接 `https://cellapp.ai`；本地或候选服务需在每条命令加同一个 `--origin <HTTPS 控制域>`。当前变更尚未发布，不能将已发布 npm 包当作本版验证。

```sh
npm ci
npm --prefix skills ci
npm run cli -- doctor
npm run cli -- doctor --project /absolute/project --json
npm run cli -- set agent codex
```

`doctor` 只读检查 Node/npm、本地 Cellapp 配置、服务连通、Cellapp 登录、Codex/Claude/Cursor CLI 安装与可确认的登录状态。只有显式给出 `--project` 才检查那个目录的构建脚本、应用关联和静态输出。`unknown` 表示不能确认，不等于已登录；诊断不会读取其他 Agent 的凭证文件或打印密钥。`set agent` 也可选 `claude` 或 `cursor`，只保存本机已安装 CLI 的首选名称，不执行它、不安装它，也不会替你登录。

接着在本机用选定 Agent 自己的方式登录，并让它遵循[公开 Cellapp 部署 Skill](../skills/skills/cellapp-deploy/SKILL.zh-CN.md)检查项目、运行 Cellapp CLI、在浏览器中请求部署授权、构建并部署静态产物。Agent 登录与 Cellapp 浏览器授权是两件独立的事；浏览器显示的验证码要与终端一致。没有编码 Agent 时，直接按[手动部署指引](../apps/website/public/deploy/AGENTS.zh-CN.md)完成同样的 CLI 步骤。部署只在本机执行构建，服务端只托管静态产物。

## 常见问题

- **线上 CLI 还不认识 `doctor` 或 `set agent`：** 当前变更尚未发布，先使用此工作区的源码版命令及匹配服务验证。
- **`doctor` 显示 Cellapp 登录过期：** 运行 `cellapp login` 并在浏览器批准新验证码；之后再次检查。
- **编码 Agent 登录为 `unknown`：** 使用该 Agent 自己的命令确认登录，不读取或复制它的凭证文件。没有编码 Agent 可按手动指引继续。
- **项目没有静态输出：** 在本机构建后确认包含 `index.html` 的目录，再传给 `cellapp deploy --output`。
- **模型列表为空：** 检查供应商 `/models`；失败时在连接中手动填写准确模型 ID，确认应用 API 1.2 与连接授权。
- **应用要求模型授权：** 在 SDK 弹窗中打开控制台授权页，核对应用和连接后返回选择模型。

首版实机与候选环境的通过状态以[验证记录](personal-agent-verification.md)为准。远程编码、Agent 分享与 fork、自动安装其他编码 CLI、Responses API 均属于后续阶段。
