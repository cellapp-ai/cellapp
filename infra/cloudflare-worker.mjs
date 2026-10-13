function originalHost(request, env) {
  const hostname = new URL(request.url).hostname;
  if (hostname === env.CONTROL_HOST) return hostname;
  const suffix = `.${env.APPS_DOMAIN}`;
  if (hostname.endsWith(suffix) && /^[a-z0-9-]+$/.test(hostname.slice(0, -suffix.length))) return hostname;
  if (env.PROBE_HOST && hostname.endsWith('.workers.dev')) return env.PROBE_HOST;
  return null;
}

export default {
  async fetch(request, env) {
    const host = originalHost(request, env);
    if (!host || !env.EDGE_ORIGIN_SECRET || !env.ORIGIN_URL) {
      return new Response('Not found', { status: 404 });
    }
    const origin = new URL(env.ORIGIN_URL);
    const incoming = new URL(request.url);
    origin.pathname = incoming.pathname;
    origin.search = incoming.search;

    const upstream = new Request(origin, request);
    upstream.headers.delete('X-Cellapp-Original-Host');
    upstream.headers.delete('X-Cellapp-Edge-Secret');
    upstream.headers.delete('X-Forwarded-Host');
    upstream.headers.set('X-Cellapp-Original-Host', host);
    upstream.headers.set('X-Cellapp-Edge-Secret', env.EDGE_ORIGIN_SECRET);
    const result = await fetch(upstream, { cache: 'no-store', redirect: 'manual' });
    const response = new Response(result.body, result);
    response.headers.set('Cache-Control', 'private, no-store');
    return response;
  },
};
