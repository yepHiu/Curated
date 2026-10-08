// Runs the production playback bridge against an isolated synthetic Server.
const { app, BrowserWindow } = require("electron")
const assert = require("node:assert/strict")
const { createServer } = require("node:http")
const { readFileSync, writeFileSync, readdirSync } = require("node:fs")
const path = require("node:path")
const { pathToFileURL } = require("node:url")
const directory = process.env.CURATED_MAC_QA_DIR
if (!directory || !path.isAbsolute(directory)) throw new Error("QA_DIRECTORY_REQUIRED")
app.setPath("userData", path.join(directory, "profile"))
const root = process.cwd()
const delay = ms => new Promise(resolve => setTimeout(resolve, ms))
async function until(check, timeout = 8000) {
  const end = Date.now() + timeout
  while (!check()) { if (Date.now() >= end) throw new Error("QA_CONDITION_TIMEOUT"); await delay(30) }
}
let playback, server
const limit = setTimeout(() => { console.error("MAC_QA_TIMEOUT"); app.exit(2) }, 45000)
app.whenReady().then(async () => {
  const { installDesktopPlayback } = await import(pathToFileURL(path.join(root, "electron-dist/playback-desktop.js")).href)
  const bytes = readFileSync(path.join(directory, "fixture.mp4"))
  const requests = [], socketPrefix = `curated-mpv-${process.pid}-`
  let rangeRequests = 0
  server = createServer((req, res) => {
    const url = new URL(req.url, "http://localhost")
    if (url.pathname === "/") { res.setHeader("Content-Type", "text/html"); return res.end("<title>Curated Mac QA</title>") }
    if (req.headers.cookie !== "qa=fixture") { res.writeHead(403).end(); return }
    if (url.pathname.endsWith("/stream")) {
      assert.ok(["p1", "p2"].includes(url.searchParams.get("fileId")))
      const range = req.headers.range?.match(/^bytes=(\d+)-(\d*)$/)
      const start = range ? Number(range[1]) : 0
      const end = range?.[2] ? Math.min(Number(range[2]), bytes.length - 1) : bytes.length - 1
      if (range) rangeRequests++
      res.writeHead(range ? 206 : 200, { "Content-Type": "video/mp4", "Accept-Ranges": "bytes", "Content-Length": end - start + 1,
        ...(range ? { "Content-Range": `bytes ${start}-${end}/${bytes.length}` } : {}) })
      return res.end(bytes.subarray(start, end + 1))
    }
    let body = ""
    req.on("data", chunk => { body += chunk })
    req.on("end", () => requests.push({ path: req.url, body }))
    res.setHeader("Content-Type", "application/json")
    if (url.pathname === "/api/auth/status") return res.end(JSON.stringify({ unlocked: true }))
    if (url.pathname.endsWith("/playback-session")) return res.end(JSON.stringify({ mode: "direct", fileId: "p1", resumePositionSec: 2, durationSec: 30 }))
    if (url.pathname === "/api/library/movies/qa") return res.end(JSON.stringify({ code: "MAC-QA", title: "macOS 原生播放 · 合成测试",
      files: [{ id: "p1", fileName: "fixture.mp4" }, { id: "p2", fileName: "fixture-part-2.mp4" }] }))
    res.statusCode = 204; res.end()
  })
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve))
  const origin = `http://127.0.0.1:${server.address().port}`
  const main = new BrowserWindow({ show: false, webPreferences: { preload: path.join(root, "electron-dist/preload.cjs"), sandbox: true, contextIsolation: true } })
  await main.webContents.session.cookies.set({ url: origin, name: "qa", value: "fixture", httpOnly: true })
  playback = installDesktopPlayback({ directory: path.join(root, "electron-dist"), appPath: root, userData: app.getPath("userData"), packaged: false,
    current: () => ({ window: main, renderer: origin, origin, generation: "qa" }), focusMain: () => {} })
  await main.loadURL(origin)
  assert.equal((await main.webContents.executeJavaScript("window.javLibrary.playback.capabilities()")).available, true)
  await main.webContents.executeJavaScript('window.javLibrary.playback.open({movieId:"qa",autoplay:true,locale:"zh-CN"})')
  await until(() => playback.snapshot().state.positionSec > 2.2 && playback.snapshot().state.diagnostics?.videoOutput === "libmpv")
  const player = BrowserWindow.getAllWindows().find(window => window !== main)
  assert.ok(player)
  assert.equal(await player.webContents.executeJavaScript("window.curatedPlayer.windowChrome"), "macos")
  const command = value => player.webContents.executeJavaScript(`(async()=>{const s=await window.curatedPlayer.snapshot(); await window.curatedPlayer.command(s.sessionId,${JSON.stringify(value)})})()`)
  await command({ action: "pause" })
  await until(() => playback.snapshot().state.status === "paused")
  await command({ action: "seek", value: 8 })
  await until(() => Math.abs(playback.snapshot().state.positionSec - 8) < 0.1)
  await command({ action: "speed", value: 1.5 })
  await command({ action: "volume", value: 35 })
  await until(() => playback.snapshot().state.speed === 1.5 && playback.snapshot().state.volume === 35)
  const capture = await player.webContents.executeJavaScript("(async()=>{const s=await window.curatedPlayer.snapshot(); return window.curatedPlayer.prepareCapture(s.sessionId)})()")
  assert.equal(capture.phase, "prepared")
  assert.ok(Math.abs(capture.positionSec - 8) < 0.1)
  assert.ok(capture.preview.startsWith("data:image/png;base64,"))
  writeFileSync(path.join(directory, "captured-frame.png"), Buffer.from(capture.preview.split(",")[1], "base64"))
  await command({ action: "fullscreen" })
  await until(() => playback.snapshot().fullscreen)
  await delay(300)
  await command({ action: "fullscreen" })
  await until(() => !playback.snapshot().fullscreen)
  await delay(300)
  player.setSize(800, 600)
  await delay(150)
  assert.equal(player.getBounds().width, 800)
  await command({ action: "minimize" })
  await until(() => player.isMinimized())
  await command({ action: "focus" })
  await until(() => !player.isMinimized() && player.isVisible())
  await command({ action: "part", fileId: "p2" })
  await until(() => playback.snapshot().state.fileId === "p2" && playback.snapshot().state.status === "playing")
  await until(() => playback.snapshot().state.diagnostics?.videoOutput === "libmpv")
  const state = playback.snapshot().state
  assert.ok(rangeRequests >= 2)
  player.close() // Native red close button uses the same close event and final progress write.
  await until(() => !playback.snapshot().windowOpen)
  assert.ok(player.isDestroyed())
  await until(() => requests.some(item => item.path.includes("/api/playback/progress/qa?fileId=p2")))
  await until(() => !readdirSync("/tmp").some(name => name.startsWith(socketPrefix)))
  writeFileSync(path.join(directory, "result.json"), JSON.stringify({ state, rangeRequests, progressSaved: true, socketsRemoved: true, closed: true }))
  main.destroy(); server.closeAllConnections(); server.close(); clearTimeout(limit); app.exit(0)
}).catch(async error => {
  console.error(error)
  if (playback) await playback.stop().catch(() => {})
  app.exit(1)
})
