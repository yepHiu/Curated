import { app, BrowserWindow, dialog, ipcMain, session } from "electron"
import path from "node:path"
import { existsSync } from "node:fs"
import { fileURLToPath, pathToFileURL } from "node:url"
import { NativePlayerLab } from "./native-player-lab.js"
import { serverSessionPartition } from "./server-connections.js"
import type { NativeLabControl, NativeLabStart } from "./native-player-contract.js"

const directory = path.dirname(fileURLToPath(import.meta.url))
const page = pathToFileURL(path.join(directory, "native-player", "index.html")).href
app.setName("Curated Native Playback Prototype")
// 验收可使用绝对临时 profile，默认原型 profile 仍独立于正式 Desktop。
const profile = process.env.CURATED_NATIVE_PROFILE
if (profile && !path.isAbsolute(profile)) throw new Error("INVALID_PROTOTYPE_PROFILE")
app.setPath("userData", profile ?? path.join(app.getPath("appData"), "Curated Native Playback Prototype"))
let window: BrowserWindow | undefined
let disposed = false
const lab = new NativePlayerLab((origin) => {
  // 认证 session 独立于正式 Desktop，只在当前来源请求中发送 Cookie。
  const scoped = session.fromPartition(`${serverSessionPartition(origin)}-native-prototype`)
  return async (url, init) => {
    // Electron session.fetch 使用 HttpOnly Cookie，跨来源媒体由代理禁止。
    if (new URL(url).origin !== origin) throw new Error("INVALID_MEDIA_ORIGIN")
    return await scoped.fetch(url, { ...init, credentials: "include", redirect: "error" })
  }
})

/** 可执行文件只来自本地主进程环境或原生选择器。 */
function setExecutable(candidate: string): void {
  if (!path.isAbsolute(candidate) || path.basename(candidate).toLowerCase() !== "mpv.exe" || !existsSync(candidate)) {
    throw new Error("MPV_EXECUTABLE_INVALID")
  }
  lab.setExecutable(candidate)
}

/** 原型 IPC 仅允许准确的本地控制页及其主 frame。 */
function assertSender(event: Electron.IpcMainInvokeEvent): void {
  if (!window || event.sender !== window.webContents || event.senderFrame !== event.sender.mainFrame || event.senderFrame.url !== page) {
    throw new Error("UNTRUSTED_NATIVE_PLAYER_SENDER")
  }
}

/** 枚举所有允许的操作；每项执行前复查窗口身份。 */
function registerIpc(): void {
  ipcMain.handle("native-lab:status", (event) => {
    // 只返回不含敏感参数的诊断快照。
    assertSender(event)
    return lab.status()
  })
  ipcMain.handle("native-lab:engine", async (event) => {
    // 只允许选择本机 mpv.exe，不提供通用执行桥。
    assertSender(event)
    const selected = await dialog.showOpenDialog(window!, { title: "选择 mpv.exe", properties: ["openFile"], filters: [{ name: "mpv", extensions: ["exe"] }] })
    if (selected.canceled || !selected.filePaths[0]) return false
    setExecutable(selected.filePaths[0])
    return true
  })
  ipcMain.handle("native-lab:connect", (event, origin: string) => {
    // 源切换与当前播放回收共用操作队列。
    assertSender(event)
    return lab.run(() => lab.connect(origin))
  })
  ipcMain.handle("native-lab:unlock", (event, pin: string) => {
    // 不记录 PIN，解锁成功后由 session 持有 HttpOnly Cookie。
    assertSender(event)
    return lab.run(() => lab.unlock(pin))
  })
  ipcMain.handle("native-lab:search", (event, query: string) => {
    // 只允许受限影片检索。
    assertSender(event)
    return lab.run(() => lab.search(query))
  })
  ipcMain.handle("native-lab:detail", (event, movieId: string) => {
    // 文件路径不作为详情返回值进入页面。
    assertSender(event)
    return lab.run(() => lab.detail(movieId))
  })
  ipcMain.handle("native-lab:resume", (event, input: { movieId: string; fileId?: string }) => {
    // 明确 direct 请求避免只查进度也启动 HLS。
    assertSender(event)
    return lab.run(() => lab.resume(input))
  })
  ipcMain.handle("native-lab:start", (event, input: NativeLabStart) => {
    // 主进程根据身份构造媒体地址，页面不能传任意 URL。
    assertSender(event)
    return lab.run(() => lab.start(input))
  })
  ipcMain.handle("native-lab:control", (event, input: NativeLabControl) => {
    // 引擎内部进一步校验动作和数字范围。
    assertSender(event)
    return lab.run(() => lab.control(input))
  })
}

app.whenReady().then(() => {
  // 原型入口独立，生产 main.ts 不加载此模块。
  if (process.platform !== "win32") throw new Error("WINDOWS_REQUIRED")
  if (process.env.CURATED_NATIVE_MPV) setExecutable(process.env.CURATED_NATIVE_MPV)
  registerIpc()
  window = new BrowserWindow({ width: 1000, height: 850, minWidth: 640, minHeight: 600,
    title: "Curated · Windows 原生播放原型", show: false, autoHideMenuBar: true,
    webPreferences: { preload: path.join(directory, "native-player-preload.cjs"), sandbox: true, contextIsolation: true, nodeIntegration: false } })
  window.webContents.setWindowOpenHandler(() => { // 控制页不打开外部窗口。
    return { action: "deny" }
  })
  window.webContents.on("will-navigate", (event) => { // 控制页只加载构建的本地入口。
    event.preventDefault()
  })
  window.webContents.on("render-process-gone", () => { // renderer 异常时停止仍在独立窗口播放的引擎。
    void lab.run(() => lab.stop())
  })
  window.webContents.session.setPermissionRequestHandler((_contents, _permission, callback) => {
    // 原型不申请浏览器设备权限。
    callback(false)
  })
  window.once("ready-to-show", () => { // 先完成页面首帧再展示。
    window?.show()
  })
  void window.loadURL(page)
}).catch(() => { // 启动失败不输出媒体或认证上下文。
  console.error("Native playback prototype startup failed")
  app.exit(1)
})

app.on("window-all-closed", () => { // 独立实验窗口退出即停止播放器。
  app.quit()
})
app.on("before-quit", (event) => { // 保存有效进度并回收本次播放器，避免孤儿进程。
  if (disposed) return
  event.preventDefault()
  disposed = true
  void lab.run(() => lab.stop()).finally(() => { app.exit(0) })
})
