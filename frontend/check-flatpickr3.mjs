import { chromium } from 'playwright';
(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
  const res = await fetch('http://172.18.0.1:19500/api/v1/auth/login', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: 'admin123' }),
  });
  const data = await res.json();
  await page.addInitScript((t) => {
    localStorage.setItem('auth_token', t);
    localStorage.setItem('username', 'admin');
  }, data.token);
  await page.goto('http://172.18.0.1:19500/login-logs', { waitUntil: 'networkidle' });
  await page.click('input[placeholder="开始时间"]');
  await page.waitForSelector('.flatpickr-calendar', { state: 'visible', timeout: 5000 });
  await page.waitForTimeout(500);
  const info = await page.evaluate(() => {
    const select = document.querySelector('.flatpickr-monthDropdown-months');
    const yearInput = document.querySelector('.flatpickr-current-month input.cur-year');
    const timeInput = document.querySelector('.flatpickr-time input');
    const s = (el) => el ? window.getComputedStyle(el) : null;
    const selS = s(select);
    const yearS = s(yearInput);
    const timeS = s(timeInput);
    return {
      select: selS ? { appearance: selS.appearance, backgroundImage: selS.backgroundImage, border: selS.border, padding: selS.padding, fontSize: selS.fontSize } : null,
      year: yearS ? { appearance: yearS.appearance, border: yearS.border, fontSize: yearS.fontSize, height: yearS.height } : null,
      time: timeS ? { appearance: timeS.appearance, border: timeS.border, fontSize: timeS.fontSize, height: timeS.height } : null,
    };
  });
  console.log(JSON.stringify(info, null, 2));
  await browser.close();
})();
