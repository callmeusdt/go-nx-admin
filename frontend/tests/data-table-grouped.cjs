const { firefox } = require('playwright')
const assert = require('node:assert/strict')

const base = process.env.NX_DATA_TABLE_GROUPED_URL || 'http://127.0.0.1:19514'
if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(base)) throw Error('Loopback fixture required')

const headerRows = async root => root.locator('thead tr').count()
const leafCount = async root => root.locator('thead th[data-leaf-index]').count()

;(async () => {
  const browser = await firefox.launch({ headless: true })
  try {
    const context = await browser.newContext()
    const page = await context.newPage({ viewport: { width: 520, height: 600 } })
    await page.goto(`${base}/tests/data-table-grouped.html`)
    const root = page.locator('#grouped-fixture')
    await root.locator('table').first().waitFor()
    await page.waitForTimeout(350)

    // Three nested header rows: 会员(2) + 日期 + 充值(2) + 游戏汇总[投注(2) + 盈亏] + 备注 + 操作.
    assert.equal(await headerRows(root), 3)
    assert.equal(await leafCount(root), 10)

    const group = name => root.locator('thead th').filter({ hasText: name })
    const member = group('会员')
    assert.equal(await member.getAttribute('scope'), 'colgroup')
    assert.equal(await member.getAttribute('colspan'), '2')

    const date = group('日期')
    assert.equal(await date.getAttribute('scope'), 'col')
    assert.equal(await date.getAttribute('rowspan'), '3')

    const deposit = group('充值')
    assert.equal(await deposit.getAttribute('colspan'), '2')

    const games = group('游戏汇总')
    assert.equal(await games.getAttribute('colspan'), '3')

    const bet = group('投注')
    assert.equal(await bet.getAttribute('colspan'), '2')

    // Only leaf headers can sort; the active leaf keeps aria-sort.
    const uid = root.locator('thead th[data-leaf-index="0"]')
    assert.equal(await uid.getAttribute('aria-sort'), 'ascending')
    assert.equal(await root.locator('thead th[aria-sort]').count(), 1)
    await uid.getByRole('button').click()
    assert.equal(await uid.getAttribute('aria-sort'), 'descending')
    await uid.getByRole('button').click()
    assert.equal(await uid.getAttribute('aria-sort'), null)

    // Body rows stay flat: one cell per leaf (+ the selection column).
    assert.equal(await root.locator('tbody tr').first().locator('td').count(), 11)

    // A group that fully sits inside the pinned start stays sticky; a group that
    // crosses into the scrolling area must not.
    assert.equal(await member.evaluate(el => getComputedStyle(el).position), 'sticky')
    assert.notEqual(await deposit.evaluate(el => getComputedStyle(el).position), 'sticky')
    assert.equal(await root.locator('tbody tr').first().locator('td').last().evaluate(el => getComputedStyle(el).position), 'sticky')

    // Hiding a sub-group recalculates colspan without leaving holes.
    await root.locator('thead th[data-leaf-index="9"] button').last().click()
    const picker = page.locator('[data-radix-popper-content-wrapper]')
    await picker.waitFor()
    await page.getByRole('checkbox', { name: '充值', exact: true }).click()
    assert.equal(await leafCount(root), 8)
    assert.equal(await group('充值').count(), 0)
    assert.equal(await member.getAttribute('colspan'), '2')
    assert.equal(await games.getAttribute('colspan'), '3')
    assert.equal(await headerRows(root), 3)

    // Hiding a whole first-level group collapses its nested level too.
    await page.getByRole('checkbox', { name: '游戏汇总', exact: true }).click()
    assert.equal(await leafCount(root), 5)
    assert.equal(await group('游戏汇总').count(), 0)
    assert.equal(await headerRows(root), 2)
    assert.equal(await root.locator('tbody tr').first().locator('td').count(), 6)

    // Restoring both groups rebuilds the original header tree.
    await page.getByRole('checkbox', { name: '充值', exact: true }).click()
    await page.getByRole('checkbox', { name: '游戏汇总', exact: true }).click()
    assert.equal(await leafCount(root), 10)
    assert.equal(await headerRows(root), 3)
    assert.equal(await games.getAttribute('colspan'), '3')
    assert.equal(await page.locator('body').evaluate(el => el.scrollWidth <= el.clientWidth), true)

    // RTL keeps logical header order: the first leaf renders rightmost.
    const rtlContext = await browser.newContext()
    const rtlPage = await rtlContext.newPage({ viewport: { width: 520, height: 600 } })
    await rtlPage.goto(`${base}/tests/data-table-grouped.html?rtl=1`)
    const rtlRoot = rtlPage.locator('#grouped-fixture')
    await rtlRoot.locator('table').first().waitFor()
    await rtlPage.waitForTimeout(350)
    assert.equal(await headerRows(rtlRoot), 3)
    assert.equal(await leafCount(rtlRoot), 10)
    const uidBox = await rtlRoot.locator('thead th[data-leaf-index="0"]').boundingBox()
    const accountBox = await rtlRoot.locator('thead th[data-leaf-index="1"]').boundingBox()
    const lastBox = await rtlRoot.locator('thead th[data-leaf-index="9"]').boundingBox()
    assert.ok(uidBox && accountBox && lastBox)
    assert.ok(uidBox.x > accountBox.x, 'UID should be right of 账号 in RTL')
    assert.ok(uidBox.x > lastBox.x, 'first logical column should be rightmost in RTL')
    await rtlContext.close()
    await context.close()

    process.stdout.write('grouped header colspan/rowspan/picker/sticky/RTL=PASS\n')
  } finally { await browser.close() }
})().catch(error => { process.stderr.write(`${error.stack}\n`); process.exitCode = 1 })
