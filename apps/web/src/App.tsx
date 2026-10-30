import { Fragment, useEffect, useRef, useState } from 'react';
import type { ComponentProps, FormEvent, MouseEvent, ReactNode } from 'react';
import { access, anonymousSettings, application, credential, dataStructure, decisionResult, detail, keyResult, list, message, modelCapabilities, modelConnection, modelGrant, modelUsage, profile, request, RequestError, session, success } from '@/api';
import type { ModelCapability, Session } from '@/api';

type Load<T> = { kind: 'loading' } | { kind: 'ready'; value: T } | { kind: 'error'; message: string };
function useResource<T>(path: string, parse: (value: unknown) => T) {
  const [state, setState] = useState<Load<T>>({ kind: 'loading' });
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    request(path, parse, { signal: controller.signal }).then(
      value => { if (!controller.signal.aborted) setState({ kind: 'ready', value }); },
      error => { if (!controller.signal.aborted) setState({ kind: 'error', message: message(error) }); },
    );
    return () => controller.abort();
  }, [path, parse, version]);
  return { state, reload: (preserve = false) => { if (!preserve) setState({ kind: 'loading' }); setVersion(value => value + 1); } };
}
const applications = list(application);
const credentials = list(credential);
const modelConnections = list(modelConnection);
const modelGrants = list(modelGrant);
const modelUsages = list(modelUsage);
function date(value: string) { return new Date(value).toLocaleString('en-GB', { timeZone: 'UTC' }) + ' UTC'; }
function loginReturn(): string {
  const value = new URLSearchParams(window.location.search).get('return') ?? '/dashboard';
  const path = value.split('?')[0] ?? '';
  return ['/dashboard', '/dashboard/applications', '/dashboard/credentials', '/dashboard/model-connections', '/dashboard/local-setup', '/dashboard/device', '/dashboard/profile'].includes(path) || /^\/dashboard\/applications\/[a-f0-9]{32}$/.test(path) ? value : '/dashboard';
}
function Feedback({ children, error = false }: { children: ReactNode; error?: boolean }) {
  return <p className={error ? 'feedback error' : 'feedback'} role={error ? 'alert' : 'status'}>{children}</p>;
}
type Navigate = (path: string) => void;
function ConsoleLink({ href, navigate, onClick, ...props }: ComponentProps<'a'> & { href: string; navigate: Navigate }) {
  const follow = (event: MouseEvent<HTMLAnchorElement>) => {
    onClick?.(event);
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey ||
      (event.currentTarget.target !== '' && event.currentTarget.target !== '_self') || event.currentTarget.hasAttribute('download') ||
      !href.startsWith('/') || href.startsWith('//')) return;
    event.preventDefault();
    navigate(href);
  };
  return <a href={href} onClick={follow} {...props} />;
}
function focusContent() { requestAnimationFrame(() => document.getElementById('content')?.focus()); }
function LoadingError<T>({ state, retry }: { state: Load<T>; retry: () => void }) {
  if (state.kind === 'loading') return <Feedback>Loading…</Feedback>;
  if (state.kind === 'error') return <div className="empty"><Feedback error>{state.message}</Feedback><button onClick={retry}>Try again</button></div>;
  return null;
}
function Copy({ value, label }: { value: string; label: string }) {
  const [feedback, setFeedback] = useState('');
  return <div className="copy"><code>{value}</code><button onClick={() => {
    void (async () => { try { await navigator.clipboard.writeText(value); setFeedback('Copied.'); } catch { setFeedback('Copy unavailable. Select the text and copy it manually.'); } })();
  }}>{label}</button><Feedback>{feedback}</Feedback></div>;
}
function Confirm({ title, children, onClose, onConfirm, busy }: { title: string; children: ReactNode; onClose: () => void; onConfirm: () => void; busy: boolean }) {
  const element = useRef<HTMLDialogElement>(null);
  const previousFocus = useRef(document.activeElement);
  useEffect(() => {
    const dialog = element.current;
    const trigger = previousFocus.current;
    dialog?.showModal();
    return () => { dialog?.close(); queueMicrotask(() => { if (trigger instanceof HTMLElement && trigger.isConnected) trigger.focus(); }); };
  }, []);
  return <dialog ref={element} aria-labelledby="confirm-title" onCancel={event => { if (busy) event.preventDefault(); else onClose(); }}>
    <h2 id="confirm-title">{title}</h2><p>{children}</p><div className="actions"><button disabled={busy} onClick={onClose}>Cancel</button><button className="danger" disabled={busy} onClick={onConfirm}>{busy ? 'Working…' : 'Confirm'}</button></div>
  </dialog>;
}
function Applications({ navigate }: { navigate: Navigate }) {
  const { state, reload } = useResource('/apps', applications);
  return <><div className="page-heading"><p className="eyebrow">WORKSPACE / APPLICATIONS</p><div className="heading-line"><div><h1>Applications</h1><p>Manage the applications you publish with Cellapp.</p></div>{state.kind === 'ready' && <span className="count-label">{state.value.length} {state.value.length === 1 ? 'application' : 'applications'}</span>}</div></div>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No applications yet</h2><p>Ask your AI assistant to deploy this project with the Cellapp Skill. Your applications will appear here after deployment.</p><button onClick={() => reload()}>Refresh applications</button></section> : <section className="application-list" aria-label="Your applications"><div className="list-heading"><span>APPLICATION</span><span>STATUS</span><span>ACTIONS</span></div>{state.value.map(app => {
      const live = !app.suspended && !!app.activeDeployment;
      return <article className="application-row" key={app.id}><div className="application-identity"><ConsoleLink href={`/dashboard/applications/${app.id}`} navigate={navigate}><h2>{app.name}</h2></ConsoleLink><p className="app-handle">@{app.namespace}/{app.slug}</p><p className="app-url">{app.url}</p></div><span className={`badge ${app.suspended ? 'is-suspended' : live ? 'is-published' : 'is-unpublished'}`}>{app.suspended ? 'Suspended' : live ? 'Published' : 'Not published'}</span><div className="row-actions"><ConsoleLink className="manage-link" href={`/dashboard/applications/${app.id}`} navigate={navigate}>Manage <span aria-hidden="true">→</span></ConsoleLink>{live && <a className="open-link" href={app.url} target="_blank" rel="noopener noreferrer">Open app <span aria-hidden="true">↗</span></a>}</div></article>;
    })}</section>)}
    {state.kind === 'ready' && state.value.length > 0 && <button className="subtle-action" onClick={() => reload()}>Refresh applications</button>}
  </>;
}
function SlugEditor({ label, value, maxLength, save }: { label: string; value: string; maxLength: number; save: (next: string) => Promise<void> }) {
  const [draft, setDraft] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const next = draft ?? value;
  const valid = /^[a-z](?:[a-z0-9-]*[a-z0-9])?$/.test(next) && next.length >= (label === 'User slug' ? 3 : 1) && next.length <= maxLength;
  return <form className="profile-form" onSubmit={event => { event.preventDefault(); if (!valid || busy || next === value) return; setBusy(true); setError(''); setFeedback(''); void save(next).then(() => { setDraft(null); setFeedback(`${label} saved.`); }, caught => { setError(`${message(caught)} Refresh to check the current address.`); }).finally(() => setBusy(false)); }}>
    <label htmlFor={label === 'User slug' ? 'user-slug' : 'app-slug'}>{label}</label>
    <input id={label === 'User slug' ? 'user-slug' : 'app-slug'} value={next} disabled={busy} onChange={event => { setDraft(event.target.value); setFeedback(''); setError(''); }} autoComplete="off" spellCheck={false} />
    <p>Use lowercase letters, digits and hyphens. Reserved names are unavailable. {label === 'User slug' ? 'At least 3 characters.' : `At most ${maxLength} characters with your current user slug.`}</p>
    <div className="actions"><button className="primary" type="submit" disabled={busy || !valid || next === value}>{busy ? 'Saving…' : `Save ${label.toLowerCase()}`}</button><button type="button" disabled={busy || next === value} onClick={() => setDraft(null)}>{`Discard ${label.toLowerCase()} changes`}</button></div>
    {feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
  </form>;
}
function AppSlugSettings({ id, slug, userSlug, reload }: { id: string; slug: string; userSlug: string; reload: (preserve?: boolean) => void }) {
  return <SlugEditor label="Application slug" value={slug} maxLength={62 - userSlug.length} save={async next => { await request(`/apps/${id}/slug`, application, { method: 'PUT', body: { slug: next } }); reload(true); }} />;
}
function ApplicationTables({ state, reload }: { state: Load<ReturnType<typeof dataStructure>>; reload: () => void }) {
  const [selectedName, setSelectedName] = useState('');
  const selected = state.kind === 'ready' ? state.value.tables.find(table => table.name === selectedName) ?? state.value.tables[0] : undefined;
  return <section className="panel tables-panel" id="tables" aria-labelledby="tables-title"><div className="section-heading"><div><p className="eyebrow">DATA</p><h2 id="tables-title">Application tables</h2></div><button className="subtle-action" onClick={reload}>Refresh structure</button></div>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && <><p className="section-summary">Applied table version {state.value.version}. Structure only; table records are not shown here.</p>{state.value.state !== 'ready' && <p className="notice" role="status">Data service is {state.value.state}. This applied definition may not be available to the application yet.</p>}
      {state.value.tables.length === 0 ? <div className="inline-empty"><h3>No tables defined</h3><p>Tables appear here after an applied data definition. Your AI assistant can manage table definitions through the Cellapp Skill.</p></div> : <div className="tables-layout"><div className="table-picker" aria-label="Tables">{state.value.tables.map(table => <button className={selected?.name === table.name ? 'table-choice is-selected' : 'table-choice'} key={table.name} aria-pressed={selected?.name === table.name} onClick={() => setSelectedName(table.name)}><code>{table.name}</code><span>{table.columns.length} {table.columns.length === 1 ? 'column' : 'columns'}</span></button>)}</div>{selected && <div className="table-detail"><h3><code>{selected.name}</code></h3><div className="table-scroll" role="region" tabIndex={0} aria-label={`Columns in ${selected.name}`}><table><caption>Columns in {selected.name}</caption><thead><tr><th scope="col">Column</th><th scope="col">Type</th><th scope="col">Nullable</th><th scope="col">Key</th></tr></thead><tbody>{selected.columns.map(column => <tr key={column.name}><th scope="row"><code>{column.name}</code></th><td><code>{column.type}</code></td><td>{column.nullable ? 'Yes' : 'No'}</td><td>{selected.primaryKey.includes(column.name) ? 'Primary' : '—'}</td></tr>)}</tbody></table></div><div className="structure-meta"><div><h4>Indexes</h4>{selected.indexes.length === 0 ? <p>None</p> : <ul>{selected.indexes.map((index, position) => <li key={position}><code>{index.columns.join(', ')}</code>{index.unique ? ' · Unique' : ''}</li>)}</ul>}</div><div><h4>Application access</h4><dl>{(['select', 'insert', 'update', 'delete'] as const).map(action => <Fragment key={action}><dt>{action}</dt><dd>{selected.policies[action] === 'own-rows' ? `Own rows${selected.policies.ownerColumn ? ` · ${selected.policies.ownerColumn}` : ''}` : selected.policies[action] === 'all-admitted-users' ? 'All admitted users' : 'No access'}</dd></Fragment>)}</dl><h4>Anonymous visitors</h4><dl><dt>Read</dt><dd>{selected.policies.anonymousSelect === 'all' ? 'Allowed' : 'No access'}</dd><dt>Submit</dt><dd>{selected.policies.anonymousInsert === 'all' ? 'Allowed' : 'No access'}</dd></dl></div></div></div>}</div>}
    </>}
  </section>;
}
function AnonymousAccess({ id, visibility }: { id: string; visibility: 'public' | 'restricted' }) {
  const { state, reload } = useResource(`/apps/${id}/anonymous`, anonymousSettings);
  const [draft, setDraft] = useState<{ pages: boolean; read: boolean; write: boolean } | null>(null);
  const [confirm, setConfirm] = useState(false);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const saved = state.kind === 'ready' ? state.value : null;
  const current = draft ?? saved;
  const save = async () => {
    if (!draft || busy) return;
    setBusy(true); setError(''); setFeedback('');
    try {
      await request(`/apps/${id}/anonymous`, anonymousSettings, { method: 'PUT', body: draft });
      setFeedback('Anonymous access updated.'); setDraft(null); reload(true);
    } catch (caught) {
      setError(`${message(caught)} Refresh to confirm the current access settings.`);
      setDraft(null);
      reload(true);
    } finally { setBusy(false); setConfirm(false); }
  };
  return <section className="panel" aria-labelledby="anonymous-access-title"><div className="section-heading"><div><h3 id="anonymous-access-title">Anonymous access</h3><p>Choose what visitors can open without signing in.</p></div><button className="subtle-action" onClick={() => { setDraft(null); reload(true); }}>Refresh</button></div>
    <LoadingError state={state} retry={reload} />
    {current && <><fieldset disabled={busy || visibility !== 'public'}><legend>Visitor permissions</legend>
      <label><input type="checkbox" checked={current.pages} onChange={event => setDraft({ pages: event.target.checked, read: event.target.checked ? current.read : false, write: event.target.checked ? current.write : false })} /> Open published pages</label>
      <label><input type="checkbox" checked={current.read} disabled={!current.pages} onChange={event => setDraft({ pages: current.pages, read: event.target.checked, write: current.write })} /> Read permitted tables</label>
      <label><input type="checkbox" checked={current.write} disabled={!current.pages} onChange={event => setDraft({ pages: current.pages, read: current.read, write: event.target.checked })} /> Submit to permitted tables</label>
    </fieldset><p role="status">{visibility !== 'public' ? 'Make this application public to enable anonymous access.' : saved?.write && !saved.operatorWritesEnabled ? 'Anonymous submissions are configured but disabled by operations.' : saved?.effectiveWrite ? 'Anonymous submissions are active for tables that allow them.' : 'Anonymous submissions are off.'}</p>
      <button disabled={busy || visibility !== 'public' || !draft || draft.pages === saved?.pages && draft.read === saved?.read && draft.write === saved?.write} onClick={() => setConfirm(true)}>Save anonymous access</button></>}
    {feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {confirm && <Confirm title="Change anonymous access?" busy={busy} onClose={() => setConfirm(false)} onConfirm={() => { void save(); }}>Visitors without an account will receive the selected page and data permissions. Table policies and the operations switch still apply.</Confirm>}
  </section>;
}
function ApplicationPage({ id, navigate }: { id: string; navigate: Navigate }) {
  const { state, reload } = useResource(`/apps/${id}`, detail);
  const accessState = useResource(`/apps/${id}/access`, access);
  const structureState = useResource(`/apps/${id}/data/structure`, dataStructure);
  const [operation, setOperation] = useState<'key' | 'delete' | null>(null);
  const [accessAction, setAccessAction] = useState<{ path: string; title: string; explanation: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [newKey, setNewKey] = useState('');
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const perform = async () => {
    if (busy) return;
    setBusy(true); setError(''); setFeedback(''); setNewKey('');
    try {
      if (operation === 'key') {
        const key = await request(`/apps/${id}/key`, keyResult, { method: 'POST' });
        setNewKey(key); setFeedback('Access key rotated. Existing user grants remain active.');
        setOperation(null); reload();
      } else {
        await request(`/apps/${id}`, success('deleted'), { method: 'DELETE' });
        navigate('/dashboard/applications');
      }
    } catch (caught) {
      setOperation(null);
      setError(`${message(caught)} The result may already have taken effect. Refresh to check, or reset the key again when ready.`);
    } finally { setBusy(false); }
  };
  const performAccess = async () => {
    if (!accessAction || busy) return;
    setBusy(true); setError(''); setFeedback('');
    try {
      await request(accessAction.path, value => value, { method: 'POST' });
      setFeedback('Access updated.'); reload(); accessState.reload();
    } catch (caught) {
      setError(`${message(caught)} Refresh access to check the current state.`);
    } finally { setBusy(false); setAccessAction(null); }
  };
  return <><ConsoleLink className="back" href="/dashboard/applications" navigate={navigate}>← Applications</ConsoleLink><LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && <><div className="page-heading detail-heading"><p className="eyebrow">WORKSPACE / APPLICATIONS / DETAIL</p><div className="heading-line"><div><h1>{state.value.name}</h1><p><code>@{state.value.namespace}/{state.value.slug}</code></p></div><span className={`badge ${state.value.suspended ? 'is-suspended' : state.value.release ? 'is-published' : 'is-unpublished'}`}>{state.value.suspended ? 'Suspended' : state.value.release ? 'Published' : 'Not published'}</span></div><div className="heading-actions"><a className="app-address" href={state.value.url} target="_blank" rel="noopener noreferrer">{state.value.url}</a>{!state.value.suspended && state.value.release && <a className="primary button" href={state.value.url} target="_blank" rel="noopener noreferrer">Open application ↗</a>}</div></div>
      <nav className="section-nav" aria-label="Application sections"><a href="#release">Release</a><a href="#tables">Tables</a><a href="#access">Access</a><a href="#settings">Settings</a></nav>
      <section className="panel" id="release"><div className="section-heading"><div><p className="eyebrow">DEPLOYMENT</p><h2>Current release</h2></div></div>{state.value.release ? <dl><dt>Release</dt><dd><code>{state.value.release.id}</code></dd><dt>Published</dt><dd>{state.value.release.publishedAt ? date(state.value.release.publishedAt) : 'Not published'}</dd><dt>Files</dt><dd>{state.value.release.fileCount}</dd><dt>Size</dt><dd>{state.value.release.bytes.toLocaleString('en-GB')} bytes</dd><dt>Navigation</dt><dd>{state.value.release.spa ? 'Single page application' : 'Static pages'}</dd></dl> : <p>No release has been published. Ask your AI assistant to deploy this project with the Cellapp Skill.</p>}</section>
    </>}
    <ApplicationTables state={structureState.state} reload={structureState.reload} />
    <section className="panel" id="access"><div className="section-heading"><div><p className="eyebrow">PEOPLE & PERMISSIONS</p><h2>Access</h2></div></div>{state.kind === 'ready' && <><dl><dt>Visibility</dt><dd>{state.value.visibility === 'public' ? 'Public to signed-in users' : 'Restricted'}</dd></dl><p>Visitors sign in with Cellapp. {state.value.visibility === 'public' ? 'Any signed-in user can access this application.' : 'Restricted access requires approval or an access key.'}</p><button disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/visibility/${state.value.visibility === 'public' ? 'restricted' : 'public'}`, title: state.value.visibility === 'public' ? 'Restrict access?' : 'Make public?', explanation: state.value.visibility === 'public' ? 'Users without an explicit grant will lose access immediately.' : 'Every signed-in Cellapp user will be able to open this application.' })}>{state.value.visibility === 'public' ? 'Restrict access' : 'Make public'}</button></>}</section>
    {state.kind === 'ready' && <AnonymousAccess id={id} visibility={state.value.visibility} />}
    <section className="panel"><h2>Access requests</h2><LoadingError state={accessState.state} retry={accessState.reload} />{accessState.state.kind === 'ready' && (accessState.state.value.requests.filter(item => item.status === 'pending').length === 0 ? <p>No pending requests.</p> : accessState.state.value.requests.filter(item => item.status === 'pending').map(item => <div className="access-row" key={item.userId}><div><code>{item.userId}</code><p>Requested {date(item.requestedAt)}</p></div><div className="actions"><button disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/requests/${item.userId}/approve`, title: 'Approve access?', explanation: 'This user will be able to open the application until you revoke their grant.' })}>Approve</button><button disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/requests/${item.userId}/deny`, title: 'Deny request?', explanation: 'This request will be marked denied.' })}>Deny</button></div></div>))}</section>
    <section className="panel"><h2>Granted users</h2>{accessState.state.kind === 'ready' && (accessState.state.value.grants.length === 0 ? <p>No user grants.</p> : accessState.state.value.grants.map(item => <div className="access-row" key={item.userId}><div><code>{item.userId}</code><p>{item.source} · {date(item.grantedAt)}</p></div><button className="danger" disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/grants/${item.userId}/revoke`, title: 'Revoke access?', explanation: 'This user will lose access immediately unless the application is public.' })}>Revoke</button></div>))}</section>
    {state.kind === 'ready' && <section className="panel" id="settings"><div className="section-heading"><div><p className="eyebrow">CONFIGURATION</p><h2>Settings</h2></div></div><h3>Application address</h3><Copy value={state.value.aliasUrl} label="Copy share link" /><Copy value={state.value.url} label="Copy address" /><p>Address format: <code>app-slug-user-slug</code>. Existing addresses stay assigned to this application.</p><AppSlugSettings id={id} slug={state.value.slug} userSlug={state.value.namespace} reload={reload} /><div className="setting-actions"><h3>Lifecycle</h3><p>Key rotation stops the old key from granting new access. Deleting this application is permanent.</p><div className="actions"><button disabled={busy} onClick={() => setOperation('key')}>Rotate access key</button><button className="danger" disabled={busy} onClick={() => setOperation('delete')}>Delete application</button><button disabled={busy} onClick={() => { setNewKey(''); reload(); accessState.reload(); structureState.reload(); }}>Refresh details</button></div></div></section>}
    {feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {newKey && <section className="panel"><h2>New access key</h2><p>Copy this key now. It will be cleared when you leave this page and cannot be retrieved later.</p><Copy value={newKey} label="Copy access key" /><button onClick={() => setNewKey('')}>Dismiss key</button></section>}
    {operation && <Confirm title={operation === 'key' ? 'Rotate access key?' : 'Delete application?'} busy={busy} onClose={() => setOperation(null)} onConfirm={() => { void perform(); }}>{operation === 'key' ? 'The previous key will stop granting new access. Existing user grants remain active.' : 'This permanently removes the application. Its address will stop working and it cannot receive more deployments.'}</Confirm>}
    {accessAction && <Confirm title={accessAction.title} busy={busy} onClose={() => setAccessAction(null)} onConfirm={() => { void performAccess(); }}>{accessAction.explanation}</Confirm>}
  </>;
}
function Credentials() {
  const { state, reload } = useResource('/credentials', credentials);
  const [selected, setSelected] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const revoke = async () => {
    if (!selected || busy) return;
    setBusy(true); setError(''); setFeedback('');
    try { await request(`/credentials/${selected}/revoke`, success('revoked'), { method: 'POST' }); setFeedback('Deployment credential revoked.'); reload(); }
    catch (caught) { setError(`${message(caught)} The result may already have taken effect. Refresh the list to check.`); }
    finally { setBusy(false); setSelected(null); }
  };
  return <><div className="page-heading"><p className="eyebrow">ACCESS / DEPLOYMENT CREDENTIALS</p><h1>Deployment credentials</h1><p>Review active access granted to local deployment tools.</p></div><LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No active credentials</h2><p>Authorize your local deployment tool when you next deploy.</p></section> : <div className="credential-list">{state.value.map(item => <section className="panel" key={item.id}><h2 className="credential-id">{item.id}</h2><dl><dt>Created</dt><dd>{date(item.createdAt)}</dd><dt>Expires</dt><dd>{date(item.expiresAt)}</dd></dl><button className="danger" disabled={busy} onClick={() => setSelected(item.id)}>Revoke</button></section>)}</div>)}
    <button disabled={busy} onClick={() => reload()}>Refresh credentials</button>{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {selected && <Confirm title="Revoke deployment access?" onClose={() => setSelected(null)} busy={busy} onConfirm={() => { void revoke(); }}>The tool using this credential will need to authorize again. Published applications remain available.</Confirm>}
  </>;
}
function CapabilityStatus({ label, result }: { label: string; result: ModelCapability | undefined }) {
  return <li><strong>{label}:</strong> {result === undefined ? 'Not tested' : result.supported ? 'Available' : `Unavailable (${result.error ?? 'test failed'})`}</li>;
}
function ModelAuthorizations() {
  const grants = useResource('/model-grants', modelGrants);
  const usage = useResource('/model-usage', modelUsages);
  const [selected, setSelected] = useState<{appId: string; connectionId: string} | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [feedback, setFeedback] = useState('');
  const revoke = async () => {
    if (!selected || busy) return;
    setBusy(true); setError(''); setFeedback('');
    try {
      await request(`/model-grants/${selected.appId}/${selected.connectionId}/revoke`, success('revoked'), {method: 'POST'});
      setFeedback('Authorization revoked. New model requests from this application will fail until you authorize it again.');
      grants.reload();
    } catch (caught) { setError(`${message(caught)} Refresh to check the current authorization.`); }
    finally { setBusy(false); setSelected(null); }
  };
  return <><section className="panel"><h2>Application authorizations</h2><p>You can authorize several provider connections for one application. The application's code can call any model on each authorized connection.</p>
    <LoadingError state={grants.state} retry={() => grants.reload()} />
    {grants.state.kind === 'ready' && (grants.state.value.length === 0 ? <p>No applications are authorized to use your models.</p> : <div className="credential-list">{grants.state.value.map(item => <article key={`${item.appId}:${item.connectionId}`} className="panel"><h3>{item.appName}</h3><p>Authorized provider: <strong>{item.connectionName}</strong></p><p>Authorized: {date(item.grantedAt)}</p><div className="actions"><a href={`/dashboard/model-consent?app=${item.appId}`}>Authorize another provider</a><button className="danger" disabled={busy} onClick={() => setSelected({appId: item.appId, connectionId: item.connectionId})}>Revoke</button></div></article>)}</div>)}
    <button disabled={busy} onClick={() => grants.reload()}>Refresh authorizations</button>{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
  </section><section className="panel"><h2>Model requests · last 30 days</h2><p>Token counts are shown only when a provider reported them. They are not a billing total.</p><LoadingError state={usage.state} retry={() => usage.reload()} />
    {usage.state.kind === 'ready' && (usage.state.value.length === 0 ? <p>No model requests in this period.</p> : <div className="credential-list">{usage.state.value.map(item => {
      const current = grants.state.kind === 'ready' ? grants.state.value.find(grant => grant.appId === item.appId && grant.connectionId === item.connectionId) : undefined;
      return <article className="panel" key={`${item.appId}:${item.connectionId}`}><h3>{current ? `${current.appName} · ${current.connectionName}` : `Application ${item.appId.slice(0, 8)} · connection ${item.connectionId.slice(0, 8)}`}</h3><p>{item.requests} requests · {item.succeeded} succeeded · {item.failed} failed · {item.cancelled} cancelled</p><p>Provider-reported tokens: {item.inputTokens} input, {item.outputTokens} output. {item.unknownUsage} {item.unknownUsage === 1 ? 'request has' : 'requests have'} unknown token usage.</p></article>;
    })}</div>)}<button onClick={() => usage.reload()}>Refresh usage</button>
  </section>{selected && <Confirm title="Revoke model access?" busy={busy} onClose={() => setSelected(null)} onConfirm={() => { void revoke(); }}>This application will need your authorization before its next model request.</Confirm>}</>;
}
function ModelConnections() {
  const { state, reload } = useResource('/model-connections', modelConnections);
  const [name, setName] = useState('');
  const [apiBase, setAPIBase] = useState('');
  const [modelId, setModelId] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [rotateId, setRotateId] = useState<string | null>(null);
  const [rotateKey, setRotateKey] = useState('');
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const create = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (busy) return;
    setBusy(true); setError(''); setFeedback('');
    try {
      await request('/model-connections', modelConnection, { method: 'POST', body: { name, apiBase, modelId, apiKey } });
      setName(''); setAPIBase(''); setModelId(''); setAPIKey(''); setFeedback('Connection saved. Run the capability test before using it.'); reload();
    } catch (caught) { setError(message(caught)); }
    finally { setBusy(false); }
  };
  const testConnection = async (id: string) => {
    if (busy) return;
    setBusy(true); setError(''); setFeedback('');
    try { const result = await request(`/model-connections/${id}/test`, modelCapabilities, { method: 'POST' }); setFeedback(result.text.supported || result.stream.supported || result.tools.supported ? 'Capability test finished. Review each result below.' : 'No tested capability is available. Check the connection and provider.'); reload(); }
    catch (caught) { setError(`${message(caught)} Refresh and try the test again.`); }
    finally { setBusy(false); }
  };
  const rotate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!rotateId || busy) return;
    setBusy(true); setError(''); setFeedback('');
    try { await request(`/model-connections/${rotateId}/key`, success('rotated'), { method: 'POST', body: { apiKey: rotateKey } }); setRotateId(null); setRotateKey(''); setFeedback('Key replaced. Run the capability test again.'); reload(); }
    catch (caught) { setError(`${message(caught)} Refresh to check whether the key changed.`); }
    finally { setBusy(false); }
  };
  const remove = async () => {
    if (!deleteId || busy) return;
    setBusy(true); setError(''); setFeedback('');
    try { await request(`/model-connections/${deleteId}`, success('deleted'), { method: 'DELETE' }); setFeedback('Connection deleted. New model requests cannot use it.'); reload(); }
    catch (caught) { setError(`${message(caught)} Refresh to check whether the connection was deleted.`); }
    finally { setBusy(false); setDeleteId(null); }
  };
  return <><div className="page-heading"><p className="eyebrow">ACCESS / MODEL CONNECTIONS</p><h1>Model connections</h1><p>Connect a cloud provider that supports OpenAI Chat Completions. Your API key stays private to your account.</p></div>
    <section className="panel"><h2>Add a connection</h2><form className="model-form" onSubmit={event => { void create(event); }}>
      <label htmlFor="model-name">Name</label><input id="model-name" value={name} onChange={event => setName(event.target.value)} maxLength={80} required disabled={busy} />
      <label htmlFor="model-base">HTTPS API base URL</label><input id="model-base" type="url" inputMode="url" placeholder="https://provider.example/v1" value={apiBase} onChange={event => setAPIBase(event.target.value)} required disabled={busy} />
      <label htmlFor="model-id">Model ID (optional fallback)</label><input id="model-id" value={modelId} onChange={event => setModelId(event.target.value)} disabled={busy} /><p>Leave blank to discover models from the provider. If discovery fails, enter a model ID manually.</p>
      <label htmlFor="model-key">Provider API key</label><input id="model-key" type="password" autoComplete="off" value={apiKey} onChange={event => setAPIKey(event.target.value)} required disabled={busy} />
      <button className="primary" type="submit" disabled={busy}>{busy ? 'Working…' : 'Save connection'}</button>
    </form></section>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No model connections</h2><p>Add your own cloud provider account above to get started.</p></section> : <div className="credential-list">{state.value.map(item => <section className="panel" key={item.id}><h2>{item.name}</h2><p className="model-address">{item.apiBase}</p><p>Test model: <code>{item.modelId}</code> (capabilities below apply to this model only)</p><ul className="model-capabilities"><CapabilityStatus label="Text" result={item.capabilities?.text} /><CapabilityStatus label="Streaming" result={item.capabilities?.stream} /><CapabilityStatus label="Tools" result={item.capabilities?.tools} /></ul><p>Last changed: {date(item.updatedAt)}</p><div className="actions"><button disabled={busy} onClick={() => { void testConnection(item.id); }}>{busy ? 'Working…' : 'Test capabilities'}</button><button disabled={busy} onClick={() => { setRotateId(item.id); setRotateKey(''); }}>Replace key</button><button className="danger" disabled={busy} onClick={() => setDeleteId(item.id)}>Delete</button></div>{rotateId === item.id && <form className="model-form" onSubmit={event => { void rotate(event); }}><label htmlFor={`rotate-${item.id}`}>New provider API key</label><input id={`rotate-${item.id}`} type="password" autoComplete="off" value={rotateKey} onChange={event => setRotateKey(event.target.value)} required disabled={busy} /><div className="actions"><button className="primary" type="submit" disabled={busy}>Replace key</button><button type="button" disabled={busy} onClick={() => { setRotateId(null); setRotateKey(''); }}>Cancel</button></div></form>}</section>)}</div>)}
    <button disabled={busy} onClick={() => reload()}>Refresh connections</button>{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    <ModelAuthorizations />
    {deleteId && <Confirm title="Delete model connection?" busy={busy} onClose={() => setDeleteId(null)} onConfirm={() => { void remove(); }}>Applications authorized to use this connection will no longer be able to send new model requests.</Confirm>}
  </>;
}
function ProfilePage({ owner, onNameChange }: { owner: Session; onNameChange: (name: string) => void }) {
  const { state, reload } = useResource('/profile', profile);
  const [draft, setDraft] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const saved = state.kind === 'ready' ? state.value.displayName || state.value.githubUsername : '';
  const name = draft ?? saved;
  const valid = name === name.trim() && Array.from(name).length <= 80 && !Array.from(name).some(character => { const code = character.codePointAt(0) ?? 0; return code < 32 || code === 127; });
  const save = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!valid || busy || state.kind !== 'ready') return;
    setBusy(true); setFeedback(''); setError('');
    try {
      const updated = await request('/profile', profile, { method: 'PUT', body: { displayName: name } });
      onNameChange(updated.displayName || updated.githubUsername);
      setDraft(null); reload(); setFeedback('Display name saved.');
    } catch (caught) {
      setError(`${message(caught)} Refresh the profile to check whether the change was saved.`);
    } finally { setBusy(false); }
  };
  return <><div className="page-heading"><p className="eyebrow">ACCOUNT / PROFILE</p><h1>Profile settings</h1><p>Manage the name shown in your Cellapp account.</p></div>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && <><section className="panel"><h2>Display name</h2><form className="profile-form" onSubmit={event => { void save(event); }}><label htmlFor="display-name">Name</label><input id="display-name" value={name} disabled={busy} onChange={event => { setDraft(event.target.value); setFeedback(''); setError(''); }} aria-describedby="display-name-help" /><p id="display-name-help">Optional, up to 80 characters. {state.value.githubUsername ? 'Leave blank to use your GitHub username.' : 'Leave blank to clear it.'}</p><div className="actions"><button className="primary" type="submit" disabled={busy || !valid || name === saved || (name === '' && state.value.displayName === '')}>{busy ? 'Saving…' : 'Save name'}</button><button type="button" disabled={busy || name === saved} onClick={() => { setDraft(null); setError(''); }}>Discard changes</button><button type="button" disabled={busy} onClick={() => { setDraft(null); setFeedback(''); setError(''); reload(); }}>Refresh profile</button></div>{!valid && <Feedback error>Remove leading or trailing spaces, control characters, and use no more than 80 characters.</Feedback>}{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}</form></section>
      <section className="panel"><h2>User slug</h2><p>Your application addresses end with this slug. {state.value.userSlug ? 'Changing it updates their current addresses; previous addresses remain assigned to the same applications.' : 'Your first application will use your GitHub username when available, or you can choose a slug now.'}</p><SlugEditor label="User slug" value={state.value.userSlug} maxLength={61} save={async next => { await request('/profile/slug', profile, { method: 'PUT', body: { userSlug: next } }); reload(true); }} /></section>
      <section className="panel"><h2>Account details</h2><dl><dt>Account ID</dt><dd><code>{owner.ownerId}</code></dd><dt>Sign-in mode</dt><dd>{owner.authMode === 'dev' ? 'Local development' : 'GitHub'}</dd>{state.value.githubUsername && <><dt>GitHub username</dt><dd>@{state.value.githubUsername}</dd></>}</dl><p>Your account ID is fixed. Changing your display name does not change application addresses.</p></section></>}
  </>;
}
function LocalSetup() {
  return <><div className="page-heading"><p className="eyebrow">WORKSPACE / LOCAL SETUP</p><h1>Local setup</h1><p>Use a coding agent already installed on your computer, or follow the same steps yourself.</p></div>
    <section className="panel"><h2>Check this computer</h2><p>Install the Cellapp CLI, then run <code>cellapp doctor</code>. To check a project, add <code>--project /absolute/project</code>. The diagnostic checks your local tools and connection without changing your files or signing in for you.</p><p>Choose an installed coding agent with <code>cellapp set agent codex</code>, <code>cellapp set agent claude</code>, or <code>cellapp set agent cursor</code>. Sign in through that agent’s own instructions.</p></section>
    <section className="panel"><h2>Build and publish</h2><p>Ask your coding agent to follow the Cellapp setup guide, or follow its manual steps. Authorize the Cellapp deployment tool in your browser when prompted, then build and deploy your static project on this computer.</p><p><a href="https://github.com/cellapp-ai/skills/blob/main/skills/cellapp-deploy/SKILL.md" target="_blank" rel="noopener noreferrer">Read the Cellapp setup and deployment guide</a></p></section>
  </>;
}
function Device() {
  const [code, setCode] = useState(new URLSearchParams(window.location.search).get('code') ?? '');
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState('');
  const [error, setError] = useState('');
  const decide = async (decision: 'approved' | 'denied') => {
    if (busy || !code.trim()) return;
    setBusy(true); setError('');
    try { const status = await request('/device/decision', decisionResult, { method: 'POST', body: { code, decision } }); setResult(status === 'approved' ? 'Deployment access approved. Return to your local tool to continue.' : 'Authorization denied. No deployment access was granted.'); }
    catch (caught) { setError(`${message(caught)} Check the code in your local tool. A submitted decision may already have taken effect.`); }
    finally { setBusy(false); }
  };
  return <><div className="page-heading"><p className="eyebrow">ACCESS / AUTHORIZE TOOL</p><h1>Authorize a deployment tool</h1><p>Only approve a request you just started on your own device.</p></div><section className="panel"><h2>Check your authorization code</h2><p>Approval gives the tool access to manage your applications. Compare this code with the one shown in your local tool.</p><label htmlFor="device-code">Authorization code</label><input id="device-code" value={code} maxLength={64} disabled={busy || !!result} onChange={event => setCode(event.target.value)} autoComplete="off" spellCheck={false} aria-describedby={error ? 'device-error' : undefined} /><div className="actions"><button className="primary" disabled={busy || !!result || !code.trim()} onClick={() => { void decide('approved'); }}>{busy ? 'Working…' : 'Approve access'}</button><button disabled={busy || !!result || !code.trim()} onClick={() => { void decide('denied'); }}>Deny access</button></div>{result && <Feedback>{result}</Feedback>}{error && <p id="device-error" className="feedback error" role="alert">{error}</p>}</section></>;
}
function Workspace({ owner }: { owner: Session }) {
  const [username, setUsername] = useState(owner.username);
  const [avatarFailed, setAvatarFailed] = useState(false);
  const [menu, setMenu] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [route, setRoute] = useState(window.location.pathname + window.location.search);
  const toggle = useRef<HTMLButtonElement>(null);
  const path = route.split('?')[0] ?? '';
  const navigate: Navigate = target => {
    if (target === window.location.pathname + window.location.search) { setMenu(false); return; }
    window.history.pushState(null, '', target);
    setRoute(target);
    setMenu(false);
    focusContent();
  };
  const logout = async () => {
    setBusy(true); setError('');
    try { await request('/logout', success('loggedOut'), { method: 'POST' }); window.location.replace('/dashboard/login'); }
    catch (caught) { setError(`${message(caught)} Sign out may already have completed. Reload to check.`); setBusy(false); }
  };
  useEffect(() => {
    const back = () => { setRoute(window.location.pathname + window.location.search); setMenu(false); focusContent(); };
    window.addEventListener('popstate', back);
    return () => window.removeEventListener('popstate', back);
  }, []);
  useEffect(() => {
    const escape = (event: KeyboardEvent) => { if (event.key === 'Escape' && menu) { setMenu(false); toggle.current?.focus(); } };
    window.addEventListener('keydown', escape);
    return () => window.removeEventListener('keydown', escape);
  }, [menu]);
  const id = path.startsWith('/dashboard/applications/') ? path.slice('/dashboard/applications/'.length) : '';
  return <div className="workspace min-w-0"><header className={menu ? 'sidebar menu-open' : 'sidebar'}>
    <ConsoleLink className="brand" href="/dashboard" navigate={navigate} aria-label="Cellapp applications"><img src="/dashboard/logos/cellapp-landscape.svg" alt="Cellapp" /></ConsoleLink>
    <button className="menu-toggle" ref={toggle} aria-label={menu ? 'Close menu' : 'Open menu'} aria-expanded={menu} aria-controls="navigation" onClick={() => setMenu(value => !value)}>Menu</button>
    <nav id="navigation" aria-label="Workspace" className={menu ? 'navigation is-open' : 'navigation'}>
      <div className="nav-group"><p className="nav-label">WORKSPACE</p><ConsoleLink href="/dashboard/applications" navigate={navigate} aria-current={path === '/dashboard' || path.startsWith('/dashboard/applications') ? 'page' : undefined}>Applications</ConsoleLink><ConsoleLink href="/dashboard/local-setup" navigate={navigate} aria-current={path === '/dashboard/local-setup' ? 'page' : undefined}>Local setup</ConsoleLink></div>
      <div className="nav-group"><p className="nav-label">ACCESS</p><ConsoleLink href="/dashboard/model-connections" navigate={navigate} aria-current={path === '/dashboard/model-connections' ? 'page' : undefined}>Model connections</ConsoleLink><ConsoleLink href="/dashboard/credentials" navigate={navigate} aria-current={path === '/dashboard/credentials' ? 'page' : undefined}>Deployment credentials</ConsoleLink><ConsoleLink href="/dashboard/device" navigate={navigate} aria-current={path === '/dashboard/device' ? 'page' : undefined}>Authorize tool</ConsoleLink></div>
      <div className="nav-group"><p className="nav-label">ACCOUNT</p><ConsoleLink href="/dashboard/profile" navigate={navigate} aria-current={path === '/dashboard/profile' ? 'page' : undefined}>Profile settings</ConsoleLink></div>
    </nav>
    <div className="account"><p className="eyebrow">{owner.authMode === 'dev' ? 'LOCAL DEVELOPMENT' : 'SIGNED IN'}</p><div className="account-person"><span className="account-avatar" aria-hidden="true"><span>{(username || owner.ownerId).slice(0, 1).toUpperCase()}</span>{owner.avatarUrl && !avatarFailed && <img src={owner.avatarUrl} alt="" onError={() => setAvatarFailed(true)} referrerPolicy="no-referrer" />}</span><span className="account-username" title={username || owner.ownerId}>{username || `${owner.ownerId.slice(0, 8)}…`}</span></div><button disabled={busy} onClick={() => { void logout(); }}>{busy ? 'Signing out…' : 'Sign out'}</button></div>
  </header><main id="content" tabIndex={-1} className="content">{error && <Feedback error>{error}</Feedback>}
    {path === '/dashboard/model-connections' ? <ModelConnections /> : path === '/dashboard/credentials' ? <Credentials /> : path === '/dashboard/local-setup' ? <LocalSetup /> : path === '/dashboard/device' ? <Device key={route} /> : path === '/dashboard/profile' ? <ProfilePage owner={owner} onNameChange={setUsername} /> : /^[a-f0-9]{32}$/.test(id) ? <ApplicationPage key={id} id={id} navigate={navigate} /> : path === '/dashboard' || path === '/dashboard/applications' ? <Applications navigate={navigate} /> : <><h1>Page not found</h1><ConsoleLink href="/dashboard" navigate={navigate}>Return to applications</ConsoleLink></>}
  </main></div>;
}
export function App() {
  const [state, setState] = useState<Load<Session> | { kind: 'signed-out' }>({ kind: 'loading' });
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    request('/session', session, { signal: controller.signal }).then(owner => {
      if (controller.signal.aborted) return;
      if (window.location.pathname === '/dashboard/') window.history.replaceState(null, '', '/dashboard' + window.location.search);
      if (window.location.pathname === '/dashboard/login') window.history.replaceState(null, '', loginReturn());
      setState({ kind: 'ready', value: owner });
    }, error => {
      if (controller.signal.aborted) return;
      if (error instanceof RequestError && error.status === 401) {
        if (window.location.pathname !== '/dashboard/login') window.history.replaceState(null, '', `/dashboard/login?return=${encodeURIComponent(window.location.pathname + window.location.search)}`);
        setState({ kind: 'signed-out' });
      } else setState({ kind: 'error', message: 'Unable to connect. Try again when the service is available.' });
    });
    return () => controller.abort();
  }, [version]);
  return <><a className="skip-link" href="#content" onClick={() => document.getElementById('content')?.focus()}>Skip to content</a>{state.kind === 'ready' ? <Workspace owner={state.value} /> : <main id="content" tabIndex={-1} className="login"><img className="login-logo" src="/dashboard/logos/cellapp-landscape.svg" alt="Cellapp" /><h1>{state.kind === 'signed-out' ? 'Sign in to Cellapp' : state.kind === 'error' ? 'Cellapp is unavailable' : 'Opening Cellapp…'}</h1>{state.kind === 'signed-out' && <p>Manage your applications and deployment access.</p>}{state.kind === 'signed-out' ? <a className="primary button" href={`/auth/login?return=${encodeURIComponent(loginReturn())}`}>Continue to sign in</a> : state.kind === 'error' ? <><Feedback error>{state.message}</Feedback><button onClick={() => setVersion(value => value + 1)}>Try again</button></> : <Feedback>Checking your session…</Feedback>}</main>}</>;
}
