import { afterEach, test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import worker, { controlRoute } from './cloudflare-worker.mjs';

const env = {
  CONTROL_HOST: 'cellapp.ai',
  APPS_DOMAIN: 'ce11.app',
  ORIGIN_URL: 'https://cellapp-example.us-west1.run.app',
  EDGE_ORIGIN_SECRET: 'a-dedicated-edge-secret-with-32-bytes',
};

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

test('every registered control endpoint has a declared apex route', () => {
  const source = readFileSync(new URL('../apps/server/internal/hosting/server.go', import.meta.url), 'utf8');
  const config = JSON.parse(readFileSync(new URL('./wrangler.edge.example.jsonc', import.meta.url), 'utf8'));
  const routePatterns = config.routes.map(route => route.pattern);
  const matches = path => routePatterns.some(pattern => {
    if (!pattern.startsWith('cellapp.ai/')) return false;
    const expression = new RegExp(`^${pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/\\\*/g, '.*')}$`);
    return expression.test(`cellapp.ai${path}`);
  });
  const registered = [...source.matchAll(/mux\.HandleFunc\("(?:GET|POST|PUT|DELETE) (\/[^"]+)"/g)].map(match => match[1]);
  for (const route of registered) {
    if (route === '/health') continue;
    assert.ok(controlRoute(route), `unclassified Go control endpoint ${route}`);
    const sample = route.replace(/\{[^}]+\}/g, 'a'.repeat(32));
    assert.ok(matches(sample), `missing Worker route for ${route}`);
  }
  for (const path of ['/', '/deploy/', '/pricing/', '/docs/', '/blog/', '/assets/site.css']) {
    assert.equal(controlRoute(path), false, path);
    assert.equal(matches(path), false, path);
  }
});

test('forwards control requests with authenticated original host and no redirect following', async () => {
  let forwarded;
  let options;
  globalThis.fetch = async (request, opts) => {
    forwarded = request;
    options = opts;
    return new Response(null, { status: 302, headers: { Location: 'https://github.com/login', 'Set-Cookie': '__Host-owner=value; Secure; Path=/' } });
  };
  const request = new Request('https://cellapp.ai/auth/login?next=%2Fapps', {
    headers: { 'X-Cellapp-Original-Host': 'evil.example', 'X-Cellapp-Edge-Secret': 'forged', 'X-Forwarded-Host': 'evil.example' },
  });
  const response = await worker.fetch(request, env);
  assert.equal(forwarded.url, 'https://cellapp-example.us-west1.run.app/auth/login?next=%2Fapps');
  assert.equal(forwarded.headers.get('X-Cellapp-Original-Host'), 'cellapp.ai');
  assert.equal(forwarded.headers.get('X-Cellapp-Edge-Secret'), env.EDGE_ORIGIN_SECRET);
  assert.equal(forwarded.headers.get('X-Forwarded-Host'), null);
  assert.equal(options.redirect, 'manual');
  assert.equal(response.status, 302);
  assert.equal(response.headers.get('Location'), 'https://github.com/login');
  assert.match(response.headers.get('Set-Cookie'), /__Host-owner=/);
  assert.equal(response.headers.get('Cache-Control'), 'private, no-store');
});

test('forwards root-host mutations and rejects public or retired paths', async () => {
  let calls = 0;
  globalThis.fetch = async request => {
    calls++;
    assert.equal(request.method, 'PUT');
    assert.equal(new URL(request.url).pathname, '/api/console/profile');
    assert.equal(new URL(request.url).search, '?view=full');
    assert.equal(request.headers.get('Cookie'), '__Host-owner=session');
    assert.equal(request.headers.get('Origin'), 'https://cellapp.ai');
    assert.equal(await request.text(), '{"displayName":"Alice"}');
    return Response.json({ displayName: 'Alice' });
  };
  const write = await worker.fetch(new Request('https://cellapp.ai/api/console/profile?view=full', {
    method: 'PUT', headers: { Cookie: '__Host-owner=session', Origin: 'https://cellapp.ai' }, body: '{"displayName":"Alice"}',
  }), env);
  assert.equal(write.status, 200);
  assert.equal(write.headers.get('Cache-Control'), 'private, no-store');
  assert.equal(calls, 1);
  for (const path of ['/', '/deploy/', '/assets/logo.svg', '/dashboard-extra', '/apps-extra', '/health']) {
    assert.equal((await worker.fetch(new Request(`https://cellapp.ai${path}`), env)).status, 404, path);
  }
  assert.equal((await worker.fetch(new Request('https://console.cellapp.ai/apps', {headers: {Authorization: 'Bearer old'}}), env)).status, 404);
  assert.equal(calls, 1);
});

test('preserves visitor upload method and body', async () => {
  let forwarded;
  globalThis.fetch = async request => {
    forwarded = request;
    return new Response('ok');
  };
  const host = `${'a'.repeat(32)}.ce11.app`;
  const response = await worker.fetch(new Request(`https://${host}/files/0`, { method: 'PUT', body: 'upload', headers: { Cookie: '__Host-cellapp=abc' } }), env);
  assert.equal(response.status, 200);
  assert.equal(forwarded.method, 'PUT');
  assert.equal(await forwarded.text(), 'upload');
  assert.equal(forwarded.headers.get('Cookie'), '__Host-cellapp=abc');
  assert.equal(forwarded.headers.get('X-Cellapp-Original-Host'), host);
});

test('keeps publication paused while an old origin serves static during rollback', async () => {
  const app = 'a'.repeat(32);
  const settings = { ...env, PUBLICATION_PAUSED: 'true', EDGE_STATIC_MODE: 'origin' };
  let calls = 0;
  globalThis.fetch = async request => { calls++; return new Response(new URL(request.url).pathname); };
  for (const [method, path] of [
    ['POST', `/apps/${app}/deployments`],
    ['PUT', `/apps/${app}/deployments/${'b'.repeat(32)}/files/0`],
    ['POST', `/apps/${app}/deployments/${'b'.repeat(32)}/publish`],
  ]) {
    const response = await worker.fetch(new Request(`https://${env.CONTROL_HOST}${path}`, { method }), settings);
    assert.equal(response.status, 503);
    assert.equal((await response.json()).error, 'publication_paused');
  }
  assert.equal(calls, 0);
  const staticResponse = await worker.fetch(new Request('https://notes.ce11.app/index.html'), settings);
  assert.equal(staticResponse.status, 200);
  assert.equal(await staticResponse.text(), '/index.html');
  assert.equal(calls, 1);
});

test('rejects unknown and malformed hosts before contacting origin', async () => {
  globalThis.fetch = () => { throw new Error('unexpected origin fetch'); };
  for (const host of ['unknown.example', 'ce11.app', 'a.b.ce11.app', 'evilconsole.cellapp.ai', 'console.cellapp.ai']) {
    const response = await worker.fetch(new Request(`https://${host}/`), env);
    assert.equal(response.status, 404, host);
  }
});

test('lets Cellapp reject an unknown single-level visitor hostname', async () => {
  let assertedHost;
  globalThis.fetch = async request => {
    assertedHost = request.headers.get('X-Cellapp-Original-Host');
    return new Response('Unknown host', { status: 404, headers: { 'X-Request-ID': 'probe' } });
  };
  const response = await worker.fetch(new Request('https://bad.ce11.app/'), env);
  assert.equal(assertedHost, 'bad.ce11.app');
  assert.equal(response.status, 404);
  assert.equal(response.headers.get('X-Request-ID'), 'probe');
});

async function edgeFixture() {
  const host = 'notes.ce11.app';
  const session = 'opaque-app-session';
  const app = 'a'.repeat(32);
  const deployment = 'b'.repeat(32);
  const pair = await crypto.subtle.generateKey({ name: 'Ed25519' }, true, ['sign', 'verify']);
  const publicKey = Buffer.from(await crypto.subtle.exportKey('raw', pair.publicKey)).toString('base64url');
  const sessionDigest = Buffer.from(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(session))).toString('base64url');
  const now = Math.floor(Date.now() / 1000);
  const issue = async (overrides = {}) => {
    const claims = { h: host, a: app, d: deployment, s: true, c: sessionDigest, i: now, e: now + 60, ...overrides };
    const signed = `v1.test.${Buffer.from(JSON.stringify(claims)).toString('base64url')}`;
    const signature = await crypto.subtle.sign('Ed25519', pair.privateKey, new TextEncoder().encode(signed));
    return `${signed}.${Buffer.from(signature).toString('base64url')}`;
  };
  const file = { path: 'index.html', size: 5, sha256: '2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824' };
  const digest = Buffer.from(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(file.path))).toString('hex');
  const objects = new Map([
    [`releases/${deployment}/_cellapp_manifest.json`, { json: async () => ({ files: [file] }) }],
    [`releases/${deployment}/${digest}`, 'hello'],
  ]);
  const bucket = {
    get: async key => typeof objects.get(key) === 'string' ? { body: new Response(objects.get(key)).body } : objects.get(key) || null,
    head: async key => objects.has(key) ? {} : null,
  };
  return { host, session, app, deployment, issue, objects, bucket,
    env: { ...env, R2_BUCKET: bucket, EDGE_GRANT_PUBLIC_KEYS: JSON.stringify({ test: publicKey }), EDGE_RATE_LIMITER: { limit: async () => ({ success: true }) } } };
}

