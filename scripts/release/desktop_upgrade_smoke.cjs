/* Run two real Desktop releases against one isolated profile and a bounded local Server fixture. */
const { createRequire } = require('node:module')
const { mkdtempSync, rmSync, readFileSync } = require('node:fs')
const { tmpdir } = require('node:os')
const { createServer } = require('node:http')
const path = require('node:path')
const assert = require('node:assert/strict')
const { _electron } = createRequire(require.resolve('@playwright/test/package.json'))('playwright')
;(async () => {
  const profile = mkdtempSync(path.join(tmpdir(), 'curated-upgrade-'))
  let modern = false
  const server = createServer((req, res) => {
    res.setHeader('Content-Type', 'application/json')
    if (req.url === '/api/health') return res.end(JSON.stringify({ name: 'curated', version: modern ? '1.7.0' : '1.6.0' }))
    if (req.url === '/api/server-info') {
      if (!modern) { res.statusCode = 423; return res.end('{"error":{"code":"AUTH_LOCKED"}}') }
      return res.end(JSON.stringify({ product: 'curated-server', serverId: '18a31111-36e0-424b-9f9f-d1e9cfb74ed2', name: 'Upgrade fixture', version: '1.7.0', protocolVersion: 1, desktopBridgeVersion: 1 }))
    }
    res.setHeader('Content-Type', 'text/html')
    res.end('<!doctype html><html><body>Upgrade fixture</body></html>')
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  const url = `http://127.0.0.1:${server.address().port}`
  const launch = executablePath => _electron.launch({ executablePath, args: ['--user-data-dir=' + profile], env: { ...process.env, CURATED_ELECTRON_BACKEND_URL: '', CURATED_BACKEND_URL: '' } })
  const connected = async app => {
    const deadline = Date.now() + 25000
    while (Date.now() < deadline) {
      const page = app.windows().find(p => p.url().startsWith(url))
      if (page) { await page.waitForLoadState(); return page }
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    throw new Error('Desktop did not connect to the compatible Server')
  }
  let app
  try {
    app = await launch(process.argv[2])
    const launcher = await app.firstWindow()
    await launcher.locator('#name').fill('Saved NAS')
    await launcher.locator('#url').fill(url)
    await launcher.locator('#server-form').evaluate(form => form.requestSubmit())
    await launcher.getByRole('button', { name: '连接', exact: true }).click()
    const old = await connected(app)
    await old.evaluate(() => { localStorage.setItem('upgrade-marker', 'preserved'); document.cookie = 'upgrade=preserved; Max-Age=3600; SameSite=Lax' })
    const userData = await app.evaluate(({ app }) => app.getPath('userData'))
    await app.close(); app = undefined
    const original = JSON.parse(readFileSync(path.join(userData, 'servers.json')))
    // Upgrade Desktop first while Server still uses the old PIN-locked API.
    app = await launch(process.argv[3])
    let current = await connected(app)
    assert.equal(await current.evaluate(() => localStorage.getItem('upgrade-marker')), 'preserved')
    assert.match(await current.evaluate(() => document.cookie), /upgrade=preserved/)
    await app.close(); app = undefined
    modern = true
    // Then upgrade Server, assigning its UUID for the first time.
    app = await launch(process.argv[3])
    current = await connected(app)
    assert.equal(await current.evaluate(() => localStorage.getItem('upgrade-marker')), 'preserved')
    assert.match(await current.evaluate(() => document.cookie), /upgrade=preserved/)
    const final = JSON.parse(readFileSync(path.join(userData, 'servers.json')))
    assert.equal(final.lastServerId, original.lastServerId)
    assert.equal(final.servers[0].id, original.servers[0].id)
    assert.equal(final.servers[0].name, 'Saved NAS')
    assert.equal(final.servers[0].serverId, '18a31111-36e0-424b-9f9f-d1e9cfb74ed2')
    console.log('Desktop 0.1.0 → 0.2.0; Server 1.6.0 → 1.7.0: records, localStorage and cookies preserved')
  } finally {
    if (app) await app.close()
    await new Promise(resolve => server.close(resolve))
    rmSync(profile, { recursive: true, force: true })
  }
})().catch(error => { console.error(error); process.exitCode = 1 })
