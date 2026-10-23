export type Cadence = 'monthly' | 'yearly';
export interface Plan { id: string; name: string; audience: string; monthly: number; yearly: number; apps: string; support: string; access: string; recommended?: boolean }
export const github = 'https://github.com/cellapp-ai/cellapp';
export const deploymentPrompt = 'Read https://cellapp.ai/deploy/AGENTS.md, build this project locally and deploy it; use its supported app database tables if needed, and give me the published URL.';
export const plans: Plan[] = [
  { id: 'starter', name: 'Starter', audience: 'Try the workflow. Share your first build.', monthly: 0, yearly: 0, apps: '1 app', support: 'Community', access: 'Public links' },
  { id: 'ship', name: 'Ship', audience: 'Keep demos and internal tools moving.', monthly: 19, yearly: 15, apps: '10 apps', support: 'Community', access: 'Per-app access keys', recommended: true },
  { id: 'scale', name: 'Scale', audience: 'Give a growing product team more room.', monthly: 59, yearly: 49, apps: 'Unlimited apps', support: 'Priority', access: 'Per-app access keys' },
];
export type DocLocale = 'en' | 'zh-CN';
export interface DocStep { heading: string; text: string; points?: string[]; note?: string; code?: string; codeLabel?: string; link?: { label: string; href: string } }
export interface DocSection { id: string; title: string; heading: string; intro: string; steps: DocStep[] }
export interface DocGuide { label: string; title: string; summary: string; outline: string; next: string; finish: string; sections: DocSection[] }
export const docGuides: Record<DocLocale, DocGuide> = {
  en: {
    label: 'Documentation', title: 'From project to published app.',
    summary: 'Prepare a static project, publish it with your coding Agent, verify the result, and recover when a step fails.',
    outline: 'On this page', next: 'Next', finish: 'Open the deployment prompt',
    sections: [
      { id: 'getting-started', title: 'Getting Started', heading: 'Prepare your first deployment.', intro: 'Cellapp hosts static frontend files. Your coding Agent works in your local project; Cellapp does not run your build or application server.', steps: [
        { heading: '1. Check that the project can become static files', text: 'Locate the project build command and its output directory. A deployable output contains index.html and the assets it needs. Plain HTML can be published without a build step.', points: ['If the app needs a continuously running server, SSR, or serverless functions, prepare a genuine static alternative first.', 'Do not put passwords, private API keys, or other secrets in frontend files; admitted viewers can read them.'] },
        { heading: '2. Start with the copyable request', text: 'Copy this request into the coding Agent already working on this project. The Agent reads the public instructions, checks the project, and obtains the official client on demand; you do not need a global installation.', code: deploymentPrompt, link: { label: 'Open the deployment page', href: '/deploy/' } },
        { heading: '3. Approve access promptly', text: 'If this machine is not signed in, the Agent should request browser authorization before a long build. Open the verification URL as soon as the Agent shares it and compare the code. An expired code needs a new authorization request.', note: 'Approve in the browser. Do not paste a password, cookie, or deployment credential into chat.' },
        { heading: '4. Check the returned result', text: 'After local build and upload, wait for a published result and an HTTPS app URL. Open the URL, check the main page and assets, then refresh a nested route you intend to share.', link: { label: 'See verification steps', href: '#deploy' } },
      ] },
      { id: 'installation', title: 'Installation', heading: 'Run the CLI yourself.', intro: 'If you prefer the terminal, use the published Cellapp CLI. Node.js 22.12 or later and npm are required. Agent-led deployment needs no manual installation: the Agent obtains the client on demand.', steps: [
        { heading: 'Install the CLI', text: 'Install the latest published npm package globally.', code: 'npm install -g @cellapp/cli@latest', codeLabel: 'Global npm install' },
        { heading: 'Verify the installation', text: 'Check that the cellapp command is available before using it in your project.', code: 'cellapp --help', codeLabel: 'Verify installation', link: { label: 'Read the CLI source and manual workflow', href: 'https://github.com/cellapp-ai/skills#manual-install' } },
      ] },
      { id: 'deploy', title: 'Deploy & Verify', heading: 'Follow the publication through.', intro: 'The Agent builds and uploads the static output on your machine. A successful publication activates a release and returns its app URL.', steps: [
        { heading: 'Read the official Agent guide', text: 'The public guide describes project inspection, early authorization, publishing, and recovery. It is the source of truth for current CLI commands and flags.', link: { label: 'Read English Agent instructions', href: 'https://cellapp.ai/deploy/AGENTS.md' } },
        { heading: 'Watch for browser authorization', text: 'The Agent verifies the published client, checks sign-in, and gives you a verification URL and code if approval is needed. Keep the Agent running while you approve; if the code expires, request a fresh one.', note: 'A chat reply alone does not complete browser approval.' },
        { heading: 'Verify the live release', text: 'Use the URL returned with published status. In a fresh browser visit, check that the landing page loads, images and scripts work, and a direct visit plus refresh of an important nested URL behaves as intended.', points: ['Confirm the URL is the intended app, not a local preview.', 'If the app is restricted, test the viewer experience with the intended access method before sharing.'] },
        { heading: 'Share the right access information', text: 'Send the app link to intended viewers. Public apps still require viewers to sign in to Cellapp; restricted apps require an additional grant or key redemption. Keep deployment credentials separate from viewer access.', link: { label: 'Open the dashboard', href: '/dashboard' } },
      ] },
      { id: 'reference', title: 'CLI/API Reference', heading: 'Know which source to use.', intro: 'Use the published Agent guide for the current workflow and the public client repository for exact CLI and HTTP details. The page does not replace either reference.', steps: [
        { heading: 'Agent workflow', text: 'The English and Chinese Agent guides cover local builds, browser authorization, retry behavior, and application tables.', link: { label: 'Read the Agent guide', href: 'https://cellapp.ai/deploy/AGENTS.md' } },
        { heading: 'Client and API contracts', text: 'Check the public repository for supported CLI options, HTTP fields, and application data contracts before automating a step. Do not infer an API from an example on this page.', link: { label: 'View public contracts', href: 'https://github.com/cellapp-ai/skills/tree/main/contracts' } },
        { heading: 'Supported application data', text: 'Cellapp supports controlled application database tables when an app needs shared persistent data. The Agent guide points to the separate plan, apply, and verify workflow. Browser-only local state does not synchronize across devices.', link: { label: 'Read the deployment Skill', href: 'https://github.com/cellapp-ai/skills/blob/main/skills/cellapp-deploy/SKILL.md' } },
      ] },
      { id: 'config', title: 'Configuration & Access', heading: 'Keep the boundary clear.', intro: 'Everything bundled into a static frontend is visible to people who can open it. Plan configuration and viewer access with that in mind.', steps: [
        { heading: 'Public frontend configuration', text: 'Use your frontend build tool for values intended to be public, such as a public endpoint URL. Keep secrets and deployment credentials outside compiled files and public environment variables.', note: 'A build-time variable is not secret merely because it came from an environment file.' },
        { heading: 'Owner authorization versus viewer access', text: 'Browser device authorization lets the owner deploy. It does not give viewers access. Public viewers sign in; restricted viewers also need approval or an access key. Share a viewer key only with its intended recipients.', link: { label: 'Manage apps and grants', href: '/dashboard' } },
        { heading: 'Static routing and assets', text: 'Ensure generated asset URLs match the hosted location. For client-side nested routes, have the Agent check whether a static entry or supported SPA fallback is needed, then test a direct visit and refresh.' },
        { heading: 'Sales are not active', text: 'Sell in the 4S vision is coming later. The current deployment flow does not create a listing, checkout, or paid subscription.' },
      ] },
      { id: 'troubleshooting', title: 'Troubleshooting', heading: 'Check the symptom, then retry.', intro: 'Identify the failing stage before changing the app. Keep the error text so the Agent can recover the same deployment when possible.', steps: [
        { heading: 'No valid static output', text: 'Check the build log and output folder for index.html. Fix the build command or output directory, then ask the Agent to rerun the build and deployment. A server-only app needs a genuine static alternative.', link: { label: 'Return to preparation', href: '#getting-started' } },
        { heading: 'Blank page or missing assets', text: 'Open browser developer tools and inspect failed asset requests. Correct the generated asset paths or public configuration, rebuild locally, and deploy the corrected output. Reopen the published URL in a fresh browser.' },
        { heading: 'Nested URL fails on refresh', text: 'Visit the nested URL directly. If it returns 404, generate static entries or use the supported SPA fallback for a client-side router; then redeploy and repeat the direct-visit and refresh check.' },
        { heading: 'Authorization expired or interrupted', text: 'Ask the Agent for a fresh browser verification URL and code, approve promptly, then continue the same deployment. Never send your password or credential in chat.', link: { label: 'Read authorization steps', href: 'https://cellapp.ai/deploy/AGENTS.md' } },
        { heading: 'Upload or viewer access fails', text: 'For a transient upload or network error, let the Agent retry the same deployment once and report the exact error if it fails again. If the URL opens but a viewer is blocked, check sign-in and the app grant or key separately from deployment.', link: { label: 'Manage viewer access', href: '/dashboard' } },
      ] },
    ],
  },
  'zh-CN': {
    label: '使用文档', title: '从本地项目到已发布应用。',
    summary: '准备静态项目，让编码 Agent 发布，核验结果，并在某一步失败时找到恢复方法。',
    outline: '本页目录', next: '下一节', finish: '打开部署提示词',
    sections: [
      { id: 'getting-started', title: '开始使用', heading: '准备首次部署。', intro: 'Cellapp 托管静态前端文件。编码 Agent 在你的本地项目中工作；Cellapp 不运行你的构建过程或应用服务器。', steps: [
        { heading: '1. 确认项目能生成静态文件', text: '找到项目的构建命令和产物目录。可部署的产物包含 index.html 及其所需资源。普通 HTML 无需构建步骤即可发布。', points: ['如果应用依赖持续运行的服务器、SSR 或 Serverless 函数，先准备真正的静态替代方案。', '不要将密码、私有 API 密钥等秘密放入前端文件；获准访问的用户可以读取它们。'] },
        { heading: '2. 复制部署请求', text: '将这条提示词发给正在此项目中工作的编码 Agent。Agent 会阅读公开指引、检查项目，并按需获取官方客户端；你无需全局安装。', code: deploymentPrompt, link: { label: '打开部署页', href: '/deploy/' } },
        { heading: '3. 尽早完成授权', text: '如果这台设备尚未登录，Agent 应在耗时构建前发起浏览器授权。收到验证网址后尽快打开并核对验证码；若验证码过期，请 Agent 重新发起授权。', note: '请在浏览器中批准。不要把密码、Cookie 或部署凭证发到聊天里。' },
        { heading: '4. 核对返回结果', text: '本地构建与上传完成后，等待发布成功状态和 HTTPS 应用网址。打开网址，检查首页与资源，再直接访问并刷新你准备分享的深层页面。', link: { label: '查看核验步骤', href: '#deploy' } },
      ] },
      { id: 'installation', title: '安装', heading: '自行使用 CLI。', intro: '如果你想在终端中操作，可以使用已发布的 Cellapp CLI。需要 Node.js 22.12 或更高版本及 npm。由 Agent 部署时无需手动安装：Agent 会按需获取客户端。', steps: [
        { heading: '安装 CLI', text: '全局安装最新发布的 npm 包。', code: 'npm install -g @cellapp/cli@latest', codeLabel: 'npm 全局安装' },
        { heading: '验证安装', text: '在项目中使用前，检查 cellapp 命令是否可用。', code: 'cellapp --help', codeLabel: '验证安装', link: { label: '阅读 CLI 源码和手动操作说明', href: 'https://github.com/cellapp-ai/skills#manual-install' } },
      ] },
      { id: 'deploy', title: '部署与核验', heading: '跟进发布全过程。', intro: 'Agent 在你的机器上构建并上传静态产物。发布成功后，新版本生效并返回应用网址。', steps: [
        { heading: '阅读官方 Agent 指引', text: '公开指引说明项目检查、提前授权、发布和失败恢复。当前 CLI 命令及参数以它为准。', link: { label: '阅读中文 Agent 指引', href: 'https://cellapp.ai/deploy/AGENTS.zh-CN.md' } },
        { heading: '留意浏览器授权', text: 'Agent 验证已发布客户端并检查登录状态。如需批准，它会提供验证网址和验证码。请保持 Agent 运行并在浏览器中批准；若验证码过期，请获取新码。', note: '只在聊天中回复，无法完成浏览器授权。' },
        { heading: '核验线上版本', text: '使用发布成功时返回的网址。在全新浏览器会话中检查首页、图片和脚本，并直接访问、刷新一个重要的深层网址。', points: ['确认打开的是目标应用，而不是本地预览。', '受限应用在分享前，还应以预期的访客授权方式验证访问。'] },
        { heading: '分享正确的访问信息', text: '将应用链接发给预期访客。公开应用的访客仍需登录 Cellapp；受限应用还需额外授权或兑换访问密钥。部署凭证与访客权限应分开管理。', link: { label: '打开控制台', href: '/dashboard' } },
      ] },
      { id: 'reference', title: 'CLI/API 参考', heading: '找到权威信息来源。', intro: '当前工作流程以已发布 Agent 指引为准，准确的 CLI 与 HTTP 细节以公开客户端仓库为准。本页不能替代这些资料。', steps: [
        { heading: 'Agent 工作流程', text: '中英文 Agent 指引覆盖本地构建、浏览器授权、重试行为和应用数据表。', link: { label: '阅读 Agent 指引', href: 'https://cellapp.ai/deploy/AGENTS.zh-CN.md' } },
        { heading: '客户端和 API 契约', text: '自动化操作前，先在公开仓库核对受支持的 CLI 参数、HTTP 字段和应用数据契约。不要从本页示例推断未说明的 API。', link: { label: '查看公开契约', href: 'https://github.com/cellapp-ai/skills/tree/main/contracts' } },
        { heading: '受控应用数据', text: '需要共享持久化数据时，Cellapp 支持受控应用数据库表。Agent 指引提供独立的计划、应用与验证流程入口。仅保存在浏览器本地的数据不会跨设备同步。', link: { label: '阅读部署 Skill', href: 'https://github.com/cellapp-ai/skills/blob/main/skills/cellapp-deploy/SKILL.md' } },
      ] },
      { id: 'config', title: '配置与访问', heading: '分清公开与私密边界。', intro: '静态前端中打包的内容，对能够打开应用的人可见。请据此规划配置和访客访问。', steps: [
        { heading: '公开的前端配置', text: '前端构建工具只应用来注入可以公开的值，例如公开接口地址。私密凭证和部署凭证不要进入编译产物或公开环境变量。', note: '值来自环境文件，并不代表打包后仍然保密。' },
        { heading: '所有者授权与访客权限', text: '浏览器设备授权允许所有者部署，不等于给访客授权。公开应用访客需要登录；受限应用访客还需获得批准或访问密钥。访问密钥仅分享给预期接收人。', link: { label: '管理应用与授权', href: '/dashboard' } },
        { heading: '静态路由和资源', text: '确保生成的资源网址与托管路径一致。若使用客户端深层路由，请 Agent 检查是否需要静态入口或受支持的 SPA 回退，并测试直接访问与刷新。' },
        { heading: '销售功能尚未开放', text: '4S 愿景中的 Sell 将来才会提供。当前部署流程不会创建商品、结账或付费订阅。' },
      ] },
      { id: 'troubleshooting', title: '故障排查', heading: '按症状检查，然后重试。', intro: '先确定失败发生在哪个阶段，再修改应用。保留错误信息，帮助 Agent 在可能时恢复同一次部署。', steps: [
        { heading: '没有有效的静态产物', text: '查看构建日志和产物目录，确认存在 index.html。修复构建命令或产物目录后，请 Agent 重新构建并部署。只能在服务器运行的应用需要真正的静态替代方案。', link: { label: '返回准备步骤', href: '#getting-started' } },
        { heading: '空白页或资源丢失', text: '打开浏览器开发者工具，检查失败的资源请求。修正资源路径或公开配置，在本地重新构建并部署，再用全新浏览器会话打开发布网址。' },
        { heading: '深层网址刷新失败', text: '直接访问深层网址。如果返回 404，应生成静态页面入口，或为客户端路由使用受支持的 SPA 回退；重新部署后，再检查直接访问与刷新。' },
        { heading: '授权过期或中断', text: '请 Agent 生成新的浏览器验证网址和验证码，尽快批准，然后继续同一次部署。不要在聊天中发送密码或凭证。', link: { label: '阅读授权步骤', href: 'https://cellapp.ai/deploy/AGENTS.zh-CN.md' } },
        { heading: '上传失败或访客无法访问', text: '暂时性上传或网络错误可请 Agent 对同一次部署重试一次；再次失败时提供准确报错。如果应用网址可打开，但访客被拒绝，应分别检查登录状态和应用授权或密钥。', link: { label: '管理访客权限', href: '/dashboard' } },
      ] },
    ],
  },
};
export interface Article { slug: string; category: string; date: string; readMinutes: number; title: string; summary: string; body: { heading: string; paragraphs: string[]; code?: string }[] }
export const articles: Article[] = [
  { slug: 'ship-static', category: 'Shipping notes', date: '2026-09-18', readMinutes: 3, title: 'The calm path from local build to live link', summary: 'Why a static frontend deserves a dependable home, even when it starts as a demo.', body: [
    { heading: 'Static does not mean temporary.', paragraphs: ['Frontend teams already do the hard work: choose the tools, compose the interface, and produce a clean build. The final step is giving that build a dependable home.', 'A static folder is a useful boundary. It holds the page, styles, scripts, and assets needed to show the work. Keeping that boundary clear makes deployment easier to understand.'] },
    { heading: 'The handoff should outlive the meeting.', paragraphs: ['A reviewer may not open the link while you are on the call. When the same address works tomorrow, the interface becomes a shared reference and feedback can stay attached to the work.', 'Before sharing, check the page in a fresh window. Follow its links, load a nested route directly, and make sure the demo has the configuration it needs.'] },
    { heading: 'Leave a clear next step.', paragraphs: ['Send a short explanation with the URL: what changed, what to try, and where feedback should go. If viewers need an access key, share it with the intended people. A good link and a little context make the handoff easier.'] },
  ] },
  { slug: 'frontend-handoff', category: 'Practice', date: '2026-08-29', readMinutes: 2, title: 'A better handoff for the next frontend review', summary: 'What makes a review link useful after the meeting ends.', body: [
    { heading: 'Give the reviewer a place to start.', paragraphs: ['A review works best when the goal is visible. Tell the reviewer which flow matters and which questions you want answered. Link to that flow directly when you can.', 'A screenshot captures one moment. An accessible build lets someone try the transitions, keyboard controls, and small-screen layout for themselves.'] },
    { heading: 'Check the experience outside your session.', paragraphs: ['Your browser may already have cached assets or signed-in state. Open a fresh window and follow the same steps as a first-time viewer.', 'Refresh the deepest link you intend to share. If it fails, fix the static entry or fallback before the review.'] },
    { heading: 'Make feedback actionable.', paragraphs: ['Ask for the route, viewport size, expected behavior, and what actually happened. These details turn a broad opinion into a change the team can verify.'] },
  ] },
  { slug: 'https-by-default', category: 'Product', date: '2026-08-12', readMinutes: 2, title: 'Why HTTPS should be the starting line', summary: 'Sharing is easier when the link is ready for a browser outside your local setup.', body: [
    { heading: 'Local assumptions do not travel.', paragraphs: ['A localhost address works for you, but it is not a handoff. A hosted HTTPS URL gives a viewer an address their own browser can open without recreating your development environment.'] },
    { heading: 'Test the browser boundary.', paragraphs: ['Some browser capabilities, including modern clipboard APIs, depend on a secure context. A hosted HTTPS page is a useful place to check the behavior you intend viewers to use.', 'HTTPS does not make frontend secrets safe. Values included in static files still need to be treated as public. Keep credentials outside the build.'] },
    { heading: 'Share deliberately.', paragraphs: ['Verify the published URL, check whether access is restricted, and send the right context with the link. Transport security and application access solve different parts of the handoff.'] },
  ] },
];
export const articleBySlug = (slug: string) => articles.find(a => a.slug === slug);
export const routePaths = ['/', '/deploy/', '/pricing/', '/docs/', '/docs/zh-CN/', '/blog/', ...articles.map(a => `/blog/${a.slug}/`)];
export const normalizeRoute = (route: string) => route === '/' ? '/' : `${route.replace(/\/+$/, '')}/`;
export function metadata(route: string) {
  if (route === '/deploy/') return { title: 'Deploy — Cellapp', description: 'Ask your coding agent to build locally and publish a static frontend with Cellapp. Approve access in your browser and receive an HTTPS app URL.' };
  if (route === '/pricing/') return { title: 'Pricing — Cellapp', description: 'Compare Cellapp plan drafts, features, and monthly or annual pricing.' };
  if (route === '/docs/') return { title: 'Documentation — Cellapp', description: 'Prepare, deploy, verify, and share a static frontend with Cellapp. Read the complete English guide.' };
  if (route === '/docs/zh-CN/') return { title: '使用文档 — Cellapp', description: '阅读 Cellapp 中文指南：准备静态项目、授权发布、核验应用、管理访问并排查问题。' };
  if (route === '/blog/') return { title: 'Journal — Cellapp', description: 'Notes on static shipping, frontend reviews, and dependable handoffs.' };
  const article = articleBySlug(route.split('/')[2]);
  if (article) return { title: `${article.title} — Cellapp`, description: article.summary };
  return { title: 'Cellapp — your apps, your rules.', description: 'Cellapp hosts your static frontend as a shareable HTTPS app with access controls. Ask your coding Agent to deploy from your project.' };
}
export const routeLanguage = (route: string) => route === '/docs/zh-CN/' ? 'zh-CN' : 'en';
