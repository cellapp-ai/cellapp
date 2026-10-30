# 个人 Agent 平台：新版首版验证记录

日期：2026-10-28。主仓工作分支 `codex/personal-agent-platform-proposal`，客户端子仓分支 `codex/personal-agent-platform-sdk`。本页记录改用现有编码 Agent 与 `cellapp doctor` 后的本轮验证；旧 Boot Agent、设备配对试验结果不计入新版首版通过状态。

| 检查 | 本轮结果 | 边界 |
| --- | --- | --- |
| 客户端 typecheck、33 项测试、build、公开边界、Skill、包内容 | 通过 | macOS；32 项通过，Windows 专用发现用例跳过；包内 52 个允许文件，无 mini-SWE 运行时 |
| `cellapp doctor --json` 真实本机命令 | 通过 | macOS Node 22.22.0；检测到 Codex、Claude、Cursor，分别报告可确认的登录状态；在线控制域返回 401，仍报告域可达但登录未确认 |
| `cellapp doctor` 离线、未登录、显式项目与无关目录测试 | 通过 | 使用隔离目录和网络替身；不泄露凭证、不扫描未指定项目 |
| `cellapp set agent` 选择、切换、无效命令及旧选择保持测试 | 通过 | 隔离私有配置，文件权限 0600；另用实际 CLI 命令完成 `set agent codex` → `doctor --json` 冒烟测试，未修改用户真实首选项 |
| 控制台 typecheck、lint、build、正式产物浏览器检查 | 通过 | 11 组浏览器检查，含新 Local setup 页面、窄屏与可访问性；API 使用明确的 UI 夹具 |
| 服务端隔离 PostgreSQL `go test -race -count=1 ./apps/server/...` | 通过 | 覆盖模型、权限、迁移和既有静态发布；新旧设备路由关闭的专项回归在其后单独通过 |
| 服务端 `go vet ./apps/server/...` | 通过 | 静态检查 |
| 主仓 `npm test` 与 `npm run build` | 通过 | 包含客户端、服务端非隔离检查、控制台 lint 和正式产物浏览器验收 |
| 旧设备凭证、Boot、项目和远程任务关闭检查 | 通过 | 隔离 PostgreSQL 中插入旧设备凭证；个人设备/Boot/任务领取接口返回 404，远程任务提交返回 409；已执行迁移未改写 |
| 官网 lint、build、正式静态预览浏览器检查 | 通过 | 既有 lint 警告仍在；九个页面和公开指引通过；首次未启动预览时测试连接拒绝，启动正式预览后重测通过 |
| Go 服务加正式控制台与真实静态 Agent 浏览器联调 | 通过 | 首次发现新 Local setup 深链未列入服务端白名单；修正后在隔离 PostgreSQL、真实浏览器与正式产物上复测通过 |
| 已登录编码 Agent 依照新 Skill 完成真实部署，以及纯手动部署 | 未执行 | 公开指引已完成；仍需真实端到端验证命令与浏览器授权提示 |
| Linux 和 Windows 实机本机设置流程 | 未执行 | Windows 实机由用户后续单独验收；客户端 CI 有三系统矩阵但本轮没有远端矩阵运行结果 |
| 隔离候选环境、云供应商、两位使用者、Cloudflare 路由与回滚 | 未执行 | 用户后续单独验收；本地服务与浏览器夹具不替代 |

远程编码、预览发布、设备绑定和 Agent 分发仍属后续阶段。`/auth/device` 是既有部署 CLI 的浏览器授权，继续可用；个人设备试验性配对已关闭。首版任务 3.6 与 7.2 在用户提供验收结果前保持未完成。
