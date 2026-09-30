import { createServer } from 'vite'
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import react from '@vitejs/plugin-react'

const server = await createServer({ configFile: false, plugins: [react()], server: { host: '127.0.0.1', port: 19571, strictPort: true } })
await server.listen()
const browser = await chromium.launch({ headless: true, ...(process.env.CHROME_PATH ? { executablePath: process.env.CHROME_PATH } : {}) })
try {
  for (const mode of ['cookie', 'bearer']) {
    const page = await browser.newPage()
    page.on('pageerror', error => console.error(error.message))
    let status = 200
    let mfa = true
    let calls = 0
    await page.route('**/api/v1/**', async route => {
      if (new URL(route.request().url()).pathname === '/api/v1/auth/me') {
        calls++
        await new Promise(resolve => setTimeout(resolve, 250))
        await route.fulfill({ status, json: { username: 'fixture', mfa_enabled: mfa } })
      } else await route.fulfill({ json: [] })
    })
    await page.route('**/src/main.tsx', route => route.fulfill({ contentType: 'application/javascript', body: `
      import React from '/node_modules/.vite/deps/react.js';
      import ReactDOM from '/node_modules/.vite/deps/react-dom_client.js';
      import {createApp} from '/src/core.tsx';
      import {Route} from '/node_modules/.vite/deps/react-router-dom.js';
      localStorage.setItem('auth_token','fixture');
      const extraRoutes = ['/test-a','/test-b'].map(path => React.createElement(Route,{key:path,path,element:React.createElement('p',null,path)}));
      ReactDOM.createRoot(document.getElementById('root')).render(React.createElement(createApp({sessionMode:'${mode}',extraRoutes})));
    ` }))
    await page.goto('http://127.0.0.1:19571/test-a')
    await page.locator('header').waitFor()
    await page.evaluate(() => {
      window.savedHeader = document.querySelector('header')
      window.go = path => { history.pushState({}, '', path); dispatchEvent(new PopStateEvent('popstate')) }
    })
    for (const path of ['/test-b', '/test-a', '/test-b']) {
      await page.evaluate(path => window.go(path), path)
      await page.waitForTimeout(100)
      assert(await page.evaluate(() => window.savedHeader === document.querySelector('header')), `${mode}: layout removed during revalidation`)
      await page.waitForTimeout(400)
      assert(await page.evaluate(() => window.savedHeader === document.querySelector('header')), `${mode}: layout remounted`)
    }
    if (mode === 'cookie') {
      assert(calls >= 4, 'route session checks must remain active')
      mfa = false
      await page.evaluate(() => window.go('/test-a'))
      await page.waitForTimeout(500)
      assert.equal(await page.locator('header').count(), 0, 'MFA enrollment must gate layout')
      status = 401
      await page.evaluate(() => window.go('/test-b'))
      await page.waitForURL('**/login')
    } else assert.equal(calls, 0)
    console.log(`${mode}: layout identity and auth gates PASS`)
    await page.close()
  }
} finally {
  await browser.close()
  await server.close()
}
