import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';

const root = resolve(import.meta.dirname, '..');
const source = await readFile(process.env.CELLAPP_API_REGISTRY_FILE ?? resolve(root, 'apps/server/internal/hosting/app_api_version.go'), 'utf8');
const changelog = await readFile(process.env.CELLAPP_API_CHANGELOG_FILE ?? resolve(root, 'apps/website/public/deploy/api-changelog.md'), 'utf8');
const guide = await readFile(resolve(root, 'apps/website/public/deploy/api-versions.md'), 'utf8');
const guideZh = await readFile(resolve(root, 'apps/website/public/deploy/api-versions.zh-CN.md'), 'utf8');
const runbooks = await Promise.all(['AGENTS.md', 'AGENTS.zh-CN.md'].map(name => readFile(resolve(root, 'apps/website/public/deploy', name), 'utf8')));
const pairs = [...source.matchAll(/\{Version: "([0-9]+\.[0-9]+)", Date: "([0-9]{6})", Status: "(?:supported|retired)"\}/g)].map(([, version, date]) => ({version, date}));
if (!pairs.length) throw new Error('No application API contract found in registry');
for (const {version, date} of pairs) {
  if (!changelog.includes(`## ${version} = ${date}`)) throw new Error(`Missing changelog entry for ${version} = ${date}`);
}
if (!guide.includes('api-changelog.md') || !guideZh.includes('api-changelog.md') ||
  runbooks.some(text => !text.includes('api-changelog.md') || !text.includes('api-versions.'))) {
  throw new Error('Agent version guide or changelog link is missing');
}
process.stdout.write(`Application API changelog covers ${pairs.length} contract(s).\n`);
