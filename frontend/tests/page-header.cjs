const { firefox } = require('playwright')
const assert = require('node:assert/strict')
const base = process.env.NX_PAGE_HEADER_QA_URL || 'http://127.0.0.1:19514'
if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(base)) throw Error('Loopback fixture required')

;(async () => {
  const browser = await firefox.launch({ headless: true })
  try {
    const context = await browser.newContext()
    const page = await context.newPage({ viewport: { width: 320, height: 240 } })
    await page.goto(`${base}/tests/page-header.html`)

    const header = page.getByTestId('header')
    assert.deepEqual(await header.locator('button').allTextContents(), ['Create', 'Batch', 'Refresh'])
    assert.equal(await header.locator('[role="separator"]').count(), 1)
    assert.equal(await header.locator('button').nth(1).isDisabled(), true)

    await page.getByRole('button', { name: 'Toggle secondary' }).click()
    assert.deepEqual(await header.locator('button').allTextContents(), ['Create', 'Batch'])
    assert.equal(await header.locator('[role="separator"]').count(), 0)

    await page.getByRole('button', { name: 'Toggle secondary' }).click()
    await page.getByRole('button', { name: 'Toggle RTL' }).click()
    assert.equal(await page.locator('main').getAttribute('dir'), 'rtl')
    assert.deepEqual(await header.locator('button').allTextContents(), ['Create', 'Batch', 'Refresh'])
    process.stdout.write('page-header grouped actions/order/visibility/rtl=PASS\n')
  } finally { await browser.close() }
})().catch(error => { process.stderr.write(`${error.stack}\n`); process.exitCode = 1 })
