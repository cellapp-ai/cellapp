const grantCookie = '__Host-cellapp-grant';
const appCookie = '__Host-cellapp';

function event(name, status) {
  console.log(JSON.stringify({ component: 'cellapp-edge', event: name, status }));
}

function originalHost(request, env) {
  const url = new URL(request.url);
  const host = env.APP_PORT ? url.host : url.hostname;
  if (host === env.CONTROL_HOST) return host;
  const suffix = `.${env.APPS_DOMAIN}${env.APP_PORT ? ':' + env.APP_PORT : ''}`;
  if (host.endsWith(suffix) && /^[a-z0-9-]+$/.test(host.slice(0, -suffix.length))) return host;
  if (env.PROBE_HOST && url.hostname.endsWith('.workers.dev')) return env.PROBE_HOST;
  return null;
}

// The apex route is shared with Pages. Only control namespaces may reach Cloud Run.
export function controlRoute(path) {
  return path === '/dashboard' || path.startsWith('/dashboard/') ||
    path === '/api/console' || path.startsWith('/api/console/') ||
    path === '/api/control/v1' || path.startsWith('/api/control/v1/') ||
    path === '/auth' || path.startsWith('/auth/') ||
    path === '/mcp' || path === '/oauth/authorize' || path === '/oauth/token' || path === '/oauth/register' || path === '/oauth/revoke' ||
    path === '/.well-known/oauth-protected-resource' || path === '/.well-known/oauth-protected-resource/mcp' || path === '/.well-known/oauth-authorization-server' ||
    path === '/device-authorizations' || path.startsWith('/device-authorizations/') ||
    path === '/apps' || path.startsWith('/apps/') ||
    path === '/limits' || path === '/credentials/current' ||
    path === '/a' || path.startsWith('/a/');
}

function cookieValue(request, name) {
  const entry = (request.headers.get('Cookie') || '').split(';').map(part => part.trim()).find(part => part.startsWith(`${name}=`));
  return entry ? entry.slice(name.length + 1) : '';
}

function base64url(bytes) {
  return btoa(String.fromCharCode(...bytes)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function unbase64url(value) {
  if (!/^[A-Za-z0-9_-]+$/.test(value)) throw new Error('invalid base64url');
  const raw = atob(value.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - value.length % 4) % 4));
  return Uint8Array.from(raw, char => char.charCodeAt(0));
}

async function sha256(value) {
  return new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value)));
}

async function verifyGrant(value, host, session, env, now = Math.floor(Date.now() / 1000)) {
  if (!value || value.length > 4096) return null;
  try {
    const parts = value.split('.');
    if (parts.length !== 4 || parts[0] !== 'v1') return null;
    const keys = JSON.parse(env.EDGE_GRANT_PUBLIC_KEYS || '{}');
    if (!Object.hasOwn(keys, parts[1])) return null;
    const key = await crypto.subtle.importKey('raw', unbase64url(keys[parts[1]]), { name: 'Ed25519' }, false, ['verify']);
    const valid = await crypto.subtle.verify('Ed25519', key, unbase64url(parts[3]), new TextEncoder().encode(parts.slice(0, 3).join('.')));
    if (!valid) return null;
    const claims = JSON.parse(new TextDecoder().decode(unbase64url(parts[2])));
    const digest = base64url(await sha256(session));
    const validSession = claims.m === 'anonymous' ? !session && claims.c === '' :
      (claims.m === 'session' || claims.m === undefined) && !!session && claims.c === digest;
    if (claims.h !== host || !validSession || !/^[a-f0-9]{32}$/.test(claims.a) || !/^[a-f0-9]{32}$/.test(claims.d) ||
        !Number.isInteger(claims.i) || !Number.isInteger(claims.e) || claims.i > now || claims.e <= now || claims.e - claims.i > 60) return null;
    return claims;
  } catch {
    return null;
  }
}

function originRequest(request, env, host, path = null) {
  const origin = new URL(env.ORIGIN_URL);
  const incoming = new URL(request.url);
  origin.pathname = path || incoming.pathname;
  origin.search = path ? '' : incoming.search;
  const upstream = new Request(origin, request);
  upstream.headers.delete('X-Cellapp-Original-Host');
  upstream.headers.delete('X-Cellapp-Edge-Secret');
  upstream.headers.delete('X-Forwarded-Host');
  upstream.headers.set('X-Cellapp-Original-Host', host);
  upstream.headers.set('X-Cellapp-Edge-Secret', env.EDGE_ORIGIN_SECRET);
  return upstream;
}

