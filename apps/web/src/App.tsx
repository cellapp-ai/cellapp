import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { application, appData, credential, decisionResult, detail, keyResult, list, message, request, RequestError, session, success } from '@/api';
import type { AppData, Session } from '@/api';

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
  return { state, reload: () => { setState({ kind: 'loading' }); setVersion(value => value + 1); } };
}
const applications = list(application);
const credentials = list(credential);
function date(value: string) { return new Date(value).toLocaleString('en-GB', { timeZone: 'UTC' }) + ' UTC'; }
function loginReturn(): string {
  const value = new URLSearchParams(window.location.search).get('return') ?? '/applications';
  const path = value.split('?')[0] ?? '';
  return ['/applications', '/credentials', '/device'].includes(path) || /^\/applications\/[a-f0-9]{32}$/.test(path) ? value : '/applications';
}
function Feedback({ children, error = false }: { children: ReactNode; error?: boolean }) {
  return <p className={error ? 'feedback error' : 'feedback'} role={error ? 'alert' : 'status'}>{children}</p>;
}
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
function Applications() {
  const { state, reload } = useResource('/apps', applications);
  return <><div className="page-heading"><p className="eyebrow">YOUR WORKSPACE</p><h1>Applications</h1><p>Manage access to your hosted applications.</p></div>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No applications yet</h2><p>Use the Cellapp deployment Skill in your AI conversation, or run <code>cellapp deploy</code> from your local project. Your applications will appear here after deployment.</p><button onClick={reload}>Refresh applications</button></section> : <div className="application-list">{state.value.map(app => <a className="application-row" href={`/applications/${app.id}`} key={app.id}><div><h2>{app.name}</h2><p>{app.url}</p></div><span className="badge">{app.suspended ? 'Suspended' : app.activeDeployment ? 'Published' : 'Not published'}</span><span aria-hidden="true">↗</span></a>)}</div>)}
  </>;
}
function DataBackend({ id, current, busy, setBusy }: { id: string; current: AppData | null; busy: boolean; setBusy: (value: boolean) => void }) {
  const [bound, setBound] = useState<AppData | null>(current);
  const [url, setUrl] = useState(current?.url ?? '');
  const [anonKey, setAnonKey] = useState(current?.anonKey ?? '');
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const [confirmClear, setConfirmClear] = useState(false);
  useEffect(() => {
    setBound(current);
    setUrl(current?.url ?? '');
    setAnonKey(current?.anonKey ?? '');
  }, [current]);
  const save = async () => {
    if (busy) return;
    setBusy(true); setError(''); setFeedback('');
    try {
      const result = await request(`/apps/${id}/data`, appData, { method: 'PUT', body: { provider: 'supabase', url, anonKey } });
      setBound(result); setUrl(result.url); setAnonKey(result.anonKey);
      setFeedback('Data backend saved. Authorized visitors share this Supabase project.');
    } catch (caught) {
      setError(`${message(caught)} The result may already have taken effect. Refresh to check, or save again when ready.`);
    } finally { setBusy(false); }
  };
  const clear = async () => {
    if (busy) return;
    setBusy(true); setError(''); setFeedback(''); setConfirmClear(false);
    try {
      await request(`/apps/${id}/data`, success('cleared'), { method: 'DELETE' });
      setBound(null); setUrl(''); setAnonKey('');
      setFeedback('Data backend removed. Visitors will no longer receive this configuration.');
    } catch (caught) {
      setError(`${message(caught)} The result may already have taken effect. Refresh to check.`);
    } finally { setBusy(false); }
  };
  return <section className="panel"><h2>Data access</h2>
    <p>Bind a Supabase project you own. Authorized visitors share one dataset. Cellapp does not run a database or proxy queries. Use the anon or publishable key, never the service role key. Anyone who can open the app can also call that project with this public key.</p>
    <p>{bound ? `Currently bound to ${bound.url}.` : 'No data backend is bound.'}</p>
    <form className="fields" onSubmit={event => { event.preventDefault(); void save(); }}>
      <label htmlFor={`data-url-${id}`}>Supabase URL<input id={`data-url-${id}`} type="url" value={url} onChange={event => setUrl(event.target.value)} autoComplete="off" spellCheck={false} required disabled={busy} /></label>
      <label htmlFor={`data-key-${id}`}>Anon or publishable key<input id={`data-key-${id}`} value={anonKey} onChange={event => setAnonKey(event.target.value)} autoComplete="off" spellCheck={false} required disabled={busy} /></label>
      <div className="actions"><button className="primary" type="submit" disabled={busy}>{busy ? 'Working…' : 'Save data backend'}</button>{bound && <button type="button" disabled={busy} onClick={() => setConfirmClear(true)}>Remove data backend</button>}</div>
    </form>
    {feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {confirmClear && <Confirm title="Remove data backend?" busy={busy} onClose={() => setConfirmClear(false)} onConfirm={() => { void clear(); }}>Visitors will stop receiving this Supabase configuration. Published files are unchanged. Anyone who already copied the public key can still call the project until you rotate it in Supabase.</Confirm>}
  </section>;
}
function ApplicationPage({ id }: { id: string }) {
  const { state, reload } = useResource(`/apps/${id}`, detail);
  const [operation, setOperation] = useState<'key' | 'delete' | null>(null);
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
        setNewKey(key); setFeedback('Share key reset. Previous keys and visitor sessions no longer work.');
        setOperation(null); reload();
      } else {
        await request(`/apps/${id}`, success('deleted'), { method: 'DELETE' });
        window.location.assign('/applications');
      }
    } catch (caught) {
      setOperation(null);
      setError(`${message(caught)} The result may already have taken effect. Refresh to check, or reset the key again when ready.`);
    } finally { setBusy(false); }
  };
  return <><a className="back" href="/applications">← Applications</a><LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && <><div className="page-heading"><p className="eyebrow">APPLICATION</p><h1>{state.value.name}</h1><span className="badge">{state.value.suspended ? 'Suspended' : state.value.release ? 'Published' : 'Not published'}</span></div>
      <section className="panel"><h2>Application address</h2><Copy value={state.value.url} label="Copy address" /><a className="text-link" href={state.value.url} target="_blank" rel="noopener noreferrer">Open application ↗</a><p>Visitors need the current share key to open this application.</p></section>
      <section className="panel"><h2>Current release</h2>{state.value.release ? <dl><dt>Release</dt><dd><code>{state.value.release.id}</code></dd><dt>Published</dt><dd>{state.value.release.publishedAt ? date(state.value.release.publishedAt) : 'Not published'}</dd><dt>Files</dt><dd>{state.value.release.fileCount}</dd><dt>Size</dt><dd>{state.value.release.bytes.toLocaleString('en-GB')} bytes</dd><dt>Navigation</dt><dd>{state.value.release.spa ? 'Single page application' : 'Static pages'}</dd></dl> : <p>No release has been published. Deploy from your local project using the Skill or CLI.</p>}</section>
      <DataBackend id={id} current={state.value.data} busy={busy} setBusy={setBusy} />
      <section className="panel"><h2>Access and lifecycle</h2><p>Reset access for visitors, or permanently delete this application.</p><div className="actions"><button disabled={busy} onClick={() => setOperation('key')}>Reset share key</button><button className="danger" disabled={busy} onClick={() => setOperation('delete')}>Delete application</button><button disabled={busy} onClick={() => { setNewKey(''); reload(); }}>Refresh details</button></div></section>
    </>}
    {feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {newKey && <section className="panel"><h2>New share key</h2><p>Copy this key now. It will be cleared when you leave this page and cannot be retrieved later.</p><Copy value={newKey} label="Copy share key" /><button onClick={() => setNewKey('')}>Dismiss key</button></section>}
    {operation && <Confirm title={operation === 'key' ? 'Reset share key?' : 'Delete application?'} busy={busy} onClose={() => setOperation(null)} onConfirm={() => { void perform(); }}>{operation === 'key' ? 'The previous key and all existing visitor sessions will stop working. Share the new key with people who still need access.' : 'This permanently removes the application. Its address will stop working and it cannot receive more deployments.'}</Confirm>}
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
  return <><div className="page-heading"><p className="eyebrow">ACCOUNT ACCESS</p><h1>Deployment credentials</h1><p>Review active access granted to local deployment tools.</p></div><LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No active credentials</h2><p>Authorize your local deployment tool when you next deploy.</p></section> : <div className="credential-list">{state.value.map(item => <section className="panel" key={item.id}><h2 className="credential-id">{item.id}</h2><dl><dt>Created</dt><dd>{date(item.createdAt)}</dd><dt>Expires</dt><dd>{date(item.expiresAt)}</dd></dl><button className="danger" disabled={busy} onClick={() => setSelected(item.id)}>Revoke</button></section>)}</div>)}
    <button disabled={busy} onClick={reload}>Refresh credentials</button>{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {selected && <Confirm title="Revoke deployment access?" onClose={() => setSelected(null)} busy={busy} onConfirm={() => { void revoke(); }}>The tool using this credential will need to authorize again. Published applications remain available.</Confirm>}
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
  return <><div className="page-heading"><p className="eyebrow">DEPLOYMENT ACCESS</p><h1>Authorize a deployment tool</h1><p>Only approve a request you just started on your own device.</p></div><section className="panel"><h2>Check your authorization code</h2><p>Approval gives the tool access to manage your applications. Compare this code with the one shown in your local tool.</p><label htmlFor="device-code">Authorization code</label><input id="device-code" value={code} maxLength={64} disabled={busy || !!result} onChange={event => setCode(event.target.value)} autoComplete="off" spellCheck={false} aria-describedby={error ? 'device-error' : undefined} /><div className="actions"><button className="primary" disabled={busy || !!result || !code.trim()} onClick={() => { void decide('approved'); }}>{busy ? 'Working…' : 'Approve access'}</button><button disabled={busy || !!result || !code.trim()} onClick={() => { void decide('denied'); }}>Deny access</button></div>{result && <Feedback>{result}</Feedback>}{error && <p id="device-error" className="feedback error" role="alert">{error}</p>}</section></>;
}
function Workspace({ owner }: { owner: Session }) {
  const [menu, setMenu] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const toggle = useRef<HTMLButtonElement>(null);
  const path = window.location.pathname;
  const logout = async () => {
    setBusy(true); setError('');
    try { await request('/logout', success('loggedOut'), { method: 'POST' }); window.location.replace('/login'); }
    catch (caught) { setError(`${message(caught)} Sign out may already have completed. Reload to check.`); setBusy(false); }
  };
  useEffect(() => {
    const escape = (event: KeyboardEvent) => { if (event.key === 'Escape' && menu) { setMenu(false); toggle.current?.focus(); } };
    window.addEventListener('keydown', escape);
    return () => window.removeEventListener('keydown', escape);
  }, [menu]);
  const id = path.startsWith('/applications/') ? path.slice('/applications/'.length) : '';
  return <div className="workspace min-w-0"><header className="sidebar"><a className="brand" href="/applications" aria-label="Cellapp applications"><img src="/logos/cellapp-landscape.svg" alt="Cellapp" /></a><button className="menu-toggle" ref={toggle} aria-label={menu ? 'Close menu' : 'Open menu'} aria-expanded={menu} aria-controls="navigation" onClick={() => setMenu(value => !value)}>Menu</button><nav id="navigation" aria-label="Workspace" className={menu ? 'navigation is-open' : 'navigation'}><a href="/applications" aria-current={path.startsWith('/applications') ? 'page' : undefined}>Applications</a><a href="/credentials" aria-current={path === '/credentials' ? 'page' : undefined}>Deployment credentials</a><a href="/device" aria-current={path === '/device' ? 'page' : undefined}>Authorize tool</a></nav><div className="account"><p className="eyebrow">{owner.authMode === 'dev' ? 'LOCAL DEVELOPMENT' : 'SIGNED IN'}</p><code title={owner.ownerId}>{owner.ownerId.slice(0, 8)}…</code><button disabled={busy} onClick={() => { void logout(); }}>{busy ? 'Signing out…' : 'Sign out'}</button></div></header><main id="content" tabIndex={-1} className="content">{error && <Feedback error>{error}</Feedback>}{path === '/credentials' ? <Credentials /> : path === '/device' ? <Device /> : /^[a-f0-9]{32}$/.test(id) ? <ApplicationPage id={id} /> : path === '/applications' ? <Applications /> : <><h1>Page not found</h1><a href="/applications">Return to applications</a></>}</main></div>;
}
export function App() {
  const [state, setState] = useState<Load<Session> | { kind: 'signed-out' }>({ kind: 'loading' });
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    request('/session', session, { signal: controller.signal }).then(owner => {
      if (controller.signal.aborted) return;
      if (window.location.pathname === '/' || window.location.pathname === '/login') window.location.replace(window.location.pathname === '/login' ? loginReturn() : '/applications');
      else setState({ kind: 'ready', value: owner });
    }, error => {
      if (controller.signal.aborted) return;
      if (error instanceof RequestError && error.status === 401) {
        if (window.location.pathname !== '/login') window.location.replace(`/login?return=${encodeURIComponent(window.location.pathname === '/' ? '/applications' : window.location.pathname + window.location.search)}`);
        else setState({ kind: 'signed-out' });
      } else setState({ kind: 'error', message: 'Unable to connect. Try again when the service is available.' });
    });
    return () => controller.abort();
  }, [version]);
  return <><a className="skip-link" href="#content" onClick={() => document.getElementById('content')?.focus()}>Skip to content</a>{state.kind === 'ready' ? <Workspace owner={state.value} /> : <main id="content" tabIndex={-1} className="login"><img className="login-logo" src="/logos/cellapp-landscape.svg" alt="Cellapp" /><h1>Sign in to Cellapp</h1><p>Manage your applications and deployment access.</p>{state.kind === 'signed-out' ? <a className="primary button" href={`/auth/login?return=${encodeURIComponent(loginReturn())}`}>Continue to sign in</a> : state.kind === 'error' ? <><Feedback error>{state.message}</Feedback><button onClick={() => setVersion(value => value + 1)}>Try again</button></> : <Feedback>Checking your session…</Feedback>}</main>}</>;
}
