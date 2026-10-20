const { firefox } = require('playwright')
const assert = require('node:assert/strict')
const base = process.env.NX_DATE_QA_URL || 'http://127.0.0.1:19514'
if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(base)) throw Error('Loopback fixture required')
;(async () => {
  const browser = await firefox.launch({ headless: true })
  try {
    for (const timezoneId of ['UTC', 'Asia/Kolkata', 'America/Los_Angeles']) {
      const context = await browser.newContext({ timezoneId })
      const page = await context.newPage()
      const errors = []
      page.on('pageerror', e => errors.push(e.message))
      await page.goto(`${base}/tests/date-input.html`)
      const date = page.getByRole('textbox', { name: 'Business date', exact: true })
      await date.waitFor()
      assert.equal(await date.inputValue(), '2026-10-20')
      await date.click()
      assert.equal(await page.locator('.flatpickr-calendar.open .flatpickr-time').count(), 0)
      await page.locator('.flatpickr-calendar.open .flatpickr-day[aria-label="October 22, 2026"]').click()
      assert.equal(await page.getByTestId('date').textContent(), '2026-10-22')
      await page.getByRole('button', { name: 'Replace', exact: true }).click()
      assert.equal(await date.inputValue(), '2026-10-21')
      await page.getByRole('button', { name: 'Clear', exact: true }).click()
      assert.equal(await date.inputValue(), '')
      const datetime = page.getByRole('textbox', { name: 'Datetime', exact: true })
      assert.equal(await datetime.inputValue(), '2026-10-20 12:34')
      await datetime.click()
      assert.equal(await page.locator('.flatpickr-calendar.open .flatpickr-time').count(), 1)
      assert.deepEqual(errors, [])
      await context.close()
    }
    console.log('date-input date/datetime/controlled/three-timezones=PASS')
  } finally { await browser.close() }
})().catch(e => { console.error(e); process.exitCode = 1 })