async function forward(request, env, host) {
  try {
    const result = await fetch(originRequest(request, env, host), { cache: 'no-store', redirect: 'manual' });
    const response = new Response(result.body, result);
    response.headers.set('Cache-Control', 'private, no-store');
    return response;
  } catch {
    return new Response('Origin unavailable', { status: 503 });
  }
}

function dynamicRoute(path) {
  return path.startsWith('/_cellapp/') || path.startsWith('/_hosting/') || path.startsWith('/rest/v1') ||
    path.startsWith('/auth/v1') || path.startsWith('/storage/v1') || path.startsWith('/functions/v1') || path.startsWith('/realtime/v1');
}

function mcpControlPath(path) {
  return path === '/mcp' || path === '/oauth/authorize' || path === '/oauth/token' || path === '/oauth/register' || path === '/oauth/revoke' ||
    path === '/.well-known/oauth-protected-resource' || path === '/.well-known/oauth-protected-resource/mcp' || path === '/.well-known/oauth-authorization-server';
}

function publicationMutation(request, path, host, env) {
  return host === env.CONTROL_HOST && /^(POST|PUT|PATCH|DELETE)$/.test(request.method) &&
    /^\/(?:api\/control\/v1\/)?apps\/[a-f0-9]{32}\/deployments(?:\/|$)/.test(path);
}

