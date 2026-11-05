const { firefox } = require('playwright')
const assert = require('node:assert/strict')
const base = process.env.NX_DATA_TABLE_QA_URL || 'http://127.0.0.1:19514'
if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(base)) throw Error('Loopback fixture required')

;(async () => {
  const browser = await firefox.launch({ headless: true })
  try {
    const context = await browser.newContext()
    const page = await context.newPage({ viewport: { width: 360, height: 320 } })
    await page.goto(`${base}/tests/data-table.html`)
    const table = page.locator('table')
    await table.waitFor()
    assert.equal(await page.getByRole('button', { name: 'Custom page control' }).count(), 1)
    assert.equal(await page.locator('th').first().evaluate(el => getComputedStyle(el).whiteSpace), 'nowrap')
    assert.equal(await page.locator('[role="dialog"]').count(), 1)
    assert.equal(await page.locator('body').evaluate(el => el.scrollWidth <= el.clientWidth), true)
    process.stdout.write('data-table pagination toolbar/header/dialog/narrow layout=PASS\n')
  } finally { await browser.close() }
})().catch(error => { process.stderr.write(`${error.stack}\n`); process.exitCode = 1 })
