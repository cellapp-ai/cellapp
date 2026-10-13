import { afterEach, test } from 'node:test';
import assert from 'node:assert/strict';
import worker from './cloudflare-worker.mjs';

const env = {
  CONTROL_HOST: 'console.cellapp.ai',
  APPS_DOMAIN: 'ce11.app',
  ORIGIN_URL: 'https://cellapp-example.us-west1.run.app',
  EDGE_ORIGIN_SECRET: 'a-dedicated-edge-secret-with-32-bytes',
};

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

test('forwards control requests with authenticated original host and no redirect following', async () => {
  let forwarded;
  let options;
  globalThis.fetch = async (request, opts) => {
    forwarded = request;
    options = opts;
    return new Response(null, { status: 302, headers: { Location: 'https://github.com/login', 'Set-Cookie': '__Host-owner=value; Secure; Path=/' } });
  };
  const request = new Request('https://console.cellapp.ai/auth/login?next=%2Fapps', {
    headers: { 'X-Cellapp-Original-Host': 'evil.example', 'X-Cellapp-Edge-Secret': 'forged', 'X-Forwarded-Host': 'evil.example' },
  });
  const response = await worker.fetch(request, env);
  assert.equal(forwarded.url, 'https://cellapp-example.us-west1.run.app/auth/login?next=%2Fapps');
  assert.equal(forwarded.headers.get('X-Cellapp-Original-Host'), 'console.cellapp.ai');
  assert.equal(forwarded.headers.get('X-Cellapp-Edge-Secret'), env.EDGE_ORIGIN_SECRET);
  assert.equal(forwarded.headers.get('X-Forwarded-Host'), null);
  assert.equal(options.redirect, 'manual');
  assert.equal(response.status, 302);
  assert.equal(response.headers.get('Location'), 'https://github.com/login');
  assert.match(response.headers.get('Set-Cookie'), /__Host-owner=/);
  assert.equal(response.headers.get('Cache-Control'), 'private, no-store');
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

test('rejects unknown and malformed hosts before contacting origin', async () => {
  globalThis.fetch = () => { throw new Error('unexpected origin fetch'); };
  for (const host of ['unknown.example', 'ce11.app', 'a.b.ce11.app', 'evilconsole.cellapp.ai']) {
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