const edgeFetch = (request, settings) => {
  request.headers.set('CF-Connecting-IP', '203.0.113.10');
  return worker.fetch(request, settings);
};

test('renews once for a page and serves subsequent assets from private R2', async () => {
  const f = await edgeFixture();
  let admissions = 0;
  const grant = await f.issue();
  globalThis.fetch = async request => {
    if (new URL(request.url).pathname !== '/_cellapp/edge-grant') throw new Error('unexpected static origin fetch');
    admissions++;
    assert.equal(request.headers.get('Cookie'), `__Host-cellapp=${f.session}`);
    return Response.json({ grant });
  };
  const first = await edgeFetch(new Request(`https://${f.host}/`, { headers: { Cookie: `__Host-cellapp=${f.session}`, Accept: 'text/html' } }), f.env);
  assert.equal(first.status, 200);
  assert.equal(await first.text(), 'hello');
  const cookie = first.headers.get('Set-Cookie');
  assert.match(cookie, /__Host-cellapp-grant=/);
  const token = cookie.match(/__Host-cellapp-grant=([^;]+)/)[1];
  const second = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${token}` } }), f.env);
  assert.equal(second.status, 200);
  assert.equal(await second.text(), 'hello');
  assert.equal(admissions, 1);
});

test('rejects host/session replay, expired grants and unlisted objects before R2 bytes', async () => {
  const f = await edgeFixture();
  const token = await f.issue();
  let reads = 0;
  const get = f.bucket.get;
  f.bucket.get = async key => { reads++; return get(key); };
  globalThis.fetch = async () => new Response('denied', { status: 403 });
  for (const [host, session] of [['other.ce11.app', f.session], [f.host, 'different']]) {
    const response = await edgeFetch(new Request(`https://${host}/index.html`, { headers: { Cookie: `__Host-cellapp=${session}; __Host-cellapp-grant=${token}` } }), f.env);
    assert.equal(response.status, 403);
  }
  const expired = await f.issue({ e: Math.floor(Date.now() / 1000) - 1 });
  const response = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${expired}` } }), f.env);
  assert.equal(response.status, 403);
  assert.equal(reads, 0);
  const unknown = await edgeFetch(new Request(`https://${f.host}/secret.js`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${token}` } }), f.env);
  assert.equal(unknown.status, 404);
});

