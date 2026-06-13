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
  await page.waitForSelector('input[placeholder="开始时间"]', { timeout: 10000 });
  await page.click('input[placeholder="开始时间"]');
  await page.waitForSelector('.flatpickr-calendar', { state: 'visible', timeout: 5000 });
  await page.waitForTimeout(500);

  const info = await page.evaluate(() => {
    const cal = document.querySelector('.flatpickr-calendar');
    const day = document.querySelector('.flatpickr-day');
    const prev = document.querySelector('.flatpickr-prev-month');
    const next = document.querySelector('.flatpickr-next-month');
    const monday = document.querySelector('.flatpickr-weekday');
    const cmonth = document.querySelector('.flatpickr-current-month');
    
    const s = (el) => el ? window.getComputedStyle(el) : null;
    const calS = s(cal);
    const dayS = s(day);
    const prevS = s(prev);
    const nextS = s(next);
    const monS = s(monday);
    const cmonS = s(cmonth);
    
    return {
      calendar: calS ? {
        boxShadow: calS.boxShadow,
        border: calS.border,
        borderWidth: calS.borderWidth,
        borderColor: calS.borderColor,
        backgroundColor: calS.backgroundColor,
        width: calS.width,
        height: calS.height,
        padding: calS.padding,
      } : null,
      day: dayS ? {
        width: dayS.width,
        height: dayS.height,
        fontSize: dayS.fontSize,
        lineHeight: dayS.lineHeight,
      } : null,
      prev: prevS ? { color: prevS.color, fill: prevS.fill, width: prevS.width } : null,
      next: nextS ? { color: nextS.color, fill: nextS.fill, width: nextS.width } : null,
      monday: monS ? { color: monS.color, fontWeight: monS.fontWeight } : null,
      cmonth: cmonS ? { color: cmonS.color, fontSize: cmonS.fontSize } : null,
    };
  });
  console.log(JSON.stringify(info, null, 2));
  await browser.close();
})();
