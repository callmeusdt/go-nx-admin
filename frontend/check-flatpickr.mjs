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

  await page.goto('http://172.18.0.1:19500/login-logs', { waitUntil: 'networkidle' });
  await page.waitForSelector('input[placeholder="开始时间"]', { timeout: 10000 });
  await page.click('input[placeholder="开始时间"]');

  await page.waitForSelector('.flatpickr-calendar', { state: 'visible', timeout: 5000 });
  await page.waitForTimeout(500);

  // Also dump the flatpickr calendar DOM for comparison
  const flatpickrHtml = await page.evaluate(() => {
    const el = document.querySelector('.flatpickr-calendar');
    return el ? el.outerHTML.substring(0, 3000) : 'NOT FOUND';
  });
  console.log('FLATPICKR HTML:', flatpickrHtml);

  // Dump computed styles
  const flatpickrStyles = await page.evaluate(() => {
    const el = document.querySelector('.flatpickr-calendar');
    if (!el) return {};
    const style = window.getComputedStyle(el);
    return {
      fontSize: style.fontSize,
      fontFamily: style.fontFamily,
      borderColor: style.borderColor,
      borderWidth: style.borderWidth,
      backgroundColor: style.backgroundColor,
    };
  });
  console.log('FLATPICKR STYLES:', JSON.stringify(flatpickrStyles));

  // Check day cell styles
  const dayStyles = await page.evaluate(() => {
    const el = document.querySelector('.flatpickr-day');
    if (!el) return {};
    const style = window.getComputedStyle(el);
    return {
      fontSize: style.fontSize,
      maxWidth: style.maxWidth,
      height: style.height,
      lineHeight: style.lineHeight,
      padding: style.padding,
      borderColor: style.borderColor,
    };
  });
  console.log('DAY STYLES:', JSON.stringify(dayStyles));

  await page.screenshot({ path: '/tmp/opencode/nx-flatpickr-check.png', fullPage: true });
  console.log('screenshot saved');
  await browser.close();
})();