test('serves HEAD, conditional responses and SPA fallback only after authorization', async () => {
  const f = await edgeFixture();
  const token = await f.issue();
  globalThis.fetch = () => { throw new Error('static request reached origin'); };
  const headers = { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${token}`, Accept: 'text/html' };
  const head = await edgeFetch(new Request(`https://${f.host}/index.html`, { method: 'HEAD', headers }), f.env);
  assert.equal(head.status, 200);
  assert.equal(head.headers.get('Content-Length'), '5');
  assert.equal(await head.text(), '');
  const conditional = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { ...headers, 'If-None-Match': head.headers.get('ETag') } }), f.env);
  assert.equal(conditional.status, 304);
  const fallback = await edgeFetch(new Request(`https://${f.host}/notes/123`, { headers }), f.env);
  assert.equal(fallback.status, 200);
  assert.equal(await fallback.text(), 'hello');
  const asset = await edgeFetch(new Request(`https://${f.host}/missing.js`, { headers }), f.env);
  assert.equal(asset.status, 404);
});

test('renews an expired grant with the existing session and fails closed if origin or R2 is unavailable', async () => {
  const f = await edgeFixture();
  const expired = await f.issue({ e: Math.floor(Date.now() / 1000) - 1 });
  const renewed = await f.issue();
  let calls = 0;
  globalThis.fetch = async () => { calls++; return Response.json({ grant: renewed }); };
  const request = new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${expired}` } });
  const response = await edgeFetch(request, f.env);
  assert.equal(response.status, 200);
  assert.equal(calls, 1);
  assert.match(response.headers.get('Set-Cookie'), /HttpOnly/);
  f.bucket.get = async () => { throw new Error('R2 unavailable'); };
  const unavailable = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${renewed}` } }), f.env);
  assert.equal(unavailable.status, 503);
  globalThis.fetch = async () => { throw new Error('origin unavailable'); };
  const originDown = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}` } }), f.env);
  assert.equal(originDown.status, 503);
});

test('keeps login recovery on the origin and rejects an incomplete edge binding', async () => {
  const f = await edgeFixture();
  let paths = [];
  globalThis.fetch = async request => {
    paths.push(new URL(request.url).pathname);
    if (paths.length === 1) return new Response('login required', { status: 401 });
    return new Response(null, { status: 303, headers: { Location: 'https://cellapp.ai/auth/app' } });
  };
  const response = await edgeFetch(new Request(`https://${f.host}/`, { headers: { Accept: 'text/html' } }), f.env);
  assert.equal(response.status, 303);
  assert.deepEqual(paths, ['/_cellapp/edge-grant', '/']);
  assert.equal((await edgeFetch(new Request(`https://${f.host}/`), { ...f.env, EDGE_GRANT_PUBLIC_KEYS: '' })).status, 503);
});

