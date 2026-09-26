/* Exercise the real installed Electron shell; no browser test runner or production data. */
const { createRequire } = require('node:module')
const { mkdtempSync, rmSync } = require('node:fs')
const { tmpdir } = require('node:os')
const path = require('node:path')
const assert = require('node:assert/strict')
const { _electron } = createRequire(require.resolve('@playwright/test/package.json'))('playwright')
;(async () => {
  const profile = mkdtempSync(path.join(tmpdir(), 'curated-desktop-check-'))
  const app = await _electron.launch({
    executablePath: process.argv[2], args: ['--user-data-dir=' + profile],
    env: { ...process.env, CURATED_ELECTRON_BACKEND_URL: '', CURATED_BACKEND_URL: '' },
  })
  try {
    const page = await app.firstWindow()
    await page.getByText('还没有服务器，请添加一个连接。').waitFor({ state: 'visible' })
    await page.getByRole('button', { name: '检查 Desktop 更新' }).waitFor({ state: 'visible' })
    const metadata = await app.evaluate(({ app }) => ({ packaged: app.isPackaged, version: app.getVersion() }))
    assert.equal(metadata.packaged, true)
    assert.match(metadata.version, /^\d+\.\d+\.\d+$/)
    if (process.env.CURATED_SMOKE_SERVER) {
      await page.locator('#name').fill('Smoke Server')
      await page.locator('#url').fill(process.env.CURATED_SMOKE_SERVER)
      await page.locator('#server-form').evaluate(form => form.requestSubmit())
      await page.getByRole('button', { name: '连接', exact: true }).click()
      const deadline = Date.now() + 30000
      while (!app.windows().some(window => window.url().startsWith(process.env.CURATED_SMOKE_SERVER))) {
        if (Date.now() > deadline) throw new Error('Desktop did not open the independent Server')
        await new Promise(resolve => setTimeout(resolve, 100))
      }
    }
    console.log(JSON.stringify({ ...metadata, standaloneStartup: 'passed' }))
  } finally {
    await app.close()
    rmSync(profile, { recursive: true, force: true })
  }
})().catch(error => { console.error(error); process.exitCode = 1 })
