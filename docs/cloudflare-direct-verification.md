# Cloudflare 直连试验记录

2026-10-13，在 `cellapp-511002` / `us-west1` 进行了不删除原 Google HTTPS 负载均衡的试验。此记录只陈述本轮实际验证，不将测试替身当作线上验收。

| 项目 | 结果 |
| --- | --- |
| 构建 | Cloud Build `fe0a2f29-8d7b-43ba-9a06-072f8a74f66a` 成功；镜像 digest 为 `sha256:3a6fa1324127cfa00fa3509996c262ee749bed19319b2fe1ed4671ed20c2a8ac`。 |
| 独立回源 | Cloud Run `cellapp-edge-probe` 在 `us-west1` 就绪，公开 ingress，最多 1 个实例，使用现有生产资源与专用 `cellapp-edge-origin-secret:1`；用户明确授权该测试服务。正式服务接管后，测试服务已删除。 |
| Cloudflare Worker | 试验 Worker `cellapp-edge-probe` 以专用 secret binding 转发到独立服务。`workers.dev/login` 返回 200 与 Cellapp 的 `X-Request-ID`；直接访问测试服务的 `run.app/login` 返回 404。正式 Worker `cellapp-edge` 现回源 `https://cellapp-nok7htj6gq-uw.a.run.app`，两个路由均已切到它；试验 Worker 已删除，正式 Worker 的 workers.dev 地址未启用。 |
| 域名切换试验 | 为 `console.cellapp.ai/*` 和 `*.ce11.app/*` 创建 Worker 路由，开启两条 DNS A 记录代理。最初两个 Zone 的 `security_level` 均为 `under_attack`，Cloudflare 在 Worker 前返回 403；用户明确授权后，均改为 `medium` 并重新切流。 |
| 真实域名 HTTP | `console.cellapp.ai/login` 返回 200 与 Cellapp `X-Request-ID`；`/auth/login` 返回 302，GitHub 回调为 `https://console.cellapp.ai/auth/callback`，保留 `__Host-login` Cookie；JS、CSS 和图标均返回 200。无效访客域 `00000000000000000000000000000000.ce11.app` 返回 Cellapp 404 与 `X-Request-ID`。 |
| 浏览器验收 | 用户于 2026-10-13 确认经 Cloudflare 入口完成 GitHub 登录并正常看到控制台。没有提供已发布访客应用的链接，访客授权与资源未做真实应用验收。 |
| 正式服务 | 对 `cellapp-00010-n87` 先加 0% 流量标签，公开 `/health` 返回 200，修订版本的 `Ready`、`Active`、`ContainerHealthy` 和 `ResourcesAvailable` 均为 `True`，再切至 100% 流量。Worker 改回源正式服务后，控制域返回 200、无效访客域返回 404，直接 `run.app/login` 返回 404。 |
| DNS 与旧入口 | `console.cellapp.ai`、`*.ce11.app` 的代理 A 记录从旧 IP 改为 Cloudflare 推荐的 `192.0.2.0` 占位地址，Worker 仍分别返回 200/404。未发布的 `cellapp.ai` 和 `ce11.app` 根域旧 A 记录已移除。Google 全局转发规则、目标 HTTPS 代理、URL 映射、后端服务、serverless NEG、专用公网 IP 和证书映射均已删除；托管证书及 DNS 授权保留。删除后再次核对公开域名仍返回 200/404。 |
| 自动审批 | 自动审批最初拒绝将两个 Zone 的全站安全级别改为 `medium`；收到用户明确授权后已执行。自动审批还拒绝将标记 `Retired` 的正式服务修订版本切到 100% 流量，该切换未执行。 |

本轮本地验证：Go hosting 测试、Worker 的 4 项测试、部署脚本的 21 项测试、`openspec validate --strict`、`git diff --check` 均通过。没有提供已发布访客应用链接，访客授权与资源未做真实应用验收。负载均衡固定资源已删除；Cloud Run、Cloudflare Worker、数据库、对象存储及流量费用仍会产生。
