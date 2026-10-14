import { renderToString } from 'react-dom/server';
import { createElement } from 'react';
import { readFile, writeFile, mkdir, cp } from 'node:fs/promises';
import path from 'node:path';
import App from '../src/App';
import { metadata, routePaths } from '../src/content';
import { logo } from '../src/logo';

const root = path.resolve(import.meta.dirname, '..');
const bundle = await readFile(path.join(root, '.build/bundle.html'), 'utf8');
const styles = [...bundle.matchAll(/<style[^>]*>[\s\S]*?<\/style>/g)].map(m => m[0]).join('\n');
const scripts = [...bundle.matchAll(/<script[^>]*>[\s\S]*?<\/script>/g)].map(m => m[0]).join('\n');
if (!styles || !scripts || /<script[^>]+src=/.test(scripts)) throw new Error('Bundle must contain inline styles and scripts');
const escape = (value: string) => value.replaceAll('&', '&amp;').replaceAll('"', '&quot;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
for (const route of routePaths) {
  const meta = metadata(route);
  const html = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="description" content="${escape(meta.description)}"><title>${escape(meta.title)}</title><link rel="icon" href="${logo}">${styles}</head><body><div id="root" data-route="${route}">${renderToString(createElement(App, { route }))}</div>${scripts}</body></html>`;
  const folder = path.join(root, 'dist', route);
  await mkdir(folder, { recursive: true });
  await writeFile(path.join(folder, 'index.html'), html);
  console.log(`Exported ${route}: ${Math.round(Buffer.byteLength(html) / 1024)} KiB`);
}
await writeFile(path.join(root, 'dist/bundle.html'), await readFile(path.join(root, 'dist/index.html')));
await cp(path.join(root, 'public/logos'), path.join(root, 'dist/logos'), { recursive: true });
await cp(path.join(root, 'public/deploy/AGENTS.md'), path.join(root, 'dist/deploy/AGENTS.md'));
for (const icon of ['favicon.svg', 'icons.svg']) {
  await cp(path.join(root, 'public', icon), path.join(root, 'dist', icon));
}
