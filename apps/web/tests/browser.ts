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
let deviceDecision = '';
let browser;
try {
  browser = await chromium.launch({ executablePath, headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();
  page.on('pageerror', error => failures.push(error.message));
  page.on('request', request => { if (!request.url().startsWith(`${origin}/`)) failures.push(`External request: ${request.url()}`); });
  // UI fixtures only: actual authentication, transactions and HTTPS are checked separately.
  await page.route('**/api/console/**', async route => {
    const path = new URL(route.request().url()).pathname.slice('/api/console'.length);
    const method = route.request().method();
    if (mode === 'expired') { await route.fulfill({status: 401, json: {error: 'login_required'}}); return; }
    if (method !== 'GET') {
      mutations++;
      if (mode === 'lost') { await route.abort('failed'); return; }
      if (path.endsWith('/key')) { await route.fulfill({json: {key}}); return; }
      if (path === '/device/decision') { deviceDecision = route.request().postDataJSON().decision; await route.fulfill({json: {status: deviceDecision}}); return; }
      await route.fulfill({json: {deleted: true, revoked: true, loggedOut: true}}); return;
    }
    if (path === '/session') { await route.fulfill({json: {ownerId: ownerID, authMode: 'github'}}); return; }
    if (mode === 'error') { await route.fulfill({status: 503, json: {message: 'Service temporarily unavailable.'}}); return; }
    const app = {id: appID, name: 'Shared notes', url: `https://${appID}.apps.localhost`, activeDeployment: null, suspended: false};
    const value = path === '/apps' ? mode === 'empty' ? [] : [app] : path === '/credentials' ? mode === 'empty' ? [] : [{id: credentialID, createdAt:'2026-09-01T00:00:00Z', expiresAt:'2026-10-01T00:00:00Z'}] : {...app, release: null};
    await route.fulfill({json: value});
  });
  await page.goto(origin + '/applications');
  await page.getByRole('heading', {name: 'Applications', exact: true}).waitFor();
  await page.getByRole('heading', {name: 'Shared notes'}).waitFor();
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
  await page.setViewportSize({width:390,height:844});
  await page.getByRole('button',{name:'Open menu'}).focus();await page.keyboard.press('Enter');
  assert.equal(await page.getByRole('button',{name:'Close menu'}).getAttribute('aria-expanded'),'true');
  await page.getByRole('link',{name:'Deployment credentials',exact:true}).focus();await page.keyboard.press('Escape');
  assert(await page.getByRole('button',{name:'Open menu'}).evaluate(e => e === document.activeElement));
  assert(!(await page.getByRole('link',{name:'Deployment credentials',exact:true}).isVisible()));
  await page.reload();await page.getByRole('heading',{name:'Applications',exact:true}).waitFor();await page.keyboard.press('Tab');
  assert(await page.getByRole('link',{name:'Skip to content'}).evaluate(e => e === document.activeElement));
  await page.keyboard.press('Enter');assert(await page.locator('main').evaluate(e => e === document.activeElement));
  checks.push('Keyboard navigation, mobile menu Escape and skip focus');
  await page.getByRole('heading',{name:'Shared notes'}).click();
  await page.getByRole('heading',{name:'Shared notes',exact:true}).waitFor();
  await page.getByText('No release has been published.',{exact:false}).waitFor();
  await page.reload();await page.getByRole('heading',{name:'Shared notes',exact:true}).waitFor();
  await page.getByRole('button',{name:'Reset share key',exact:true}).click();
  await page.getByRole('dialog').waitFor();
  const before = mutations;
  await page.getByRole('button',{name:'Cancel',exact:true}).click();assert.equal(mutations,before);
  assert(await page.getByRole('button',{name:'Reset share key',exact:true}).evaluate(e => e === document.activeElement));
  await page.getByRole('button',{name:'Reset share key',exact:true}).click();await page.getByRole('button',{name:'Confirm',exact:true}).click();
  await page.getByRole('heading',{name:'New share key'}).waitFor();
  assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length),0);
  await page.evaluate('Object.defineProperty(navigator, "clipboard", {configurable:true,value:{writeText: () => Promise.reject(new Error("blocked"))}})');
  await page.getByRole('button',{name:'Copy share key'}).click();
  await page.getByRole('status').filter({hasText:'Copy unavailable'}).waitFor();
  assert(!page.url().includes(key));
  assert.deepEqual((await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa']).analyze()).violations,[]);
  await page.getByRole('button',{name:'Dismiss key'}).click();assert.equal(await page.getByText(key,{exact:true}).count(),0);
  mode='lost';await page.getByRole('button',{name:'Reset share key',exact:true}).click();const sent=mutations;await page.getByRole('button',{name:'Confirm',exact:true}).click();
  await page.getByRole('alert').filter({hasText:'may already have taken effect'}).waitFor();assert.equal(mutations,sent+1);
  checks.push('Details deep link, unpublished state, confirmation/cancel focus, transient key and uncertain write without retry');
  mode='normal';await page.goto(origin+'/credentials');await page.getByRole('button',{name:'Revoke',exact:true}).click();await page.getByRole('button',{name:'Confirm',exact:true}).click();await page.getByRole('status').filter({hasText:'credential revoked'}).waitFor();
  await page.goto(origin+'/device?code=ABC123');await page.getByLabel('Authorization code').waitFor();assert.equal(await page.getByLabel('Authorization code').inputValue(),'ABC123');assert.equal(deviceDecision,'');
  await page.getByRole('button',{name:'Deny access'}).click();await page.getByRole('status').filter({hasText:'Authorization denied'}).waitFor();assert.equal(deviceDecision,'denied');
  checks.push('Credential confirmation and explicit device decision with prefilled code');
  mode='empty';await page.goto(origin+'/applications');await page.getByRole('heading',{name:'No applications yet'}).waitFor();
  mode='error';await page.getByRole('button',{name:'Refresh applications'}).click();await page.getByRole('alert').filter({hasText:'Service temporarily unavailable'}).waitFor();
  assert.equal(await page.getByRole('heading',{name:'No applications yet'}).count(),0);
  mode='normal';await page.getByRole('button',{name:'Try again'}).click();await page.getByRole('heading',{name:'Shared notes'}).waitFor();
  await page.locator('h2').evaluate(e => {e.textContent='A-very-long-application-name-'.repeat(15);});
  assert(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth));
  await page.emulateMedia({reducedMotion:'reduce'});assert.equal(await page.locator('button').first().evaluate(e => getComputedStyle(e).transitionDuration),'0s');
  await page.setViewportSize({width:720,height:800});await page.locator('html').evaluate(e => {e.style.fontSize='200%';});assert(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth));
  checks.push('Empty/error/retry, long content, text zoom and reduced motion');
  mode='expired';await page.goto(origin+'/applications');await page.getByRole('link',{name:'Continue to sign in'}).waitFor();assert.equal(new URL(page.url()).pathname,'/login');
  const disconnected = await browser.newContext();const disconnectedPage = await disconnected.newPage();await disconnectedPage.goto(origin);await disconnectedPage.getByRole('alert').filter({hasText:'Unable to connect'}).waitFor();await disconnected.close();
  const noScript=await browser.newContext({javaScriptEnabled:false,viewport:{width:320,height:800}});const plain=await noScript.newPage();await plain.goto(origin);assert(await plain.getByText(/Enable JavaScript/).isVisible());await noScript.close();
  checks.push('Session expiry, standalone unavailable service, meaningful no-script text');
  assert.deepEqual(failures,[]);
  await writeFile(`${artifacts}/report.json`,JSON.stringify({kind:'UI fixtures and standalone preview; not real control-domain acceptance',checks,failures},null,2));
  console.log(`Web browser checks passed (${checks.length} groups).`);
} finally { await browser?.close();await new Promise<void>((resolve,reject) => server.httpServer.close(e => e ? reject(e) : resolve())); }
