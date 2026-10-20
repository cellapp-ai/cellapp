export type Session = { ownerId: string; authMode: 'dev' | 'github' };
export type Profile = { displayName: string; userSlug: string };
export type Application = { id: string; name: string; namespace: string; slug: string; aliasUrl: string; activeDeployment: string | null; suspended: boolean; url: string; visibility: 'public' | 'restricted'; dataState: string };
export type Release = { id: string; status: string; createdAt: string; publishedAt: string | null; spa: boolean; fileCount: number; bytes: number };
export type ApplicationDetail = Application & { release: Release | null };
export type Credential = { id: string; createdAt: string; expiresAt: string };
export type Access = { requests: { userId: string; status: 'pending' | 'approved' | 'denied'; requestedAt: string }[]; grants: { userId: string; source: string; grantedAt: string }[] };
export type DataStatus = { state: string; version: number; digest: string };
export type TableStructure = {
  name: string;
  columns: { name: string; type: 'text' | 'uuid' | 'bigint' | 'boolean' | 'timestamptz' | 'jsonb'; nullable: boolean }[];
  primaryKey: string[];
  indexes: { columns: string[]; unique: boolean }[];
  policies: { select: string; insert: string; update: string; delete: string; ownerColumn: string };
};
export type DataStructure = { state: 'ready' | 'pending' | 'error'; version: number; tables: TableStructure[] };

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
export function profile(value: unknown): Profile {
  const data = record(value);
  const displayName = text(data.displayName);
  if (Array.from(displayName).length > 80 || displayName !== displayName.trim()) throw new Error('Unexpected profile response.');
  const userSlug = text(data.userSlug);
  if (userSlug !== '' && !/^[a-z][a-z0-9-]{0,62}$/.test(userSlug)) throw new Error('Unexpected user slug.');
  return { displayName, userSlug };
}
export function application(value: unknown): Application {
  const data = record(value);
  const url = new URL(text(data.url));
  if (url.protocol !== 'https:' || url.username || url.password || url.search || url.hash) throw new Error('Unexpected application address.');
  const visibility = data.visibility;
  if (visibility !== 'public' && visibility !== 'restricted') throw new Error('Unexpected access setting.');
  const namespace = text(data.namespace);
  const slug = text(data.slug);
  const aliasUrl = new URL(text(data.aliasUrl));
  if (!/^[a-z][a-z0-9-]{0,62}$/.test(namespace) || !/^[a-z][a-z0-9-]{0,62}$/.test(slug) ||
      aliasUrl.protocol !== 'https:' || aliasUrl.username || aliasUrl.password || aliasUrl.search || aliasUrl.hash) throw new Error('Unexpected application alias.');
  return { id: identifier(data.id), name: text(data.name), namespace, slug, aliasUrl: aliasUrl.href, activeDeployment: data.activeDeployment === null ? null : identifier(data.activeDeployment), suspended: boolean(data.suspended), url: url.href, visibility, dataState: text(data.dataState) };
}
export function access(value: unknown): Access {
  const data = record(value);
  if (!Array.isArray(data.requests) || !Array.isArray(data.grants)) throw new Error('Unexpected access list.');
  return {
    requests: data.requests.map(value => {
      const item = record(value);
      if (item.status !== 'pending' && item.status !== 'approved' && item.status !== 'denied') throw new Error('Unexpected request status.');
      return { userId: identifier(item.userId), status: item.status, requestedAt: date(item.requestedAt) };
    }),
    grants: data.grants.map(value => {
      const item = record(value);
      return { userId: identifier(item.userId), source: text(item.source), grantedAt: date(item.grantedAt) };
    }),
  };
}
export function dataStatus(value: unknown): DataStatus {
  const data = record(value);
  return { state: text(data.state), version: number(data.version), digest: text(data.digest) };
}
function keys(data: Record<string, unknown>, allowed: string[]) {
  if (Object.keys(data).some(key => !allowed.includes(key))) throw new Error('Unexpected table structure response. Try again.');
}
function tableName(value: unknown): string {
  const name = text(value);
  if (!/^[a-z][a-z0-9_]{0,62}$/.test(name)) throw new Error('Unexpected table structure response. Try again.');
  return name;
}
function names(value: unknown): string[] {
  if (!Array.isArray(value) || value.length > 30) throw new Error('Unexpected table structure response. Try again.');
  return value.map(tableName);
}
export function dataStructure(value: unknown): DataStructure {
  const data = record(value);
  keys(data, ['state', 'version', 'tables']);
  if (data.state !== 'ready' && data.state !== 'pending' && data.state !== 'error') throw new Error('Unexpected table structure response. Try again.');
  const version = number(data.version);
  if (!Array.isArray(data.tables) || data.tables.length > 20 || (version === 0 && data.tables.length > 0)) throw new Error('Unexpected table structure response. Try again.');
  const tables = data.tables.map((value): TableStructure => {
    const table = record(value);
    keys(table, ['name', 'columns', 'primaryKey', 'indexes', 'policies']);
    if (!Array.isArray(table.columns) || table.columns.length < 1 || table.columns.length > 30 || !Array.isArray(table.indexes) || table.indexes.length > 10) throw new Error('Unexpected table structure response. Try again.');
    const columns = table.columns.map((value): TableStructure['columns'][number] => {
      const column = record(value);
      keys(column, ['name', 'type', 'nullable']);
      const type = column.type;
      if (type !== 'text' && type !== 'uuid' && type !== 'bigint' && type !== 'boolean' && type !== 'timestamptz' && type !== 'jsonb') throw new Error('Unexpected table structure response. Try again.');
      if (column.nullable !== undefined && typeof column.nullable !== 'boolean') throw new Error('Unexpected table structure response. Try again.');
      return { name: tableName(column.name), type, nullable: column.nullable === true };
    });
    const primaryKey = names(table.primaryKey);
    if (primaryKey.length < 1 || primaryKey.length > 3 || primaryKey.some(name => !columns.some(column => column.name === name))) throw new Error('Unexpected table structure response. Try again.');
    const indexes = table.indexes.map(value => {
      const index = record(value);
      keys(index, ['columns', 'unique']);
      const indexColumns = names(index.columns);
      if (indexColumns.length < 1 || indexColumns.length > 3 || indexColumns.some(name => !columns.some(column => column.name === name)) || (index.unique !== undefined && typeof index.unique !== 'boolean')) throw new Error('Unexpected table structure response. Try again.');
      return { columns: indexColumns, unique: index.unique === true };
    });
    const policy = record(table.policies);
    keys(policy, ['select', 'insert', 'update', 'delete', 'ownerColumn']);
    const mode = (value: unknown) => {
      if (value === undefined || value === 'none') return 'none';
      if (value === 'all-admitted-users' || value === 'own-rows') return value;
      throw new Error('Unexpected table structure response. Try again.');
    };
    const ownerColumn = policy.ownerColumn === undefined ? '' : tableName(policy.ownerColumn);
    const policies = { select: mode(policy.select), insert: mode(policy.insert), update: mode(policy.update), delete: mode(policy.delete), ownerColumn };
    if (ownerColumn && !columns.some(column => column.name === ownerColumn)) throw new Error('Unexpected table structure response. Try again.');
    return { name: tableName(table.name), columns, primaryKey, indexes, policies };
  });
  if (new Set(tables.map(table => table.name)).size !== tables.length) throw new Error('Unexpected table structure response. Try again.');
  return { state: data.state, version, tables };
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
    if (path !== '/session') window.location.replace(`/dashboard/login?return=${encodeURIComponent(window.location.pathname + window.location.search)}`);
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
