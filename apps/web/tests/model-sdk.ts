import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {build} from 'esbuild';
import {chromium} from 'playwright-core';

const source = fileURLToPath(new URL('../../../skills/packages/sdk/src/index.ts', import.meta.url));
const bundled = await build({entryPoints:[source],bundle:true,platform:'browser',format:'iife',globalName:'CellappSDK',write:false});
const script = bundled.outputFiles[0]?.text;
if (!script) throw new Error('SDK browser bundle missing');
const executablePath = process.env.CHROME_PATH ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const browser = await chromium.launch({executablePath,headless:true});
const origin = 'https://agent.apps.example';
const model = `cellapp:${'a'.repeat(32)}:shared`;
let authorized = false;
let calls = 0;
let chatModel = '';
try {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.route(`${origin}/**`, async route => {
    const url = new URL(route.request().url());
    if (url.pathname === '/') {
      await route.fulfill({contentType:'text/html',body:'<!doctype html><html><head><meta name="cellapp-api-version" content="1.2"></head><body><h1>Agent</h1></body></html>'});
    } else if (url.pathname === '/_cellapp/v1/models') {
      calls++;
      await route.fulfill(authorized ? {json:{object:'list',data:[{id:model,object:'model',created:0,owned_by:'First provider'}]},headers:{Link:`<https://cellapp.ai/dashboard/model-consent?app=${'b'.repeat(32)}>; rel="cellapp-authorize"`}} :
        {status:403,json:{error:{message:'Authorize',type:'model_authorization_required',code:'model_authorization_required',details:{authorizeUrl:`https://cellapp.ai/dashboard/model-consent?app=${'b'.repeat(32)}`}}}});
    } else if (url.pathname === '/_cellapp/v1/chat/completions') {
      chatModel = (route.request().postDataJSON() as {model:string}).model;
      await route.fulfill({json:{choices:[{message:{content:'pong'}}]}});
    } else {
      throw new Error(`Unexpected SDK request: ${route.request().url()}`);
    }
  });
  await page.goto(origin);
  await page.addScriptTag({content:script});
  await page.evaluate(() => {
    const sdk = (window as unknown as {CellappSDK:{createCellappClient:()=>{models:{current:()=>Promise<unknown>}}}}).CellappSDK.createCellappClient();
    (window as unknown as {selected?: unknown}).selected = undefined;
    void sdk.models.current().then(value => { (window as unknown as {selected?: unknown}).selected = value; });
  });
  const dialog = page.getByRole('dialog',{name:'Choose a model for this application'});
  await dialog.waitFor();
  await dialog.getByRole('link',{name:'Authorize a model provider'}).waitFor();
  assert.equal(await dialog.getByRole('button',{name:'Use this model'}).isDisabled(),true);
  authorized = true;
  await dialog.getByRole('button',{name:'Refresh models'}).click();
  await dialog.getByRole('option',{name:'First provider · shared'}).waitFor({state:'attached'});
  await dialog.getByRole('button',{name:'Use this model'}).click();
  await page.waitForFunction(expected => (window as unknown as {selected?: {id:string}}).selected?.id === expected,model);
  assert.equal(await page.evaluate(() => localStorage.getItem('cellapp:model:v1')),model);
  const before = calls;
  await page.evaluate(async () => {
    const sdk = (window as unknown as {CellappSDK:{createCellappClient:()=>{models:{current:()=>Promise<{id:string}>}}}}).CellappSDK.createCellappClient();
    (window as unknown as {again?: string}).again = (await sdk.models.current()).id;
  });
  assert.equal(await page.evaluate(() => (window as unknown as {again?: string}).again),model);
  const answer = await page.evaluate(async () => {
    const sdk = (window as unknown as {CellappSDK:{createCellappClient:()=>{chat:(options:{messages:{role:'user';content:string}[]})=>Promise<{choices:{message:{content:string}}[]}>}}}).CellappSDK.createCellappClient();
    return (await sdk.chat({messages:[{role:'user',content:'ping'}]})).choices[0]?.message.content;
  });
  assert.equal(answer,'pong');
  assert.equal(chatModel,model);
  assert.equal(calls,before+2);
  assert.equal(await dialog.count(),0);
} finally {
  await browser.close();
}
