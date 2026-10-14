import { chromium } from 'playwright-core';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import worker from '../infra/cloudflare-worker.mjs';

const control = process.env.TEST_CONTROL_ORIGIN;
const appURL = process.env.TEST_EDGE_APP_URL;
const deployment = process.env.TEST_EDGE_DEPLOYMENT;
const html = process.env.TEST_EDGE_HTML;
const appHost = new URL(appURL).hostname;
const origin = new URL(control);
const bytes = new Map([['index.html', Buffer.from(html)], ['main.js', Buffer.from('x')]]);
const manifest = { files: Array.from(bytes, ([path, body]) => ({
  path, size: body.length, sha256: createHash('sha256').update(body).digest('hex'),
})) };
const objects = new Map(Array.from(bytes, ([path, body]) => [
  `releases/${deployment}/${createHash('sha256').update(path).digest('hex')}`, body,
]));
const bucket = {
  get: async key => key === `releases/${deployment}/_cellapp_manifest.json`
    ? { json: async () => manifest }
    : objects.has(key) ? { body: new Response(objects.get(key)).body } : null,
  head: async key => objects.has(key) ? {} : null,
};
const settings = {
  CONTROL_HOST: origin.host,
  APPS_DOMAIN: 'apps.localhost',
  APP_PORT: origin.port,
  ORIGIN_URL: control,
  EDGE_ORIGIN_SECRET: process.env.TEST_EDGE_ORIGIN_SECRET,
  EDGE_GRANT_PUBLIC_KEYS: JSON.stringify({ 'browser-key': process.env.TEST_EDGE_PUBLIC_KEY }),
  EDGE_RATE_LIMITER: { limit: async () => ({ success: true }) },
  R2_BUCKET: bucket,
};
const nativeFetch = globalThis.fetch;
const originCalls = [];
globalThis.fetch = async (request, options) => {
  originCalls.push(new URL(request.url).pathname);
  return nativeFetch(request, options);
};
const browser = await chromium.launch({
  executablePath: process.env.CHROME_PATH ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  headless: true,
  args: ['--host-resolver-rules=MAP *.localhost 127.0.0.1',
    `--ignore-certificate-errors-spki-list=${process.env.TEST_BROWSER_SPKI}`],
});
const realNow = Date.now;
try {
  const context = await browser.newContext({ ignoreHTTPSErrors: false });
  await context.route('**/*', async route => {
    const original = route.request();
    if (!new URL(original.url()).hostname.endsWith('.apps.localhost')) {
      await route.continue();
      return;
    }
    const headers = await original.allHeaders();
    headers['CF-Connecting-IP'] = '127.0.0.1';
    const body = original.postDataBuffer();
    const request = new Request(original.url(), {
      method: original.method(), headers,
      ...(body && original.method() !== 'GET' && original.method() !== 'HEAD' ? { body } : {}),
    });
    const response = await worker.fetch(request, settings);
    await route.fulfill({
      status: response.status,
      headers: Object.fromEntries(response.headers),
      body: Buffer.from(await response.arrayBuffer()),
    });
  });
  const page = await context.newPage();
  await page.goto(control + '/_test/visitor');
  await page.goto(appURL + '/notes');
  await page.getByRole('heading', { name: 'Edge app' }).waitFor();
  // The public first visit obtains the app session from the origin; reload to exercise edge delivery.
  await page.reload();
  await page.getByRole('heading', { name: 'Edge app' }).waitFor();
  assert.equal(new URL(page.url()).hostname, appHost);
  let cookies = await context.cookies(appURL);
  let session = cookies.find(cookie => cookie.name === '__Host-cellapp');
  let grant = cookies.find(cookie => cookie.name === '__Host-cellapp-grant');
  assert.ok(session && grant?.httpOnly && grant.secure && !grant.domain.startsWith('.'),
    JSON.stringify(cookies.map(cookie => ({ name: cookie.name, domain: cookie.domain, secure: cookie.secure, httpOnly: cookie.httpOnly }))));

  const beforeAssets = originCalls.length;
  const assets = await page.evaluate(async () => Promise.all(
    Array.from({ length: 8 }, () => fetch('/main.js').then(async response => [response.status, await response.text()])),
  ));
  assert.deepEqual(assets, Array.from({ length: 8 }, () => [200, 'x']));
  assert.equal(originCalls.length, beforeAssets, 'authorized assets reached Cloud Run');

  await context.clearCookies();
  await context.addCookies([session]);
  const renewed = await page.evaluate(async () => {
    const response = await fetch('/main.js');
    return [response.status, await response.text()];
  });
  assert.deepEqual(renewed, [200, 'x']);
  assert.equal(new URL(page.url()).hostname, appHost, 'renewal redirected browser to login');

  const data = await page.evaluate(async () => (await fetch('/rest/v1/notes')).status);
  assert.ok(data >= 400);
  assert.ok(originCalls.includes('/rest/v1/notes'), 'data API did not reach Cloud Run');

  assert.equal((await nativeFetch(control + '/_test/restrict-edge', { method: 'POST' })).status, 204);
  const immediateData = await page.evaluate(async () => (await fetch('/rest/v1/notes')).status);
  assert.equal(immediateData, 403);
  const stillGranted = await page.evaluate(async () => (await fetch('/main.js')).status);
  assert.equal(stillGranted, 200);
  Date.now = () => realNow() + 61_000;
  const afterRevocation = await page.evaluate(async () => {
    const response = await fetch('/main.js');
    return [response.status, await response.text()];
  });
  assert.equal(afterRevocation[0], 403);
  assert.notEqual(afterRevocation[1], 'x');
  Date.now = realNow;

  assert.equal((await nativeFetch(control + '/_test/public-edge', { method: 'POST' })).status, 204);
  cookies = await context.cookies(appURL);
  session = cookies.find(cookie => cookie.name === '__Host-cellapp');
  await context.clearCookies();
  await context.addCookies([session]);
  assert.equal(await page.evaluate(async () => (await fetch('/main.js')).status), 200);
  assert.equal((await nativeFetch(control + '/_test/suspend-edge', { method: 'POST' })).status, 204);
  assert.equal(await page.evaluate(async () => (await fetch('/main.js')).status), 200);
  Date.now = () => realNow() + 61_000;
  const afterSuspension = await page.evaluate(async () => {
    const response = await fetch('/main.js');
    return [response.status, await response.text()];
  });
  assert.equal(afterSuspension[0], 403);
  assert.notEqual(afterSuspension[1], 'x');
  console.log('Edge browser: handoff, private R2-binding stub, data forwarding, renewal, revocation and 60-second suspension bound passed.');
} finally {
  Date.now = realNow;
  globalThis.fetch = nativeFetch;
  await browser.close();
}