function safePath(pathname) {
  let value;
  try { value = decodeURIComponent(pathname).slice(1) || 'index.html'; } catch { return null; }
  if (value.length > 512 || /[\\%?#\x00-\x1f\x7f]/.test(value)) return null;
  for (const part of value.split('/')) {
    const lower = part.toLowerCase();
    if (!part || part.startsWith('.') || ['_hosting', 'node_modules', 'credentials', 'id_rsa', 'id_ed25519'].includes(lower) || lower.startsWith('credentials.') ||
      ['.pem', '.key', '.p12', '.pfx'].some(suffix => lower.endsWith(suffix))) return null;
  }
  return value;
}

function contentType(path) {
  const ext = path.slice(path.lastIndexOf('.')).toLowerCase();
  return ({ '.html': 'text/html; charset=utf-8', '.css': 'text/css; charset=utf-8', '.js': 'text/javascript; charset=utf-8',
    '.mjs': 'text/javascript; charset=utf-8', '.json': 'application/json', '.svg': 'image/svg+xml', '.png': 'image/png',
    '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg', '.webp': 'image/webp', '.gif': 'image/gif', '.ico': 'image/vnd.microsoft.icon',
    '.woff': 'font/woff', '.woff2': 'font/woff2', '.txt': 'text/plain; charset=utf-8' })[ext] || 'application/octet-stream';
}

async function releaseFile(request, env, claims) {
  const incoming = new URL(request.url);
  let requested = safePath(incoming.pathname);
  if (!requested) return new Response('Not found', { status: 404 });
  const manifestObject = await env.R2_BUCKET.get(`releases/${claims.d}/_cellapp_manifest.json`);
  if (!manifestObject) {
    event('r2_manifest_missing', 503);
    return new Response('Release unavailable', { status: 503 });
  }
  let manifest;
  try { manifest = await manifestObject.json(); } catch {
    event('r2_manifest_invalid', 503);
    return new Response('Release unavailable', { status: 503 });
  }
  if (!Array.isArray(manifest.files)) {
    event('r2_manifest_invalid', 503);
    return new Response('Release unavailable', { status: 503 });
  }
  let file = manifest.files.find(entry => entry.path === requested);
  const navigation = request.method === 'GET' && (request.headers.get('Accept') || '').includes('text/html') &&
    (!request.headers.get('Sec-Fetch-Dest') || request.headers.get('Sec-Fetch-Dest') === 'document');
  if (!file && claims.s && navigation && !requested.split('/').at(-1).includes('.')) {
    requested = 'index.html';
    file = manifest.files.find(entry => entry.path === requested);
  }
  if (!file || !Number.isSafeInteger(file.size) || !/^[a-f0-9]{64}$/.test(file.sha256)) return new Response('Not found', { status: 404 });
  const digest = Array.from(await sha256(requested), byte => byte.toString(16).padStart(2, '0')).join('');
  const key = `releases/${claims.d}/${digest}`;
  const object = request.method === 'HEAD' ? await env.R2_BUCKET.head(key) : await env.R2_BUCKET.get(key);
  if (!object) {
    event('r2_object_missing', 404);
    return new Response('Not found', { status: 404 });
  }
  const etag = `"${file.sha256}"`;
  const headers = { 'Content-Type': contentType(requested), 'Cache-Control': 'private, no-store', ETag: etag,
    'Content-Length': String(file.size), 'X-Content-Type-Options': 'nosniff' };
  if (request.headers.get('If-None-Match') === etag) return new Response(null, { status: 304, headers });
  if (request.headers.has('If-Match') && request.headers.get('If-Match') !== etag) return new Response(null, { status: 412, headers });
  return new Response(request.method === 'HEAD' ? null : object.body, { status: 200, headers });
}

export default {
  async fetch(request, env) {
    const host = originalHost(request, env);
    if (!host || !env.EDGE_ORIGIN_SECRET || !env.ORIGIN_URL) return new Response('Not found', { status: 404 });
    const pathname = new URL(request.url).pathname;
    if (host === env.CONTROL_HOST && !controlRoute(pathname)) return new Response('Not found', { status: 404 });
    if (host !== env.CONTROL_HOST && mcpControlPath(pathname)) return new Response('Not found', { status: 404 });
    if (pathname === '/_cellapp/edge-grant') return new Response('Not found', { status: 404 });
    if (env.PUBLICATION_PAUSED === 'true' && publicationMutation(request, pathname, host, env)) {
      return Response.json({ error: 'publication_paused', message: 'Application publication is temporarily paused' },
        { status: 503, headers: { 'Cache-Control': 'private, no-store' } });
    }
    if (host !== env.CONTROL_HOST && env.R2_BUCKET) {
      const source = request.headers.get('CF-Connecting-IP');
      if (!source || !env.EDGE_RATE_LIMITER) return new Response('Edge protection unavailable', { status: 503 });
      try {
        const { success } = await env.EDGE_RATE_LIMITER.limit({ key: source });
        if (!success) {
          event('edge_rate_limited', 429);
          return new Response('Too many requests', { status: 429, headers: { 'Cache-Control': 'private, no-store' } });
        }
      } catch {
        return new Response('Edge protection unavailable', { status: 503 });
      }
    }
    if (host === env.CONTROL_HOST || request.method !== 'GET' && request.method !== 'HEAD' || dynamicRoute(pathname)) return forward(request, env, host);
    if (env.EDGE_STATIC_MODE === 'origin') return forward(request, env, host);
    if (env.EDGE_STATIC_MODE && env.EDGE_STATIC_MODE !== 'r2') return new Response('Edge configuration unavailable', { status: 503 });
    if (Boolean(env.R2_BUCKET) !== Boolean(env.EDGE_GRANT_PUBLIC_KEYS)) return new Response('Edge configuration unavailable', { status: 503 });
    if (!env.R2_BUCKET) return forward(request, env, host);
    const session = cookieValue(request, appCookie);
    let value = cookieValue(request, grantCookie);
    let claims = await verifyGrant(value, host, session, env);
    let renewed = false;
    if (!claims) {
      let result;
      try {
        result = await fetch(originRequest(request, env, host, '/_cellapp/edge-grant'), { cache: 'no-store', redirect: 'manual' });
      } catch {
        return new Response('Authorization unavailable', { status: 503 });
      }
      if (result.status !== 200) {
        event(result.status === 429 ? 'visitor_limit' : 'grant_denied', result.status);
        const navigation = request.method === 'GET' && (request.headers.get('Accept') || '').includes('text/html');
        if (navigation && (result.status === 401 || result.status === 403)) return forward(request, env, host);
        return new Response(result.body, { status: result.status, headers: { 'Cache-Control': 'private, no-store' } });
      }
      try { value = (await result.json()).grant; } catch { return new Response('Authorization unavailable', { status: 503 }); }
      claims = await verifyGrant(value, host, session, env);
      if (!claims) {
        event('grant_invalid', 503);
        return new Response('Authorization unavailable', { status: 503 });
      }
      event('grant_issued', 200);
      renewed = true;
    }
    try {
      const response = await releaseFile(request, env, claims);
      if (renewed) response.headers.append('Set-Cookie', `${grantCookie}=${value}; Path=/; Secure; HttpOnly; SameSite=Lax; Max-Age=${Math.max(0, claims.e - Math.floor(Date.now() / 1000))}`);
      return response;
    } catch {
      event('r2_error', 503);
      return new Response('Storage unavailable', { status: 503 });
    }
  },
};
