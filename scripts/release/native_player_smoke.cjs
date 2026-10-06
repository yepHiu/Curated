/* Exercise the installed Windows player with synthetic authenticated HTTP/Range media. */
const { createRequire } = require('node:module')
const { execFileSync } = require('node:child_process')
const { mkdtempSync, readFileSync, rmSync } = require('node:fs')
const { createServer } = require('node:http')
const { tmpdir } = require('node:os')
const path = require('node:path')
const assert = require('node:assert/strict')
const { _electron } = createRequire(require.resolve('@playwright/test/package.json'))('playwright')

;(async () => {
  assert.equal(process.platform, 'win32')
  const executable = path.resolve(process.argv[2])
  const temporary = mkdtempSync(path.join(tmpdir(), 'curated-native-package-'))
  const video = path.join(temporary, 'synthetic.mp4')
  execFileSync('ffmpeg', ['-hide_banner', '-loglevel', 'error', '-f', 'lavfi', '-i', 'testsrc2=size=320x180:rate=30',
    '-f', 'lavfi', '-i', 'sine=frequency=440:sample_rate=48000', '-t', '20', '-c:v', 'libx264', '-preset', 'ultrafast',
    '-g', '30', '-c:a', 'aac', '-movflags', '+faststart', video], { windowsHide: true })
  const bytes = readFileSync(video)
  let ranges = 0
  let captures = 0
  const server = createServer(async (req, res) => {
    const route = new URL(req.url, 'http://localhost')
    const send = value => { res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(value)) }
    if (route.pathname === '/api/server-info') return send({ product: 'curated-server', serverId: '00000000-0000-0000-0000-000000000058', name: 'Packaged native smoke', version: '1.7.10', protocolVersion: 1, desktopBridgeVersion: 1 })
    if (route.pathname === '/') {
      res.setHeader('Set-Cookie', 'nativeSmoke=authenticated; Path=/; HttpOnly; SameSite=Lax')
      res.setHeader('Content-Type', 'text/html')
      return res.end('<!doctype html><html><title>Synthetic native package check</title><body>Native smoke fixture</body></html>')
    }
    if (!req.headers.cookie?.includes('nativeSmoke=authenticated')) return res.writeHead(403).end()
    if (route.pathname === '/api/auth/status') return send({ unlocked: true, pinEnabled: true })
    if (route.pathname === '/api/library/movies/fixture') return send({ id: 'fixture', code: 'TEST-058', title: 'Synthetic package check', actors: [], files: [{ id: 'one', fileName: 'one.mp4' }, { id: 'two', fileName: 'two.mp4' }] })
    if (route.pathname.endsWith('/playback-session')) return send({ mode: 'direct', fileId: route.searchParams.get('fileId'), resumePositionSec: 0, durationSec: 20 })
    if (route.pathname.endsWith('/stream')) {
      const range = req.headers.range?.match(/^bytes=(\d+)-(\d*)$/)
      const start = range ? Number(range[1]) : 0
      const end = range?.[2] ? Math.min(Number(range[2]), bytes.length - 1) : bytes.length - 1
      if (range) ranges++
      res.writeHead(range ? 206 : 200, { 'Content-Type': 'video/mp4', 'Accept-Ranges': 'bytes',
        'Content-Length': end - start + 1, ...(range ? { 'Content-Range': `bytes ${start}-${end}/${bytes.length}` } : {}) })
      return res.end(req.method === 'HEAD' ? undefined : bytes.subarray(start, end + 1))
    }
    if (route.pathname === '/api/curated-frames') {
      const chunks = []
      for await (const chunk of req) chunks.push(chunk)
      const form = await new Request('http://localhost', { method: 'POST', headers: { 'Content-Type': req.headers['content-type'] }, body: Buffer.concat(chunks) }).formData()
      const image = Buffer.from(await form.get('image').arrayBuffer())
      assert.equal(image.readUInt32BE(16), 320)
      assert.equal(image.readUInt32BE(20), 180)
      captures++
      return res.writeHead(204).end()
    }
    if (route.pathname.includes('/progress/') || route.pathname.includes('/played-movies/') || route.pathname === '/api/playback/watch-time/daily') return res.writeHead(204).end()
    res.writeHead(404).end()
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  const origin = `http://127.0.0.1:${server.address().port}`
  const wait = async (operation, predicate, message) => {
    const deadline = Date.now() + 20000
    while (Date.now() < deadline) {
      const value = await operation()
      if (predicate(value)) return value
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    throw new Error(message)
  }
  let app
  try {
    const system = process.env.SystemRoot || 'C:\\Windows'
    app = await _electron.launch({ executablePath: executable, args: ['--user-data-dir=' + path.join(temporary, 'profile')],
      env: { ...process.env, PATH: `${system}\\System32;${system}`, CURATED_NATIVE_MPV: path.join(temporary, 'missing-development-mpv.exe'), CURATED_ELECTRON_BACKEND_URL: '', CURATED_BACKEND_URL: '' } })
    const launcher = await app.firstWindow()
    await launcher.locator('#server-address').fill(origin)
    await launcher.locator('form').evaluate(form => form.requestSubmit())
    const page = await wait(async () => app.windows().find(window => window.url().startsWith(origin)), Boolean, 'Fixture did not connect')
    await page.waitForLoadState()
    const caps = await wait(() => page.evaluate(() => window.javLibrary.playback.capabilities()), value => value.available, 'Packaged native capability missing')
    assert.equal(caps.available, true)
    await page.evaluate(() => window.javLibrary.playback.open({ movieId: 'fixture', fileId: 'one', startSec: 3, autoplay: false }))
    const snapshot = () => page.evaluate(() => window.javLibrary.playback.snapshot())
    const first = await wait(snapshot, value => value.state.status === 'paused' && value.state.durationSec > 19, 'Bundled player did not load')
    const command = input => page.evaluate(({ session, input }) => window.javLibrary.playback.command(session, input), { session: first.sessionId, input })
    await command({ action: 'resume' })
    await wait(snapshot, value => value.state.positionSec > 3.5 && value.state.status === 'playing', 'Bundled media clock did not advance')
    await command({ action: 'pause' })
    await command({ action: 'seek', value: 10 })
    await wait(snapshot, value => Math.abs(value.state.positionSec - 10) < 0.5, 'Bundled seek failed')
    const playerPage = await wait(async () => app.windows().find(window => window.url().includes('/player/index.html')), Boolean, 'Native controls missing')
    await wait(async () => {
      try { return await playerPage.evaluate(session => window.curatedPlayer.capture(session), first.sessionId) }
      catch (error) { if (String(error).includes('CAPTURE_NOT_READY')) return undefined; throw error }
    }, Boolean, 'Bundled frame capture never became ready')
    assert.equal(captures, 1)
    const candidates = JSON.parse(execFileSync('powershell.exe', ['-NoProfile', '-Command',
      `ConvertTo-Json -Compress -InputObject @(Get-CimInstance Win32_Process -Filter "Name='mpv.exe' OR Name='native-player-host.exe'" | Select-Object ProcessId,ExecutablePath)`], { encoding: 'utf8', windowsHide: true }))
    const children = candidates.filter(child => child.ExecutablePath?.toLowerCase().startsWith(path.dirname(executable).toLowerCase() + path.sep))
    assert.equal(children.length, 2)
    const bundled = path.join(path.dirname(executable), 'resources', 'app', 'native-player', 'mpv.exe')
    assert.ok(children.some(child => child.ExecutablePath.toLowerCase() === bundled.toLowerCase()), 'Player used an external engine')
    await command({ action: 'part', fileId: 'two' })
    const second = await wait(snapshot, value => value.state.fileId === 'two' && value.sessionId !== first.sessionId, 'Part replacement failed')
    assert.equal(second.windowOpen, true)
    await page.evaluate(session => window.javLibrary.playback.command(session, { action: 'close' }), second.sessionId)
    await wait(snapshot, value => !value.windowOpen, 'Native close did not finish')
    await app.close()
    app = undefined
    for (const child of children) {
      assert.throws(() => process.kill(child.ProcessId, 0), 'Native child survived close')
    }
    assert.ok(ranges > 0)
    console.log(JSON.stringify({ packagedNative: 'passed', engineFromInstall: true, isolatedPath: true, authenticatedRanges: ranges, capture: 'passed', seek: 'passed', partReplacement: 'passed', cleanup: 'passed' }))
  } finally {
    if (app) await app.close()
    server.closeAllConnections()
    await new Promise(resolve => server.close(resolve))
    rmSync(temporary, { recursive: true, force: true })
  }
})().catch(error => { console.error(error); process.exitCode = 1 })
