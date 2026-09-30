import { chromium, type APIResponse } from 'playwright-core';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, rm, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { Client, randomKey } from '../skills/packages/cli/src/client.js';
import { deploy } from '../skills/packages/cli/src/deploy.js';

const origin = process.env.TEST_CONTROL_ORIGIN!;
const root = await mkdtemp(join(tmpdir(), 'cellapp-browser-'));
const browser = await chromium.launch({executablePath: process.env.CHROME_PATH ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', headless: true, args: ['--host-resolver-rules=MAP *.localhost 127.0.0.1']});
try {
  // The test CA is trusted by the OS; both browser and CLI verify TLS.
  const ownerContext = await browser.newContext({ignoreHTTPSErrors: false});
  const ownerPage = await ownerContext.newPage();
  const client = new Client(origin, join(root, 'credentials'));
  let verification!: (uri: string) => void;
  const ready = new Promise<string>(resolve => { verification = resolve; });
  const loggingIn = client.login(message => { verification(message.match(/https:\/\/\S+/)![0]); });
  await ownerPage.goto(await ready);
  await ownerPage.getByRole('link', {name: 'Continue to sign in'}).click();
  const decisionResponse = ownerPage.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/console/device/decision');
  await ownerPage.getByRole('button', {name: 'Approve access'}).click();
  const decision = await decisionResponse;
  assert.equal(decision.status(), 200, await decision.text());
  await ownerPage.getByRole('status').filter({hasText: 'Deployment access approved'}).waitFor();
  await loggingIn;
  assert.ok(client.token);
  const cli = async (...args: string[]) => {
    const {stdout} = await promisify(execFile)(process.execPath, ['--import', 'tsx', 'skills/packages/cli/src/main.ts', ...args, '--origin', origin, '--config-dir', join(root, 'credentials')]);
    return JSON.parse(stdout);
  };

  const project = join(root, 'project'); await mkdir(join(project, 'dist'), {recursive: true});
  const html = (version: string) => `<h1>${version}</h1><script src="/main.js"></script>`;
  await writeFile(join(project, 'dist', 'index.html'), html('Version 1'));
  await writeFile(join(project, 'dist', 'main.js'), 'document.body.dataset.loaded = "yes";');
  const first = await deploy(client, project, {output: 'dist', spa: true}, () => {});
  assert.ok('shareKey' in first && first.shareKey);
  const key = (first as {shareKey: string}).shareKey;
  const metadata = JSON.parse(await readFile(join(project, 'cellapp.json'), 'utf8'));
  assert.ok((await cli('apps')).some((app: {id: string}) => app.id === metadata.appId));
  assert.ok(!JSON.stringify(metadata).includes(key));
  const context1 = await browser.newContext({ignoreHTTPSErrors: false});
  const context2 = await browser.newContext({ignoreHTTPSErrors: false});
  const page1 = await context1.newPage(); const page2 = await context2.newPage();
  for (const page of [page1, page2]) {
    await page.goto(first.url + '/notes');
    await page.getByLabel('分享密钥').fill(key);
    await page.getByRole('button', {name: '打开应用'}).click();
    await page.getByRole('heading', {name: 'Version 1'}).waitFor();
    assert.equal(new URL(page.url()).pathname, '/notes');
    await page.waitForFunction(() => document.body.dataset.loaded === 'yes');
  }
  assert.ok((await context1.cookies()).every(c => c.httpOnly && c.secure && !c.domain.startsWith('.')));
  await page1.evaluate(() => localStorage.setItem('private-data', 'browser-one'));
  assert.equal(await page2.evaluate(() => localStorage.getItem('private-data')), null);
  const secondProject = join(root, 'second'); await mkdir(join(secondProject, 'dist'), {recursive: true});
  await writeFile(join(secondProject, 'dist', 'index.html'), '<h1>Another app</h1>');
  const second = await deploy(client, secondProject, {output: 'dist', spa: false}, () => {});
  const secondKey = (second as {shareKey: string}).shareKey;
  assert.notEqual(secondKey, key);
  const otherPage = await context1.newPage();
  await otherPage.goto(second.url);
  await otherPage.getByLabel('分享密钥').fill(secondKey);
  await otherPage.getByRole('button', {name: '打开应用'}).click();
  await otherPage.getByRole('heading', {name: 'Another app'}).waitFor();
  assert.equal(await otherPage.evaluate(() => localStorage.getItem('private-data')), null);
  assert.equal((await context1.request.get(second.url + '/notes', {headers: {Accept: 'text/html'}})).status(), 404);
  assert.equal(await page1.evaluate(async other => {
    try { await fetch(other, {credentials: 'include'}); return false; } catch { return true; }
  }, second.url), true);
  assert.equal(await page1.evaluate(async () => {
    try { await navigator.serviceWorker.register('/main.js'); return false; } catch { return true; }
  }), true);
  assert.equal(await page1.evaluate(async origin => {
    try { await fetch(origin + '/apps', {credentials: 'include'}); return false; } catch { return true; }
  }, origin), true);
  const asset = await context1.request.get(first.url + '/missing.js'); assert.equal(asset.status(), 404);
  const outsider = await browser.newContext({ignoreHTTPSErrors: false});
  for (const method of ['GET', 'HEAD']) {
    const response: APIResponse = await outsider.request.fetch(first.url + '/main.js', {method, headers: {'Accept': '*/*', 'If-None-Match': '*'}});
    assert.equal(response.status(), 401);
  }
  await writeFile(join(project, 'dist', 'index.html'), html('Version 2'));
  const originalRequest = client.request.bind(client);
  client.request = async function<T>(path: string, method?: string, body?: unknown, binary?: boolean): Promise<T> {
    const result = await originalRequest<T>(path, method, body, binary);
    if (path.endsWith('/publish')) throw new Error('Simulated lost publish response');
    return result;
  };
  await assert.rejects(deploy(client, project, {}, () => {}), /Simulated lost publish response/);
  client.request = originalRequest;
  const update = await deploy(client, project, {}, () => {}); assert.equal(update.url, first.url);
  await page1.reload(); await page1.getByRole('heading', {name: 'Version 2'}).waitFor();
  await ownerPage.goto(origin + '/applications');
  await ownerPage.locator(`a[href="/applications/${metadata.appId}"]`).waitFor();
  await ownerPage.goto(origin + '/applications/' + metadata.appId);
  await ownerPage.reload();
  await ownerPage.getByRole('heading',{name:'Current release'}).waitFor();
  await ownerPage.getByRole('button',{name:'Reset share key',exact:true}).click();
  await ownerPage.getByRole('button',{name:'Confirm',exact:true}).click();
  await ownerPage.getByRole('heading',{name:'New share key'}).waitFor();
  const webKey = await ownerPage.getByRole('heading',{name:'New share key'}).locator('..').locator('code').textContent();
  assert(webKey && /^[a-f0-9]{64}$/.test(webKey));
  await page1.reload(); await page1.getByLabel('分享密钥').waitFor();
  await page1.getByLabel('分享密钥').fill(webKey);await page1.getByRole('button',{name:'打开应用'}).click();await page1.getByRole('heading',{name:'Version 2'}).waitFor();
  const newKey: string = (await cli('reset-key', metadata.appId)).shareKey;
  for (const page of [page1, page2]) { await page.reload(); await page.getByLabel('分享密钥').waitFor(); }
  await page1.getByLabel('分享密钥').fill(newKey); await page1.getByRole('button', {name: '打开应用'}).click();
  await page1.getByRole('heading', {name: 'Version 2'}).waitFor();
  await ownerPage.getByRole('button',{name:'Delete application',exact:true}).click();
  await ownerPage.getByRole('button',{name:'Confirm',exact:true}).click();
  await ownerPage.getByRole('heading',{name:'Applications',exact:true}).waitFor();
  assert.equal((await context1.request.get(first.url)).status(), 404);
  await otherPage.reload(); await otherPage.getByRole('heading', {name: 'Another app'}).waitFor();
  await ownerPage.goto(origin + '/credentials');
  await ownerPage.getByRole('button', {name: 'Revoke', exact: true}).click();
  await ownerPage.getByRole('button', {name: 'Confirm', exact: true}).click();
  await ownerPage.getByRole('status').filter({hasText:'Deployment credential revoked'}).waitFor();
  await assert.rejects(client.request('/apps'), /expired or revoked/);
  await client.logout(); await assert.rejects(readFile(client.credentialFile));
  console.log('Browser + CLI: device login, publish, two sessions, local data isolation, SW/CORS rejection, update, reset, delete and logout passed.');
} finally {
  await browser.close(); await rm(root, {recursive: true, force: true});
}
