import assert from 'node:assert/strict';
import { chromium } from 'playwright-core';

const control = process.env.TEST_CONTROL_ORIGIN!;
const appURL = process.env.TEST_APP_URL!;
const appID = process.env.TEST_APP_ID!;
const browser = await chromium.launch({
  executablePath: process.env.CHROME_PATH ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  headless: true,
  args: ['--host-resolver-rules=MAP *.localhost 127.0.0.1', `--ignore-certificate-errors-spki-list=${process.env.TEST_BROWSER_SPKI}`],
});
try {
  const context = await browser.newContext({ignoreHTTPSErrors: false});
  await context.addCookies([{name: '__Host-owner', value: process.env.TEST_OWNER_COOKIE!, url: control, secure: true, httpOnly: true}]);
  const page = await context.newPage();
  await page.goto(control + '/profile');
  await page.getByRole('heading', {name: 'Profile settings'}).waitFor();
  await page.getByLabel('User slug').fill('alice2');
  await page.getByRole('button', {name: 'Save user slug'}).click();
  await page.getByRole('status').filter({hasText: 'User slug saved'}).waitFor();
  await page.goto(control + '/applications/' + appID);
  await page.getByLabel('Application slug').fill('journal');
  await page.getByRole('button', {name: 'Save application slug'}).click();
  await page.getByRole('status').filter({hasText: 'Application slug saved'}).waitFor();
  const currentURL = appURL.replace('notes-alice.', 'journal-alice2.');
  await page.getByText(currentURL).waitFor();
  await page.goto(currentURL + '/');
  const me = await page.goto(currentURL + '/_cellapp/me');
  assert.equal(me?.status(), 200);
  const cookies = await context.cookies();
  assert(cookies.some(cookie => cookie.name === '__Host-cellapp' && cookie.domain === 'journal-alice2.apps.localhost' && cookie.secure && cookie.httpOnly));
  assert(!cookies.some(cookie => cookie.name === '__Host-cellapp' && cookie.domain === 'notes-alice.apps.localhost'));
  const legacy = appURL.replace('notes-alice.', `${appID}.`);
  await page.goto(legacy + '/');
  assert.equal((await page.goto(legacy + '/_cellapp/me'))?.status(), 200);
  assert((await context.cookies()).some(cookie => cookie.name === '__Host-cellapp' && cookie.domain === `${appID}.apps.localhost`));
  console.log('Slug host browser checks passed: settings, login return and host-only cookies.');
} finally {
  await browser.close();
}
