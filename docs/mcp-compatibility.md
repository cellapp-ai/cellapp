# MCP 兼容性决策

本变更固定使用官方 `github.com/modelcontextprotocol/go-sdk` v1.3.1。项目保持 Go 1.24；SDK v1.5.0 及更新版本要求 Go 1.25，会同时改变服务镜像和构建链，因此首版以 SDK 完整支持的 MCP `2025-06-18` 协议和 Streamable HTTP 为验收基线。旧版客户端可在 SDK 支持范围内协商；`2025-11-25` 的部分实现不作为首版承诺，`2026-07-28` 需要后续连同 Go 工具链升级评估。服务使用无状态 HTTP 处理，并以客户端库完成工具发现、调用和错误响应测试。

Codex 需要可公开访问的 HTTPS MCP URL，使用 Streamable HTTP 与 OAuth 浏览器授权。豆包侧以火山引擎 AgentKit 的外部 MCP 接入作为候选集成目标：其文档要求 Streamable HTTP、`tools/list`/`tools/call` 和可配置的出站凭据。AgentKit 是否能把 Cellapp 的每位所有者交互式 OAuth 授权传递给导入的 MCP 服务，须在隔离候选环境实际验证；在验证前不声明豆包连接已通过。若产品所指的“豆包 Agent”是另一个客户端，需另补该客户端的连接和授权验收记录。

SDK 增加一个公开维护的 Go 依赖及其传递依赖，需要随 MCP 协议和 Go 版本升级持续维护。当前只用于协议层；所有者鉴权、数据隔离、清单校验和发布事务继续由 Cellapp 业务层负责。

参考：[Go SDK 版本与协议矩阵](https://github.com/modelcontextprotocol/go-sdk)、[Codex MCP 连接](https://developers.openai.com/resources/docs-mcp)、[AgentKit 导入已有 MCP 服务](https://docs.volcengine.com/docs/agentkit/Integrating_existing_MCP_Servers_into_AgentKit_Gateway?lang=en)。
