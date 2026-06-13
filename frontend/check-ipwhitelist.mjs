import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  const res = await fetch('http://172.18.0.1:19500/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: 'admin123' }),
  });
  const data = await res.json();
  const token = data.token;

  await page.addInitScript((t) => {
    localStorage.setItem('auth_token', t);
    localStorage.setItem('username', 'admin');
  }, token);

  await page.goto('http://172.18.0.1:19500/users', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);

  // Click the Shield icon button (IP白名单) for the first user
  const shieldButtons = await page.locator('button[title="IP白名单"]');
  if (await shieldButtons.count() > 0) {
    await shieldButtons.first().click();
    await page.waitForTimeout(500);
  }

  await page.screenshot({ path: '/tmp/opencode/nx-ipwhitelist.png', fullPage: true });
  console.log('screenshot saved to /tmp/opencode/nx-ipwhitelist.png');

  // Also take a screenshot of the account menu dialog
  // First close the IP whitelist dialog
  await page.goto('http://172.18.0.1:19500/dashboard', { waitUntil: 'networkidle' });
  await page.waitForTimeout(500);
  
  // Click user menu to open dropdown
  await page.click('button:has-text("admin")');
  await page.waitForTimeout(300);
  
  // Click 账号管理
  await page.click('text=账号管理');
  await page.waitForTimeout(500);
  
  await page.screenshot({ path: '/tmp/opencode/nx-account-dialog.png', fullPage: true });
  console.log('screenshot saved to /tmp/opencode/nx-account-dialog.png');

  await browser.close();
})();
