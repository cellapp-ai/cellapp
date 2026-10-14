export type Cadence = 'monthly' | 'yearly';
export interface Plan { id: string; name: string; audience: string; monthly: number; yearly: number; apps: string; support: string; access: string; recommended?: boolean }
export const github = 'https://github.com/cellapp-ai/skills';
export const plans: Plan[] = [
  { id: 'starter', name: 'Starter', audience: 'Try the workflow. Share your first build.', monthly: 0, yearly: 0, apps: '1 app', support: 'Community', access: 'Public links' },
  { id: 'ship', name: 'Ship', audience: 'Keep demos and internal tools moving.', monthly: 19, yearly: 15, apps: '10 apps', support: 'Community', access: 'Per-app access keys', recommended: true },
  { id: 'scale', name: 'Scale', audience: 'Give a growing product team more room.', monthly: 59, yearly: 49, apps: 'Unlimited apps', support: 'Priority', access: 'Per-app access keys' },
];
export interface DocSection { id: string; title: string; heading: string; intro: string; steps: { heading: string; text: string; code?: string; link?: { label: string; href: string } }[] }
export const docs: DocSection[] = [
  { id: 'getting-started', title: 'Getting Started', heading: 'Give your AI one request.', intro: 'Open the deployment guide in the project you are building. Your AI can inspect the project, install the CLI and deploy it; you approve access in the browser.', steps: [
    { heading: '1. Copy the deployment prompt', text: 'Use the prompt on the deployment page with the AI tool working in your project.', link: { label: 'Open the deployment guide', href: '/deploy/' } },
    { heading: '2. Approve access', text: 'When the CLI requests device authorization, open the link and compare the code. Do not send your password or credential in chat.' },
    { heading: '3. Verify and share', text: 'Wait for a published result, then open the HTTPS URL. Check the main page, assets and nested routes before sharing.' },
  ] },
  { id: 'deploy', title: 'Deploy', heading: 'Build locally. Publish the result.', intro: 'Cellapp accepts static frontends. Vite, Astro, or an exported Next.js site can fit when the output does not need a running application server.', steps: [
    { heading: 'Manual CLI path', text: 'Node.js 22.12 or later is required. Install @cellapp/cli from npm if needed, then run this command from the project directory, changing dist to your actual output folder.', code: 'cellapp deploy --project . --output dist --build "npm run build"' },
    { heading: 'Agent instructions', text: 'The public runbook covers project inspection, CLI setup, browser authorization, publishing and recovery.', link: { label: 'Read Agent instructions', href: 'https://cellapp.ai/deploy/AGENTS.md' } },
  ] },
  { id: 'reference', title: 'CLI / API Reference', heading: 'Know what the command does.', intro: 'The deployment example publishes a static directory. Refer to the current CLI help and project documentation for supported flags, authentication, and API contracts.', steps: [
    { heading: 'Inspect available commands', text: 'Use the installed CLI help as the source of truth for your version.', code: 'cellapp --help' },
    { heading: 'API reference', text: 'The public client repository publishes HTTP contracts for supported deployment and application data operations.', link: { label: 'View public client contracts', href: 'https://github.com/cellapp-ai/skills/tree/main/contracts' } },
  ] },
  { id: 'config', title: 'Configuration', heading: 'Keep secrets out of the build.', intro: 'Static frontend files can be read by people who load your app. Treat values embedded in the build as public.', steps: [
    { heading: 'Separate public values from secrets', text: 'Use your frontend build tool for public configuration. Keep service credentials and deployment credentials outside the compiled frontend.' },
    { heading: 'Share access deliberately', text: 'If access keys are enabled for an application, distribute the key to intended viewers separately from deployment credentials.' },
  ] },
  { id: 'troubleshooting', title: 'Troubleshooting', heading: 'Find the missing step.', intro: 'Most static deployment problems are visible in the output folder or the browser network panel.', steps: [
    { heading: 'A blank page or missing assets', text: 'Confirm index.html exists and that asset URLs match your hosting path. Check the browser network panel for failed requests, then rebuild and deploy the output folder.' },
    { heading: 'A nested URL returns 404', text: 'Generate static entries for your routes or configure a host fallback that serves your entry document. Test a direct visit and a refresh, not just client-side navigation.' },
    { heading: 'The upload did not finish', text: 'Read the CLI error, confirm the directory and network connection, and retry when the underlying issue is resolved. Include the command and error text when requesting help.', link: { label: 'Find support in the project', href: github } },
  ] },
];
export interface Article { slug: string; category: string; date: string; readMinutes: number; title: string; summary: string; body: { heading: string; paragraphs: string[]; code?: string }[] }
export const articles: Article[] = [
  { slug: 'ship-static', category: 'Shipping notes', date: '2026-09-18', readMinutes: 3, title: 'The calm path from local build to live link', summary: 'Why a static frontend deserves a dependable home, even when it starts as a demo.', body: [
    { heading: 'Static does not mean temporary.', paragraphs: ['Frontend teams already do the hard work: choose the tools, compose the interface, and produce a clean build. The final step is giving that build a dependable home.', 'A static folder is a useful boundary. It holds the page, styles, scripts, and assets needed to show the work. Keeping that boundary clear makes deployment easier to understand.'] },
    { heading: 'The handoff should outlive the meeting.', paragraphs: ['A reviewer may not open the link while you are on the call. When the same address works tomorrow, the interface becomes a shared reference and feedback can stay attached to the work.', 'Before sharing, check the page in a fresh window. Follow its links, load a nested route directly, and make sure the demo has the configuration it needs.'], code: 'cellapp deploy --project . --output dist' },
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
export const routePaths = ['/', '/deploy/', '/pricing/', '/docs/', '/blog/', ...articles.map(a => `/blog/${a.slug}/`)];
export const normalizeRoute = (route: string) => route === '/' ? '/' : `${route.replace(/\/+$/, '')}/`;
export function metadata(route: string) {
  if (route === '/deploy/') return { title: 'Deploy — Cellapp', description: 'Ask your coding agent to build and deploy a static app to Cellapp. One prompt handles CLI setup, authorization and publication.' };
  if (route === '/pricing/') return { title: 'Pricing — Cellapp', description: 'Compare Cellapp plan drafts, features, and monthly or annual pricing.' };
  if (route === '/docs/') return { title: 'Documentation — Cellapp', description: 'Prepare, deploy, verify, and share a static frontend with Cellapp.' };
  if (route === '/blog/') return { title: 'Journal — Cellapp', description: 'Notes on static shipping, frontend reviews, and dependable handoffs.' };
  const article = articleBySlug(route.split('/')[2]);
  if (article) return { title: `${article.title} — Cellapp`, description: article.summary };
  return { title: 'Cellapp — Ship a frontend. Share it safely.', description: 'Cellapp turns static frontends into stable, shareable HTTPS applications.' };
}
