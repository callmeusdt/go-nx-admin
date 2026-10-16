// Component-level check for DataTable. Run: /root/playwright/run frontend/check-data-table.mjs
import { createServer } from 'vite'
import { firefox } from '/root/playwright/node_modules/playwright/index.mjs'
import assert from 'node:assert/strict'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const root = path.dirname(fileURLToPath(import.meta.url))
const server = await createServer({ root, configFile: false, plugins: [react()], server: { host: '127.0.0.1', port: 19591, strictPort: true }, logLevel: 'error' })
await server.listen()
const browser = await firefox.launch({ headless: true })
const results = []
const ok = name => results.push(name)

try {
  const page = await browser.newPage()
  page.on('pageerror', e => console.error('pageerror:', e.message))
  await page.goto('http://127.0.0.1:19591/check-data-table.html')
  await page.evaluate(() => localStorage.clear())
  await page.reload()
  await page.locator('table').waitFor()

  const rowCount = () => page.locator('tbody tr').count()
  const state = () => page.evaluate(() => window.__state)
  const headers = () => page.locator('thead th').allTextContents()

  // Column visibility defaults
  assert.equal(await rowCount(), 3); ok('renders rows')
  let h = await headers()
  assert.ok(!h.includes('Secret'), 'hidden-by-default column must not render'); ok('defaultVisible=false column hidden')

  // Column picker: required column is not offered, toggling shows the column and persists.
  await page.getByRole('button', { name: /Columns|列设置/ }).click()
  const labels = await page.locator('label').allTextContents()
  assert.ok(labels.some(l => l.includes('Secret')) && !labels.some(l => l.includes('Ops')), 'picker lists optional columns only')
  await page.getByLabel('Secret').check()
  assert.ok((await headers()).includes('Secret'))
  await page.reload(); await page.locator('table').waitFor()
  assert.ok((await headers()).includes('Secret'), 'column choice persists across reload'); ok('column picker + persistence')

  // Row 1 is disabled; select-all must skip it.
  const boxes = page.locator('tbody input[type=checkbox]')
  assert.equal(await boxes.nth(0).isDisabled(), true); ok('isRowDisabled disables the checkbox')
  await page.locator('thead input[type=checkbox]').check()
  assert.deepEqual((await state()).sel, [2, 3]); ok('select-all skips disabled rows')
  assert.equal(await page.locator('thead input[type=checkbox]').evaluate(e => e.indeterminate), false)
  assert.equal(await page.locator('thead input[type=checkbox]').isChecked(), true)

  // Partial selection shows indeterminate.
  await boxes.nth(2).uncheck()
  assert.equal(await page.locator('thead input[type=checkbox]').evaluate(e => e.indeterminate), true); ok('indeterminate header on partial selection')

  // Selection survives paging; select-all on page 2 does not drop page-1 picks.
  await page.getByRole('button', { name: /Next|下一页/ }).click()
  assert.equal((await state()).page, 2)
  assert.deepEqual((await state()).sel, [2], 'page change keeps selection')
  await page.locator('thead input[type=checkbox]').check()
  assert.deepEqual((await state()).sel.sort(), [2, 4, 5, 6]); ok('selection persists across pages')
  await page.locator('thead input[type=checkbox]').uncheck()
  assert.deepEqual((await state()).sel, [2], 'unselect-all on a page only removes that page'); ok('unselect-all only affects current page')

  // Batch actions receive the keys; clear resets.
  await page.locator('#batch').click()
  assert.deepEqual(await page.evaluate(() => window.__batch), [2])
  assert.deepEqual((await state()).sel, []); ok('batch action gets keys and can clear')
  assert.equal(await page.locator('#batch').count(), 0, 'toolbar hides when nothing selected')

  // Sorting cycles asc -> desc -> none.
  const idBtn = page.locator('thead th button', { hasText: 'ID' })
  await idBtn.click(); assert.deepEqual((await state()).sort, { field: 'id', order: 'asc' })
  assert.equal(await page.locator('thead th').first().getAttribute('aria-sort') ?? await page.locator('thead th[aria-sort]').getAttribute('aria-sort'), 'ascending')
  await idBtn.click(); assert.deepEqual((await state()).sort, { field: 'id', order: 'desc' })
  await idBtn.click(); assert.equal((await state()).sort, null); ok('sort cycles asc/desc/off with aria-sort')

  // Non-sortable column renders no button.
  assert.equal(await page.locator('thead th button', { hasText: 'Name' }).count(), 0); ok('non-sortable header has no button')

  console.log('PASS\n - ' + results.join('\n - '))
} catch (e) {
  console.error('FAIL', e.message)
  process.exitCode = 1
} finally {
  await browser.close()
  await server.close()
}
