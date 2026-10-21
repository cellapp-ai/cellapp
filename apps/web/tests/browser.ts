import assert from 'node:assert/strict';
import { access, mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright-core';
import AxeBuilder from '@axe-core/playwright';
import { preview } from 'vite';

const root = fileURLToPath(new URL('../', import.meta.url));
const artifacts = fileURLToPath(new URL('../../../test-results/web/', import.meta.url));
const executablePath = process.env.CHROME_PATH ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
await access(executablePath);
await mkdir(artifacts, { recursive: true });
const server = await preview({ root, preview: { host: '127.0.0.1', port: 0, strictPort: true } });
const address = server.httpServer.address();
assert(address && typeof address !== 'string');
const origin = `http://127.0.0.1:${address.port}`;
const failures: string[] = [];
const checks: string[] = [];
const appID = '1'.repeat(32);
const ownerID = '2'.repeat(32);
const credentialID = '3'.repeat(32);
const key = '4'.repeat(64);
let mode: 'normal' | 'empty' | 'error' | 'expired' | 'lost' = 'normal';
let mutations = 0;
let sessionRequests = 0;
let deviceDecision = '';
let displayName = '';
const githubUsername = 'Octocat';
const avatarUrl = 'https://avatars.githubusercontent.com/u/123456?s=96';
let userSlug = 'demo';
let appSlug = 'shared-notes';
let structureMode: 'valid' | 'pending' | 'empty' | 'malformed' | 'unavailable' | 'long' = 'valid';
let appView: 'unpublished' | 'published' | 'suspended' = 'unpublished';
let appName = 'Shared notes';
let appVisibility: 'public' | 'restricted' = 'restricted';
let anonymous = {pages:false,read:false,write:false,operatorWritesEnabled:false,effectiveWrite:false};
let loggedOut = false;
let browser;
try {
  browser = await chromium.launch({ executablePath, headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();
  page.on('pageerror', error => failures.push(error.message));
  page.on('request', request => { if (!request.url().startsWith(`${origin}/`) && request.url() !== avatarUrl) failures.push(`External request: ${request.url()}`); if (request.url().includes('/rest/v1/')) failures.push('Dashboard fetched table rows'); });
  await page.route('https://avatars.githubusercontent.com/**', route => route.fulfill({status:200, contentType:'image/svg+xml', body:'<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96"><circle cx="48" cy="48" r="48" fill="#63788d"/></svg>'}));
  // UI fixtures only: actual authentication, transactions and HTTPS are checked separately.
  await page.route('**/api/console/**', async route => {
    const path = new URL(route.request().url()).pathname.slice('/api/console'.length);
    const method = route.request().method();
    if (mode === 'expired') { await route.fulfill({status: 401, json: {error: 'login_required'}}); return; }
    if (method !== 'GET') {
      mutations++;
      if (mode === 'lost') { await route.abort('failed'); return; }
      if (mode === 'error') { await route.fulfill({status:503,json:{message:'Service temporarily unavailable.'}}); return; }
      if (path === '/logout') { loggedOut = true; await route.fulfill({json: {loggedOut:true}}); return; }
      if (path.endsWith('/key')) { await route.fulfill({json: {key}}); return; }
      if (path.endsWith('/anonymous') && method === 'PUT') {
        const body = route.request().postDataJSON() as Record<string, unknown>;
        if (Object.keys(body).sort().join(',') !== 'pages,read,write') { await route.fulfill({status:400,json:{error:'invalid_json'}}); return; }
        anonymous = {pages:body.pages as boolean,read:body.read as boolean,write:body.write as boolean,operatorWritesEnabled:false,effectiveWrite:false};
        await route.fulfill({json: anonymous}); return;
      }
      if (path.endsWith('/visibility/public')) { appVisibility = 'public'; await route.fulfill({json: {visibility:appVisibility}}); return; }
      if (path.endsWith('/visibility/restricted')) { appVisibility = 'restricted'; anonymous = {pages:false,read:false,write:false,operatorWritesEnabled:false,effectiveWrite:false}; await route.fulfill({json: {visibility:appVisibility}}); return; }
      if (path === '/device/decision') { deviceDecision = route.request().postDataJSON().decision; await route.fulfill({json: {status: deviceDecision}}); return; }
      if (path === '/profile' && method === 'PUT') { displayName = route.request().postDataJSON().displayName; await route.fulfill({json: {displayName,userSlug,githubUsername}}); return; }
      if (path === '/profile/slug' && method === 'PUT') { userSlug = route.request().postDataJSON().userSlug; await route.fulfill({json: {displayName,userSlug,githubUsername}}); return; }
      if (path.endsWith('/slug') && method === 'PUT') { appSlug = route.request().postDataJSON().slug; await route.fulfill({json: {id:appID,name:'Shared notes',namespace:userSlug,slug:appSlug,aliasUrl:`https://control.localhost/a/${userSlug}/${appSlug}`,url:`https://${appSlug}-${userSlug}.apps.localhost`,activeDeployment:null,suspended:false,visibility:'restricted',dataState:'ready'}}); return; }
      await route.fulfill({json: {deleted: true, revoked: true, loggedOut: true}}); return;
    }
    if (path === '/session') { sessionRequests++; await route.fulfill(loggedOut ? {status:401,json:{error:'login_required'}} : {json: {ownerId: ownerID, authMode: 'github', username:displayName || githubUsername, avatarUrl}}); return; }
    if (mode === 'error') { await route.fulfill({status: 503, json: {message: 'Service temporarily unavailable.'}}); return; }
    const app = {id: appID, name: appName, namespace: userSlug, slug: appSlug, aliasUrl: `https://control.localhost/a/${userSlug}/${appSlug}`, url: `https://${appSlug}-${userSlug}.apps.localhost`, activeDeployment: appView === 'unpublished' ? null : credentialID, suspended: appView === 'suspended', visibility: appVisibility, dataState: 'ready'};
    if (path.endsWith('/data/structure')) {
      if (structureMode === 'unavailable') { await route.fulfill({status:503,json:{message:'Application table structure unavailable.'}}); return; }
      if (structureMode === 'malformed') { await route.fulfill({json:{state:'ready',version:1,tables:[{name:'notes',rows:[{secret:'never show'}]}]}}); return; }
      const table = {name:structureMode === 'long' ? 'notes_'+'x'.repeat(50) : 'notes',columns:[{name:'id',type:'uuid'},{name:'owner_id',type:'text'},{name:'body',type:'text',nullable:true}],primaryKey:['id'],indexes:[{columns:['owner_id']}],policies:{select:'own-rows',insert:'own-rows',ownerColumn:'owner_id'}};
      await route.fulfill({json:{state:structureMode === 'pending' ? 'pending' : 'ready',version:structureMode === 'empty' ? 0 : 1,tables:structureMode === 'empty' ? [] : [table]}}); return;
    }
    const value = path === '/apps' ? mode === 'empty' ? [] : [app] : path === '/credentials' ? mode === 'empty' ? [] : [{id: credentialID, createdAt:'2026-09-01T00:00:00Z', expiresAt:'2026-10-01T00:00:00Z'}] : path === '/profile' ? {displayName,userSlug,githubUsername} : path.endsWith('/access') ? {requests: [], grants: []} : path.endsWith('/anonymous') ? anonymous : path.endsWith('/data') ? {state:'ready',version:1,digest:'f'.repeat(64)} : {...app, release: null};
    await route.fulfill({json: value});
  });
  await page.goto(origin + '/dashboard/');
  await page.getByRole('heading', {name: 'Applications', exact: true}).waitFor();
  assert.equal(new URL(page.url()).pathname, '/dashboard');
  assert.equal(await page.locator('link[rel="icon"]').getAttribute('href'), '/dashboard/logos/cellapp-square.svg');
  assert((await page.locator('script[src]').first().getAttribute('src'))?.startsWith('/dashboard/assets/'));
  await page.getByRole('heading', {name: 'Shared notes'}).waitFor();
  await page.locator('.account-username').getByText(githubUsername).waitFor();
  assert.equal(await page.locator('.account-avatar img').getAttribute('src'), avatarUrl);
  await page.getByText('@demo/shared-notes').waitFor();
  assert.equal(await page.title(), 'Cellapp');
  await page.evaluate(() => document.fonts.ready);
  assert(await page.evaluate(() => document.fonts.check('16px "Space Grotesk"')));
  for (const width of [320,390,768,800,801,1440]) {
    await page.setViewportSize({width,height:1000});
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `Overflow at ${width}`);
    assert.deepEqual((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa']).analyze()).violations, []);
    if ([390,1440].includes(width)) await page.screenshot({path: `${artifacts}/${width}.png`,fullPage:true});
  }
  checks.push('Management layout, local fonts, responsive sizes and axe with explicit UI fixtures');
  assert.equal(sessionRequests,1);
  await page.getByRole('link',{name:'Deployment credentials',exact:true}).click();
  await page.getByRole('heading',{name:'Deployment credentials'}).waitFor();
  assert.equal(sessionRequests,1);
  assert.equal(await page.locator('main').evaluate(e => e === document.activeElement),true);
  await page.goBack();await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();
  assert.equal(sessionRequests,1);
  await page.locator('a.manage-link').click();await page.getByRole('heading',{name:'Shared notes',exact:true}).waitFor();
  assert.equal(sessionRequests,1);
  await page.getByRole('link',{name:'Applications',exact:false}).last().click();
  await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();
  assert.equal(sessionRequests,1);
  await page.getByRole('link',{name:'Profile settings'}).click();
  await page.getByRole('heading',{name:'Profile settings'}).waitFor();
  assert.equal(new URL(page.url()).pathname,'/dashboard/profile');
  assert.equal(sessionRequests,1);
  await page.goBack();await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();
  assert.equal(sessionRequests,1);
  checks.push('Internal links and browser history keep the authenticated shell without another session check');
  await page.setViewportSize({width:390,height:844});
  await page.getByRole('button',{name:'Open menu'}).focus();await page.keyboard.press('Enter');
  assert.equal(await page.getByRole('button',{name:'Close menu'}).getAttribute('aria-expanded'),'true');
  assert(await page.getByRole('button',{name:'Sign out'}).isVisible());
  await page.getByRole('link',{name:'Deployment credentials',exact:true}).focus();await page.keyboard.press('Escape');
  assert(await page.getByRole('button',{name:'Open menu'}).evaluate(e => e === document.activeElement));
  assert(!(await page.getByRole('link',{name:'Deployment credentials',exact:true}).isVisible()));
  assert(!(await page.getByRole('button',{name:'Sign out'}).isVisible()));
  await page.reload();await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();await page.keyboard.press('Tab');
  assert(await page.getByRole('link',{name:'Skip to content'}).evaluate(e => e === document.activeElement));
  await page.keyboard.press('Enter');assert(await page.locator('main').evaluate(e => e === document.activeElement));
  checks.push('Keyboard navigation, mobile menu Escape and skip focus');
  await page.getByRole('heading',{name:'Shared notes'}).click();
  await page.getByRole('heading',{name:'Shared notes',exact:true}).waitFor();
  await page.getByText('@demo/shared-notes').waitFor();
  await page.getByRole('button',{name:'Copy share link'}).waitFor();
  await page.getByLabel('Application slug').fill('notes');
  await page.getByRole('button',{name:'Save application slug'}).click();
  await page.getByRole('status').filter({hasText:'Application slug saved'}).waitFor();
  await page.locator('.app-address').getByText('https://notes-demo.apps.localhost').waitFor();
  await page.getByText('No release has been published.',{exact:false}).waitFor();
  await page.getByText('Applied table version 1.',{exact:false}).waitFor();
  await page.getByRole('button',{name:'notes 3 columns'}).waitFor();
  await page.getByRole('table',{name:'Columns in notes'}).waitFor();
  assert(await page.getByRole('checkbox',{name:'Open published pages'}).isDisabled());
  await page.getByRole('button',{name:'Make public'}).click();await page.getByRole('dialog').getByRole('button',{name:'Confirm'}).click();
  await page.getByRole('checkbox',{name:'Open published pages'}).waitFor();
  await page.getByRole('checkbox',{name:'Open published pages'}).check();
  await page.getByRole('checkbox',{name:'Submit to permitted tables'}).check();
  await page.getByRole('button',{name:'Save anonymous access'}).click();
  await page.getByRole('dialog').getByRole('button',{name:'Confirm'}).click();
  await page.getByRole('status').filter({hasText:'Anonymous access updated'}).waitFor();
  await page.getByRole('checkbox',{name:'Read permitted tables'}).check();
  await page.getByRole('button',{name:'Save anonymous access'}).click();
  await page.getByRole('dialog').getByRole('button',{name:'Confirm'}).click();
  await page.getByRole('status').filter({hasText:'Anonymous access updated'}).waitFor();
  assert.equal(anonymous.read,true);
  await page.getByText('Anonymous submissions are configured but disabled by operations.').waitFor();
  checks.push('Anonymous controls, confirmation and operator-off feedback');
  assert.equal(await page.getByText('never show').count(),0);
  await page.setViewportSize({width:1440,height:1000});await page.evaluate(() => {window.scrollTo(0,0);(document.activeElement as HTMLElement)?.blur();});await page.screenshot({path:`${artifacts}/detail-1440.png`,fullPage:true});
  await page.setViewportSize({width:390,height:844});await page.evaluate(() => window.scrollTo(0,0));await page.screenshot({path:`${artifacts}/detail-390.png`,fullPage:true});
  structureMode='long';await page.getByRole('button',{name:'Refresh structure'}).click();await page.getByRole('button',{name:/notes_x+ 3 columns/}).focus();await page.keyboard.press('Enter');
  assert(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth));
  structureMode='valid';await page.getByRole('button',{name:'Refresh structure'}).click();await page.getByRole('table',{name:'Columns in notes'}).waitFor();
  structureMode='pending';await page.getByRole('button',{name:'Refresh structure'}).click();await page.getByRole('status').filter({hasText:'Data service is pending'}).waitFor();
  structureMode='empty';await page.getByRole('button',{name:'Refresh structure'}).click();await page.getByRole('heading',{name:'No tables defined'}).waitFor();
  structureMode='malformed';await page.getByRole('button',{name:'Refresh structure'}).click();await page.getByRole('alert').filter({hasText:'Unexpected table structure response'}).waitFor();
  structureMode='unavailable';await page.getByRole('button',{name:'Try again'}).click();await page.getByRole('alert').filter({hasText:'Application table structure unavailable'}).waitFor();
  structureMode='valid';await page.getByRole('button',{name:'Try again'}).click();await page.getByRole('table',{name:'Columns in notes'}).waitFor();
  await page.reload();await page.getByRole('heading',{name:'Shared notes',exact:true}).waitFor();
  await page.getByRole('button',{name:'Rotate access key',exact:true}).click();
  await page.getByRole('dialog').waitFor();
  const before = mutations;
  await page.getByRole('button',{name:'Cancel',exact:true}).click();assert.equal(mutations,before);
  assert(await page.getByRole('button',{name:'Rotate access key',exact:true}).evaluate(e => e === document.activeElement));
  await page.getByRole('button',{name:'Rotate access key',exact:true}).click();await page.getByRole('button',{name:'Confirm',exact:true}).click();
  await page.getByRole('heading',{name:'New access key'}).waitFor();
  assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length),0);
  await page.evaluate('Object.defineProperty(navigator, "clipboard", {configurable:true,value:{writeText: () => Promise.reject(new Error("blocked"))}})');
  await page.getByRole('button',{name:'Copy access key'}).click();
  await page.getByRole('status').filter({hasText:'Copy unavailable'}).waitFor();
  assert(!page.url().includes(key));
  assert.deepEqual((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa']).analyze()).violations,[]);
  await page.locator('a.back').click();await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();
  assert.equal(await page.getByText(key,{exact:true}).count(),0);
  await page.locator('a.manage-link').click();await page.getByRole('heading',{name:'Shared notes',exact:true}).waitFor();
  assert.equal(await page.getByRole('heading',{name:'New access key'}).count(),0);
  mode='lost';await page.getByRole('button',{name:'Rotate access key',exact:true}).click();const sent=mutations;await page.getByRole('button',{name:'Confirm',exact:true}).click();
  await page.getByRole('alert').filter({hasText:'may already have taken effect'}).waitFor();assert.equal(mutations,sent+1);
  checks.push('Details deep link, unpublished state, confirmation/cancel focus, transient key and uncertain write without retry');
  mode='normal';await page.goto(origin+'/dashboard/credentials');await page.getByRole('button',{name:'Revoke',exact:true}).click();await page.getByRole('button',{name:'Confirm',exact:true}).click();await page.getByRole('status').filter({hasText:'credential revoked'}).waitFor();
  mode='lost';const beforeRevoke=mutations;await page.getByRole('button',{name:'Revoke',exact:true}).click();await page.getByRole('button',{name:'Confirm',exact:true}).click();await page.getByRole('alert').filter({hasText:'may already have taken effect'}).waitFor();assert.equal(mutations,beforeRevoke+1);
  mode='empty';await page.getByRole('button',{name:'Refresh credentials'}).click();await page.getByRole('heading',{name:'No active credentials'}).waitFor();
  mode='error';await page.getByRole('button',{name:'Refresh credentials'}).click();await page.getByRole('alert').filter({hasText:'Service temporarily unavailable'}).waitFor();
  mode='normal';await page.getByRole('button',{name:'Try again'}).click();await page.getByRole('heading',{name:credentialID}).waitFor();
  await page.goto(origin+'/dashboard/device?code=ABC123');await page.getByLabel('Authorization code').waitFor();assert.equal(await page.getByLabel('Authorization code').inputValue(),'ABC123');assert.equal(deviceDecision,'');
  await page.getByRole('button',{name:'Deny access'}).click();await page.getByRole('status').filter({hasText:'Authorization denied'}).waitFor();assert.equal(deviceDecision,'denied');
  await page.goto(origin+'/dashboard/device?code=DEF456');await page.getByLabel('Authorization code').waitFor();mode='error';await page.getByRole('button',{name:'Approve access'}).click();await page.getByRole('alert').filter({hasText:'Service temporarily unavailable'}).waitFor();mode='normal';
  checks.push('Credential confirmation and explicit device decision with prefilled code');
  mode='error';await page.goto(origin+'/dashboard/profile');await page.getByRole('alert').filter({hasText:'Service temporarily unavailable'}).waitFor();
  mode='normal';await page.getByRole('button',{name:'Try again'}).click();await page.getByRole('heading',{name:'Profile settings'}).waitFor();
  await page.getByLabel('Name',{exact:true}).fill('Alice');
  await page.getByRole('button',{name:'Save name'}).click();
  await page.getByRole('status').filter({hasText:'Display name saved'}).waitFor();
  assert.equal(displayName,'Alice');
  assert.equal(await page.locator('.account-username').textContent(), 'Alice');
  await page.reload();assert.equal(await page.getByLabel('Name',{exact:true}).inputValue(),'Alice');
  await page.getByLabel('Name',{exact:true}).fill(' changed ');
  assert(await page.getByRole('button',{name:'Save name'}).isDisabled());
  await page.getByRole('button',{name:'Discard changes'}).click();
  assert.equal(await page.getByLabel('Name',{exact:true}).inputValue(),'Alice');
  await page.getByLabel('User slug').fill('alice');
  await page.getByRole('button',{name:'Save user slug'}).click();
  await page.getByRole('status').filter({hasText:'User slug saved'}).waitFor();
  assert.equal(userSlug,'alice');
  for (const width of [320,390,768,800,801,1440]) {
    await page.setViewportSize({width,height:900});
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `Profile overflow at ${width}`);
    assert.deepEqual((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa']).analyze()).violations, []);
    if ([390,1440].includes(width)) await page.screenshot({path: `${artifacts}/profile-${width}.png`,fullPage:true});
  }
  checks.push('Profile deep link, save, reload, validation, discard and responsive layout');
  mode='empty';await page.goto(origin+'/dashboard/applications');await page.getByRole('heading',{name:'No applications yet'}).waitFor();
  mode='error';await page.getByRole('button',{name:'Refresh applications'}).click();await page.getByRole('alert').filter({hasText:'Service temporarily unavailable'}).waitFor();
  assert.equal(await page.getByRole('heading',{name:'No applications yet'}).count(),0);
  mode='normal';await page.getByRole('button',{name:'Try again'}).click();await page.getByRole('heading',{name:'Shared notes'}).waitFor();
  appView='published';await page.getByRole('button',{name:'Refresh applications'}).click();await page.getByText('Published',{exact:true}).waitFor();assert.equal(await page.getByRole('link',{name:/Open app/}).count(),1);
  appView='suspended';await page.getByRole('button',{name:'Refresh applications'}).click();await page.getByText('Suspended',{exact:true}).waitFor();assert.equal(await page.getByRole('link',{name:/Open app/}).count(),0);
  appView='unpublished';appName='薄肌训练工作台';await page.getByRole('button',{name:'Refresh applications'}).click();await page.getByRole('heading',{name:appName}).waitFor();assert(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth));
  await page.locator('h2').evaluate(e => {e.textContent='A-very-long-application-name-'.repeat(15);});
  assert(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth));
  await page.emulateMedia({reducedMotion:'reduce'});assert.equal(await page.locator('button').first().evaluate(e => getComputedStyle(e).transitionDuration),'0s');
  await page.setViewportSize({width:720,height:800});await page.locator('html').evaluate(e => {e.style.fontSize='200%';});assert(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth));
  checks.push('Empty/error/retry, long content, text zoom and reduced motion');
  mode='expired';await page.goto(origin+'/dashboard/device?code=ABC123');await page.getByRole('link',{name:'Continue to sign in'}).waitFor();assert.equal(new URL(page.url()).pathname,'/dashboard/login');
  assert.equal(new URLSearchParams(new URL(page.url()).search).get('return'), '/dashboard/device?code=ABC123');
  assert.equal(new URL((await page.getByRole('link',{name:'Continue to sign in'}).getAttribute('href')) ?? '', origin).searchParams.get('return'), '/dashboard/device?code=ABC123');
  const disconnected = await browser.newContext();const disconnectedPage = await disconnected.newPage();await disconnectedPage.goto(origin+'/dashboard/');await disconnectedPage.getByRole('alert').filter({hasText:'Unable to connect'}).waitFor();await disconnected.close();
  const noScript=await browser.newContext({javaScriptEnabled:false,viewport:{width:320,height:800}});const plain=await noScript.newPage();await plain.goto(origin+'/dashboard/');assert(await plain.getByText(/Enable JavaScript/).isVisible());await noScript.close();
  mode='normal';await page.goto(origin+'/dashboard/applications');await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();await page.setViewportSize({width:1440,height:900});
  await page.locator('.account-avatar img').evaluate(element => element.dispatchEvent(new Event('error')));
  assert.equal(await page.locator('.account-avatar img').count(), 0);
  assert.equal(await page.locator('.account-avatar').textContent(), 'A');
  await page.getByRole('button',{name:'Sign out'}).click();await page.getByRole('link',{name:'Continue to sign in'}).waitFor();assert(loggedOut);
  checks.push('Session expiry, standalone unavailable service, meaningful no-script text');
  assert.deepEqual(failures,[]);
  await writeFile(`${artifacts}/report.json`,JSON.stringify({kind:'UI fixtures and standalone preview; not real control-domain acceptance',checks,failures},null,2));
  console.log(`Web browser checks passed (${checks.length} groups).`);
} finally { await browser?.close();await new Promise<void>((resolve,reject) => server.httpServer.close(e => e ? reject(e) : resolve())); }