test('limits a visitor source before reading grants or R2', async () => {
  const f = await edgeFixture();
  const token = await f.issue();
  let checked = '';
  f.env.EDGE_RATE_LIMITER = { limit: async ({ key }) => { checked = key; return { success: false }; } };
  f.bucket.get = async () => { throw new Error('R2 must not be read'); };
  globalThis.fetch = () => { throw new Error('origin must not be called'); };
  const response = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${token}` } }), f.env);
  assert.equal(response.status, 429);
  assert.equal(checked, '203.0.113.10');
  assert.equal((await worker.fetch(new Request(`https://${f.host}/`), f.env)).status, 503);
});

test('parallel assets reuse one grant while application data reaches Cloud Run', async () => {
  const f = await edgeFixture();
  const token = await f.issue();
  let forwarded = 0;
  globalThis.fetch = async request => {
    forwarded++;
    assert.equal(new URL(request.url).pathname, '/rest/v1/notes');
    return Response.json([{ id: 1 }]);
  };
  const cookie = `__Host-cellapp=${f.session}; __Host-cellapp-grant=${token}`;
  const assets = await Promise.all(Array.from({ length: 8 }, () => edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: cookie } }), f.env)));
  assert.deepEqual(assets.map(response => response.status), Array(8).fill(200));
  assert.equal(forwarded, 0);
  const data = await edgeFetch(new Request(`https://${f.host}/rest/v1/notes`, { headers: { Cookie: cookie } }), f.env);
  assert.equal(data.status, 200);
  assert.equal(forwarded, 1);
});

test('never serves a guessed private key or another release through the app host', async () => {
  const f = await edgeFixture();
  const token = await f.issue();
  const cookie = `__Host-cellapp=${f.session}; __Host-cellapp-grant=${token}`;
  let objectReads = 0;
  const get = f.bucket.get;
  f.bucket.get = async key => { objectReads++; return get(key); };
  globalThis.fetch = async () => new Response('denied', { status: 401 });
  const guessed = await edgeFetch(new Request(`https://${f.host}/releases/${f.deployment}/index.html`, { headers: { Cookie: cookie } }), f.env);
  assert.equal(guessed.status, 404);
  assert.equal(objectReads, 1, 'only the private manifest may be read');
  const oldToken = await f.issue({ d: 'c'.repeat(32) });
  const old = await edgeFetch(new Request(`https://${f.host}/index.html`, { headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${oldToken}` } }), f.env);
  assert.equal(old.status, 503, 'missing old release must not read current release');
  const anonymous = await edgeFetch(new Request(`https://${f.host}/index.html`), f.env);
  assert.equal(anonymous.status, 401);
  assert.equal(objectReads, 2, 'anonymous request touched R2');
});

test('keeps private release lookups separate across application hosts', async () => {
  const f = await edgeFixture();
  const otherHost = 'other.ce11.app';
  const otherDeployment = 'c'.repeat(32);
  const otherBody = 'second';
  const otherHash = Buffer.from(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(otherBody))).toString('hex');
  const pathHash = Buffer.from(await crypto.subtle.digest('SHA-256', new TextEncoder().encode('index.html'))).toString('hex');
  f.objects.set(`releases/${otherDeployment}/_cellapp_manifest.json`, {
    json: async () => ({ files: [{ path: 'index.html', size: otherBody.length, sha256: otherHash }] }),
  });
  f.objects.set(`releases/${otherDeployment}/${pathHash}`, otherBody);
  const first = await f.issue();
  const second = await f.issue({ h: otherHost, a: 'd'.repeat(32), d: otherDeployment });
  globalThis.fetch = () => { throw new Error('valid static grant must not reach origin'); };
  const request = (host, grant) => edgeFetch(new Request(`https://${host}/index.html`, {
    headers: { Cookie: `__Host-cellapp=${f.session}; __Host-cellapp-grant=${grant}` },
  }), f.env);
  assert.equal(await (await request(f.host, first)).text(), 'hello');
  assert.equal(await (await request(otherHost, second)).text(), otherBody);
  assert.equal(await (await request(f.host, first)).text(), 'hello');
});
