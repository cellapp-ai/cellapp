import { useEffect, useRef, useState } from 'react';
import type { ComponentProps, MouseEvent, ReactNode } from 'react';
import { access, application, credential, dataStatus, decisionResult, detail, keyResult, list, message, profile, request, RequestError, session, success } from '@/api';
import type { Session } from '@/api';

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
function date(value: string) { return new Date(value).toLocaleString('en-GB', { timeZone: 'UTC' }) + ' UTC'; }
function loginReturn(): string {
  const value = new URLSearchParams(window.location.search).get('return') ?? '/applications';
  const path = value.split('?')[0] ?? '';
  return ['/applications', '/credentials', '/device', '/profile'].includes(path) || /^\/applications\/[a-f0-9]{32}$/.test(path) ? value : '/applications';
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
  return <><div className="page-heading"><p className="eyebrow">YOUR WORKSPACE</p><h1>Applications</h1><p>Manage access to your hosted applications.</p></div>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No applications yet</h2><p>Use the Cellapp deployment Skill in your AI conversation, or run <code>cellapp deploy</code> from your local project. Your applications will appear here after deployment.</p><button onClick={() => reload()}>Refresh applications</button></section> : <div className="application-list">{state.value.map(app => <ConsoleLink className="application-row" href={`/applications/${app.id}`} navigate={navigate} key={app.id}><div><h2>{app.name}</h2><p>@{app.namespace}/{app.slug}</p><p>{app.url}</p></div><span className="badge">{app.suspended ? 'Suspended' : app.activeDeployment ? 'Published' : 'Not published'}</span><span aria-hidden="true">↗</span></ConsoleLink>)}</div>)}
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
function ApplicationPage({ id, navigate }: { id: string; navigate: Navigate }) {
  const { state, reload } = useResource(`/apps/${id}`, detail);
  const accessState = useResource(`/apps/${id}/access`, access);
  const dataState = useResource(`/apps/${id}/data`, dataStatus);
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
        navigate('/applications');
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
  return <><ConsoleLink className="back" href="/applications" navigate={navigate}>← Applications</ConsoleLink><LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && <><div className="page-heading"><p className="eyebrow">APPLICATION</p><h1>{state.value.name}</h1><span className="badge">{state.value.suspended ? 'Suspended' : state.value.release ? 'Published' : 'Not published'}</span></div>
      <section className="panel"><h2>Application address</h2><p><code>@{state.value.namespace}/{state.value.slug}</code></p><Copy value={state.value.aliasUrl} label="Copy share link" /><Copy value={state.value.url} label="Copy address" /><a className="text-link" href={state.value.url} target="_blank" rel="noopener noreferrer">Open application ↗</a><p>Address format: <code>app-slug-user-slug</code>. Existing addresses stay assigned to this application.</p><AppSlugSettings id={id} slug={state.value.slug} userSlug={state.value.namespace} reload={reload} /><p>Visitors sign in with Cellapp. {state.value.visibility === 'public' ? 'Any signed-in user can access this application.' : 'Restricted access requires approval or an access key.'}</p></section>
      <section className="panel"><h2>Current release</h2>{state.value.release ? <dl><dt>Release</dt><dd><code>{state.value.release.id}</code></dd><dt>Published</dt><dd>{state.value.release.publishedAt ? date(state.value.release.publishedAt) : 'Not published'}</dd><dt>Files</dt><dd>{state.value.release.fileCount}</dd><dt>Size</dt><dd>{state.value.release.bytes.toLocaleString('en-GB')} bytes</dd><dt>Navigation</dt><dd>{state.value.release.spa ? 'Single page application' : 'Static pages'}</dd></dl> : <p>No release has been published. Deploy from your local project using the Skill or CLI.</p>}</section>
      <section className="panel"><h2>Access and lifecycle</h2><dl><dt>Visibility</dt><dd>{state.value.visibility === 'public' ? 'Public to signed-in users' : 'Restricted'}</dd></dl><div className="actions"><button disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/visibility/${state.value.visibility === 'public' ? 'restricted' : 'public'}`, title: state.value.visibility === 'public' ? 'Restrict access?' : 'Make public?', explanation: state.value.visibility === 'public' ? 'Users without an explicit grant will lose access immediately.' : 'Every signed-in Cellapp user will be able to open this application.' })}>{state.value.visibility === 'public' ? 'Restrict access' : 'Make public'}</button><button disabled={busy} onClick={() => setOperation('key')}>Rotate access key</button><button className="danger" disabled={busy} onClick={() => setOperation('delete')}>Delete application</button><button disabled={busy} onClick={() => { setNewKey(''); reload(); accessState.reload(); dataState.reload(); }}>Refresh details</button></div></section>
    </>}
    <section className="panel"><h2>Application tables</h2><LoadingError state={dataState.state} retry={dataState.reload} />{dataState.state.kind === 'ready' && <dl><dt>Provisioning</dt><dd>{dataState.state.value.state}</dd><dt>Table version</dt><dd>{dataState.state.value.version}</dd></dl>}</section>
    <section className="panel"><h2>Access requests</h2><LoadingError state={accessState.state} retry={accessState.reload} />{accessState.state.kind === 'ready' && (accessState.state.value.requests.filter(item => item.status === 'pending').length === 0 ? <p>No pending requests.</p> : accessState.state.value.requests.filter(item => item.status === 'pending').map(item => <div className="access-row" key={item.userId}><div><code>{item.userId}</code><p>Requested {date(item.requestedAt)}</p></div><div className="actions"><button disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/requests/${item.userId}/approve`, title: 'Approve access?', explanation: 'This user will be able to open the application until you revoke their grant.' })}>Approve</button><button disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/requests/${item.userId}/deny`, title: 'Deny request?', explanation: 'This request will be marked denied.' })}>Deny</button></div></div>))}</section>
    <section className="panel"><h2>Granted users</h2>{accessState.state.kind === 'ready' && (accessState.state.value.grants.length === 0 ? <p>No user grants.</p> : accessState.state.value.grants.map(item => <div className="access-row" key={item.userId}><div><code>{item.userId}</code><p>{item.source} · {date(item.grantedAt)}</p></div><button className="danger" disabled={busy} onClick={() => setAccessAction({ path: `/apps/${id}/grants/${item.userId}/revoke`, title: 'Revoke access?', explanation: 'This user will lose access immediately unless the application is public.' })}>Revoke</button></div>))}</section>
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
  return <><div className="page-heading"><p className="eyebrow">ACCOUNT ACCESS</p><h1>Deployment credentials</h1><p>Review active access granted to local deployment tools.</p></div><LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && (state.value.length === 0 ? <section className="empty"><h2>No active credentials</h2><p>Authorize your local deployment tool when you next deploy.</p></section> : <div className="credential-list">{state.value.map(item => <section className="panel" key={item.id}><h2 className="credential-id">{item.id}</h2><dl><dt>Created</dt><dd>{date(item.createdAt)}</dd><dt>Expires</dt><dd>{date(item.expiresAt)}</dd></dl><button className="danger" disabled={busy} onClick={() => setSelected(item.id)}>Revoke</button></section>)}</div>)}
    <button disabled={busy} onClick={() => reload()}>Refresh credentials</button>{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}
    {selected && <Confirm title="Revoke deployment access?" onClose={() => setSelected(null)} busy={busy} onConfirm={() => { void revoke(); }}>The tool using this credential will need to authorize again. Published applications remain available.</Confirm>}
  </>;
}
function ProfilePage({ owner }: { owner: Session }) {
  const { state, reload } = useResource('/profile', profile);
  const [draft, setDraft] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const saved = state.kind === 'ready' ? state.value.displayName : '';
  const name = draft ?? saved;
  const valid = name === name.trim() && Array.from(name).length <= 80 && !Array.from(name).some(character => { const code = character.codePointAt(0) ?? 0; return code < 32 || code === 127; });
  const save = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!valid || busy || state.kind !== 'ready') return;
    setBusy(true); setFeedback(''); setError('');
    try {
      await request('/profile', profile, { method: 'PUT', body: { displayName: name } });
      setDraft(null); reload(); setFeedback('Display name saved.');
    } catch (caught) {
      setError(`${message(caught)} Refresh the profile to check whether the change was saved.`);
    } finally { setBusy(false); }
  };
  return <><div className="page-heading"><p className="eyebrow">YOUR ACCOUNT</p><h1>Profile settings</h1><p>Manage the name shown in your Cellapp account.</p></div>
    <LoadingError state={state} retry={reload} />
    {state.kind === 'ready' && <><section className="panel"><h2>Display name</h2><form className="profile-form" onSubmit={event => { void save(event); }}><label htmlFor="display-name">Name</label><input id="display-name" value={name} disabled={busy} onChange={event => { setDraft(event.target.value); setFeedback(''); setError(''); }} aria-describedby="display-name-help" /><p id="display-name-help">Optional, up to 80 characters. Leave blank to clear it.</p><div className="actions"><button className="primary" type="submit" disabled={busy || !valid || name === saved}>{busy ? 'Saving…' : 'Save name'}</button><button type="button" disabled={busy || name === saved} onClick={() => { setDraft(null); setError(''); }}>Discard changes</button><button type="button" disabled={busy} onClick={() => { setDraft(null); setFeedback(''); setError(''); reload(); }}>Refresh profile</button></div>{!valid && <Feedback error>Remove leading or trailing spaces, control characters, and use no more than 80 characters.</Feedback>}{feedback && <Feedback>{feedback}</Feedback>}{error && <Feedback error>{error}</Feedback>}</form></section>
      <section className="panel"><h2>User slug</h2><p>Your application addresses end with this slug. Changing it updates their current addresses; previous addresses remain assigned to the same applications.</p><SlugEditor label="User slug" value={state.value.userSlug} maxLength={61} save={async next => { await request('/profile/slug', profile, { method: 'PUT', body: { userSlug: next } }); reload(true); }} /></section>
      <section className="panel"><h2>Account details</h2><dl><dt>Account ID</dt><dd><code>{owner.ownerId}</code></dd><dt>Sign-in mode</dt><dd>{owner.authMode === 'dev' ? 'Local development' : 'GitHub'}</dd></dl><p>Your account ID is fixed. Changing your display name does not change application addresses.</p></section></>}
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
    try { await request('/logout', success('loggedOut'), { method: 'POST' }); window.location.replace('/login'); }
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
  const id = path.startsWith('/applications/') ? path.slice('/applications/'.length) : '';
  return <div className="workspace min-w-0"><header className="sidebar">
    <ConsoleLink className="brand" href="/applications" navigate={navigate} aria-label="Cellapp applications"><img src="/logos/cellapp-landscape.svg" alt="Cellapp" /></ConsoleLink>
    <button className="menu-toggle" ref={toggle} aria-label={menu ? 'Close menu' : 'Open menu'} aria-expanded={menu} aria-controls="navigation" onClick={() => setMenu(value => !value)}>Menu</button>
    <nav id="navigation" aria-label="Workspace" className={menu ? 'navigation is-open' : 'navigation'}>
      <ConsoleLink href="/applications" navigate={navigate} aria-current={path.startsWith('/applications') ? 'page' : undefined}>Applications</ConsoleLink>
      <ConsoleLink href="/credentials" navigate={navigate} aria-current={path === '/credentials' ? 'page' : undefined}>Deployment credentials</ConsoleLink>
      <ConsoleLink href="/device" navigate={navigate} aria-current={path === '/device' ? 'page' : undefined}>Authorize tool</ConsoleLink>
      <ConsoleLink href="/profile" navigate={navigate} aria-current={path === '/profile' ? 'page' : undefined}>Profile settings</ConsoleLink>
    </nav>
    <div className="account"><p className="eyebrow">{owner.authMode === 'dev' ? 'LOCAL DEVELOPMENT' : 'SIGNED IN'}</p><code title={owner.ownerId}>{owner.ownerId.slice(0, 8)}…</code><button disabled={busy} onClick={() => { void logout(); }}>{busy ? 'Signing out…' : 'Sign out'}</button></div>
  </header><main id="content" tabIndex={-1} className="content">{error && <Feedback error>{error}</Feedback>}
    {path === '/credentials' ? <Credentials /> : path === '/device' ? <Device key={route} /> : path === '/profile' ? <ProfilePage owner={owner} /> : /^[a-f0-9]{32}$/.test(id) ? <ApplicationPage key={id} id={id} navigate={navigate} /> : path === '/applications' ? <Applications navigate={navigate} /> : <><h1>Page not found</h1><ConsoleLink href="/applications" navigate={navigate}>Return to applications</ConsoleLink></>}
  </main></div>;
}
export function App() {
  const [state, setState] = useState<Load<Session> | { kind: 'signed-out' }>({ kind: 'loading' });
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    request('/session', session, { signal: controller.signal }).then(owner => {
      if (controller.signal.aborted) return;
      if (window.location.pathname === '/' || window.location.pathname === '/login') window.history.replaceState(null, '', window.location.pathname === '/login' ? loginReturn() : '/applications');
      setState({ kind: 'ready', value: owner });
    }, error => {
      if (controller.signal.aborted) return;
      if (error instanceof RequestError && error.status === 401) {
        if (window.location.pathname !== '/login') window.history.replaceState(null, '', `/login?return=${encodeURIComponent(window.location.pathname === '/' ? '/applications' : window.location.pathname + window.location.search)}`);
        setState({ kind: 'signed-out' });
      } else setState({ kind: 'error', message: 'Unable to connect. Try again when the service is available.' });
    });
    return () => controller.abort();
  }, [version]);
  return <><a className="skip-link" href="#content" onClick={() => document.getElementById('content')?.focus()}>Skip to content</a>{state.kind === 'ready' ? <Workspace owner={state.value} /> : <main id="content" tabIndex={-1} className="login"><img className="login-logo" src="/logos/cellapp-landscape.svg" alt="Cellapp" /><h1>{state.kind === 'signed-out' ? 'Sign in to Cellapp' : state.kind === 'error' ? 'Cellapp is unavailable' : 'Opening Cellapp…'}</h1>{state.kind === 'signed-out' && <p>Manage your applications and deployment access.</p>}{state.kind === 'signed-out' ? <a className="primary button" href={`/auth/login?return=${encodeURIComponent(loginReturn())}`}>Continue to sign in</a> : state.kind === 'error' ? <><Feedback error>{state.message}</Feedback><button onClick={() => setVersion(value => value + 1)}>Try again</button></> : <Feedback>Checking your session…</Feedback>}</main>}</>;
}
