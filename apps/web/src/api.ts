export type Session = { ownerId: string; authMode: 'dev' | 'github' };
export type Application = { id: string; name: string; activeDeployment: string | null; suspended: boolean; url: string };
export type Release = { id: string; status: string; createdAt: string; publishedAt: string | null; spa: boolean; fileCount: number; bytes: number };
export type ApplicationDetail = Application & { release: Release | null };
export type Credential = { id: string; createdAt: string; expiresAt: string };

export class RequestError extends Error {
  constructor(public status: number, message: string) { super(message); }
}
function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Unexpected response. Try again.');
  return Object.fromEntries(Object.entries(value));
}
function text(value: unknown): string {
  if (typeof value !== 'string') throw new Error('Unexpected response. Try again.');
  return value;
}
function boolean(value: unknown): boolean {
  if (typeof value !== 'boolean') throw new Error('Unexpected response. Try again.');
  return value;
}
function number(value: unknown): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0) throw new Error('Unexpected response. Try again.');
  return value;
}
function date(value: unknown): string {
  const result = text(value);
  if (!Number.isFinite(Date.parse(result))) throw new Error('Unexpected response. Try again.');
  return result;
}
function identifier(value: unknown): string {
  const result = text(value);
  if (!/^[a-f0-9]{32}$/.test(result)) throw new Error('Unexpected response. Try again.');
  return result;
}
export function session(value: unknown): Session {
  const data = record(value);
  if (data.authMode !== 'dev' && data.authMode !== 'github') throw new Error('Unexpected response. Try again.');
  return { ownerId: identifier(data.ownerId), authMode: data.authMode };
}
export function application(value: unknown): Application {
  const data = record(value);
  const url = new URL(text(data.url));
  if (url.protocol !== 'https:' || url.username || url.password || url.search || url.hash) throw new Error('Unexpected application address.');
  return { id: identifier(data.id), name: text(data.name), activeDeployment: data.activeDeployment === null ? null : identifier(data.activeDeployment), suspended: boolean(data.suspended), url: url.href };
}
export function detail(value: unknown): ApplicationDetail {
  const data = record(value);
  const app = application(data);
  if (data.release === null) return { ...app, release: null };
  const release = record(data.release);
  return { ...app, release: { id: identifier(release.id), status: text(release.status), createdAt: date(release.createdAt), publishedAt: release.publishedAt === null ? null : date(release.publishedAt), spa: boolean(release.spa), fileCount: number(release.fileCount), bytes: number(release.bytes) } };
}
export function credential(value: unknown): Credential {
  const data = record(value);
  return { id: identifier(data.id), createdAt: date(data.createdAt), expiresAt: date(data.expiresAt) };
}
export function list<T>(parse: (value: unknown) => T): (value: unknown) => T[] {
  return (value) => {
    if (!Array.isArray(value)) throw new Error('Unexpected response. Try again.');
    return value.map(parse);
  };
}
export function keyResult(value: unknown): string {
  const key = text(record(value).key);
  if (!/^[a-f0-9]{64}$/.test(key)) throw new Error('Unexpected response. Reset the key again when ready.');
  return key;
}
export function success(field: string): (value: unknown) => void {
  return (value) => { if (record(value)[field] !== true) throw new Error('Unexpected response. Check the current state.'); };
}
export function decisionResult(value: unknown): 'approved' | 'denied' {
  const status = record(value).status;
  if (status !== 'approved' && status !== 'denied') throw new Error('Unexpected response. Check the authorization request.');
  return status;
}
export async function request<T>(path: string, parse: (value: unknown) => T, options: { method?: string; body?: unknown; signal?: AbortSignal } = {}): Promise<T> {
  const response = await fetch(`/api/console${path}`, {
    method: options.method ?? 'GET', credentials: 'same-origin', cache: 'no-store', signal: options.signal,
    headers: options.body === undefined ? { Accept: 'application/json' } : { Accept: 'application/json', 'Content-Type': 'application/json' },
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  if (response.status === 401) {
    // Reload discards every private component state, including a newly generated share key.
    if (path !== '/session') window.location.replace(`/login?return=${encodeURIComponent(window.location.pathname + window.location.search)}`);
    throw new RequestError(401, 'Sign in to continue.');
  }
  const value: unknown = await response.json();
  if (!response.ok) {
    const data = record(value);
    throw new RequestError(response.status, typeof data.message === 'string' ? data.message : 'Request could not be completed.');
  }
  return parse(value);
}
export function message(error: unknown): string {
  return error instanceof Error ? error.message : 'Request could not be completed. Try again.';
}
