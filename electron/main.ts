import { app, BrowserWindow, dialog, ipcMain, Menu, nativeImage, session, shell, Tray, type IpcMainInvokeEvent } from "electron"
import path from "node:path"
import { fileURLToPath, pathToFileURL } from "node:url"
import { checkDesktopUpdate } from "./updates.js"
import { discoverServers } from "./discovery.js"
import { ConnectionStore, connectionPartition, localServerSuggestion, normalizeServerUrl, probeServer, type SavedConnection } from "./connections.js"
import { resolveAppIconPath, withCuratedDesktopVersion, withCuratedDesktopRequestHeaders } from "./desktop-shell.js"

const directory = path.dirname(fileURLToPath(import.meta.url))
const launcherPath = path.join(directory, "launcher", "index.html")
const launcherURL = pathToFileURL(launcherPath).href
let launcher: BrowserWindow | undefined
let library: BrowserWindow | undefined
let tray: Tray | undefined
let store: ConnectionStore
let current: SavedConnection | undefined
let attempt: AbortController | undefined
let quitting = false
let discoveryScan: AbortController | undefined

// Desktop owns only its windows and discovery sockets, never Server processes.
if (!app.requestSingleInstanceLock()) app.quit()
else {
  app.on("second-instance", () => showWindow())
  app.whenReady().then(() => {
    store = new ConnectionStore(app.getPath("userData"))
    registerIPC()
    const icon = resolveAppIconPath(app.getAppPath())
    if (icon) {
      // 图标解码失败不应阻止离线连接页；macOS 菜单栏需使用小尺寸图标。
      const trayImage = nativeImage.createFromPath(icon)
      if (!trayImage.isEmpty()) {
        tray = new Tray(process.platform === "darwin" ? trayImage.resize({ width: 18, height: 18 }) : trayImage)
        tray.setToolTip("Curated Desktop")
        // 点击托盘恢复当前业务窗口或连接页。
        tray.on("click", () => showWindow())
      }
    }
    refreshMenus()
    showLauncher()
  }).catch(error => { dialog.showErrorBox("Curated Desktop", String(error)); app.quit() })
}
app.on("activate", () => showWindow())
app.on("before-quit", () => { quitting = true; attempt?.abort(); discoveryScan?.abort() })
app.on("window-all-closed", () => { if (!tray && process.platform !== "darwin") app.quit() })

function showWindow(): void {
  const window = library && !library.isDestroyed() ? library : launcher
  if (!window || window.isDestroyed()) { showLauncher(); return }
  if (window.isMinimized()) window.restore()
  window.show(); window.focus()
}
function showLauncher(): void {
  if (!launcher || launcher.isDestroyed()) {
    launcher = new BrowserWindow({ width: 720, height: 740, minWidth: 520, minHeight: 540, title: "Curated Desktop", webPreferences: {
      preload: path.join(directory, "launcher-preload.cjs"), contextIsolation: true, nodeIntegration: false, sandbox: true,
    } })
    launcher.webContents.setWindowOpenHandler(() => ({ action: "deny" }))
    launcher.webContents.on("will-navigate", event => event.preventDefault())
    launcher.on("close", event => { if (!quitting && tray) { event.preventDefault(); launcher?.hide() } })
    void launcher.loadFile(launcherPath)
  }
  launcher.show(); launcher.focus()
}
function refreshMenus(): void {
  const template = [
    { label: "Curated Desktop", enabled: false },
    { label: current?.name ?? "未连接服务器", enabled: false },
    { label: "打开 Curated", click: () => showWindow() },
    { label: "连接 / 更换服务器", click: () => showLauncher() },
    { label: "浏览器打开", enabled: !!current, click: () => { if (current) void shell.openExternal(current.url) } },
    { label: "退出 Desktop", click: () => app.quit() },
  ]
  const menu = Menu.buildFromTemplate(template)
  tray?.setContextMenu(menu)
  Menu.setApplicationMenu(menu)
}
function requireLauncher(event: IpcMainInvokeEvent): void {
  if (!launcher || event.sender !== launcher.webContents || event.senderFrame !== launcher.webContents.mainFrame || event.senderFrame.url !== launcherURL) throw new Error("Unauthorized desktop request")
}
function requireLibrary(event: IpcMainInvokeEvent): void {
  if (!library || !current || event.sender !== library.webContents || event.senderFrame !== library.webContents.mainFrame || new URL(event.senderFrame.url).origin !== current.url) throw new Error("Unauthorized desktop request")
}
function registerIPC(): void {
  ipcMain.handle("curated:desktop-update", async event => {
    requireLauncher(event)
    const update = await checkDesktopUpdate()
    if (!update) return "当前发布中没有适用于此设备的 Desktop 安装包。"
    const answer = await dialog.showMessageBox(launcher!, { message: `Curated Desktop ${update.version}`, detail: "打开官方 Desktop 安装包下载。不会更新 Server。", buttons: ["取消", "下载"], cancelId: 0 })
    if (answer.response === 1) await shell.openExternal(update.url)
    return `Desktop ${update.version}`
  })
  ipcMain.handle("curated:discover", async event => {
    requireLauncher(event)
    discoveryScan?.abort()
    const scan = new AbortController()
    discoveryScan = scan
    try { return await discoverServers(scan.signal) }
    finally { if (discoveryScan === scan) discoveryScan = undefined }
  })
  ipcMain.handle("curated:connections", event => {
    requireLauncher(event)
    return { ...store.read(), desktopVersion: app.getVersion(), activeUrl: current?.url, suggestedUrl: localServerSuggestion() }
  })
  ipcMain.handle("curated:connect", async (event, value: unknown) => {
    requireLauncher(event)
    if (typeof value !== "string") throw new Error("Invalid server address")
    return connect(value)
  })
  ipcMain.handle("curated:cancel-connect", event => { requireLauncher(event); attempt?.abort() })
  ipcMain.handle("curated:forget", async (event, value: unknown) => {
    requireLauncher(event)
    if (typeof value !== "string") throw new Error("Invalid server address")
    const url = normalizeServerUrl(value)
    if (url === current?.url) throw new Error("请先更换服务器，再忘记当前连接。")
    const saved = store.read().connections.find(item => item.url === url)
    if (saved) {
      const isolated = session.fromPartition(connectionPartition(saved))
      await isolated.clearStorageData()
      await isolated.clearCache()
    }
    store.forget(url)
  })
  ipcMain.handle("curated:change-server", event => { requireLibrary(event); showLauncher() })
  ipcMain.handle("curated:desktop-info", event => {
    requireLibrary(event)
    return { version: app.getVersion(), bridgeVersion: 1, serverUrl: current?.url, serverPaths: true }
  })
}
async function connect(raw: string): Promise<{ ok: boolean; error?: string }> {
  attempt?.abort()
  const controller = new AbortController()
  attempt = controller
  let candidate: BrowserWindow | undefined
  try {
    const url = normalizeServerUrl(raw)
    const info = await probeServer(url, controller.signal)
    controller.signal.throwIfAborted()
    const previous = store.read().connections.find(item => item.url === url)
    if (previous && previous.serverId !== info.serverId) {
      const answer = await dialog.showMessageBox(launcher!, { type: "warning", message: "此地址的服务器身份已改变", detail: "继续将建立独立会话，需要重新解锁服务器。", buttons: ["取消", "连接新服务器"], defaultId: 0, cancelId: 0 })
      if (answer.response !== 1) return { ok: false, error: "已取消连接。" }
    }
    if (library && !library.isDestroyed()) {
      const answer = await dialog.showMessageBox(launcher!, { type: "question", message: "更换服务器？", detail: "当前播放、上传和未保存内容将中断。已提交的服务器任务继续运行。", buttons: ["取消", "更换"], defaultId: 0, cancelId: 0 })
      if (answer.response !== 1) return { ok: false, error: "已取消更换。" }
    }
    controller.signal.throwIfAborted()
    const selected = { url, serverId: info.serverId, name: info.name }
    candidate = new BrowserWindow({ width: 1280, height: 820, minWidth: 960, minHeight: 640, show: false, title: "Curated", webPreferences: {
      partition: connectionPartition(selected), preload: path.join(directory, "preload.cjs"), sandbox: true, contextIsolation: true, nodeIntegration: false,
    } })
    const window = candidate
    window.webContents.session.setPermissionCheckHandler(() => false)
    window.webContents.session.setPermissionRequestHandler((_contents, _permission, callback) => callback(false))
    window.webContents.session.webRequest.onBeforeSendHeaders((details, callback) => {
      callback({ requestHeaders: new URL(details.url).origin === url ? withCuratedDesktopRequestHeaders(details.requestHeaders, app.getVersion()) : details.requestHeaders })
    })
    const allowed = (target: string) => { try { return new URL(target).origin === url } catch { return false } }
    window.webContents.setWindowOpenHandler(({ url: target }) => {
      if (/^https?:\/\//i.test(target)) void shell.openExternal(target)
      return { action: "deny" }
    })
    window.webContents.on("will-navigate", (event, target) => { if (!allowed(target)) event.preventDefault() })
    window.webContents.on("will-redirect", (event, target) => { if (!allowed(target)) event.preventDefault() })
    window.webContents.on("will-attach-webview", event => event.preventDefault())
    window.webContents.on("did-fail-load", (_event, code, _description, _url, main) => {
      if (main && code !== -3 && window === library) showLauncher()
    })
    window.webContents.on("render-process-gone", () => { if (window === library) showLauncher() })
    window.on("close", event => { if (!quitting && tray) { event.preventDefault(); window.hide() } })
    const abort = () => { if (!window.isDestroyed()) window.destroy() }
    controller.signal.addEventListener("abort", abort, { once: true })
    const timeout = setTimeout(() => controller.abort(), 15000)
    try { await window.loadURL(withCuratedDesktopVersion(url, app.getVersion())) }
    finally { clearTimeout(timeout); controller.signal.removeEventListener("abort", abort) }
    controller.signal.throwIfAborted()
    store.save(selected)
    library?.destroy()
    library = window; current = selected; candidate = undefined
    window.show(); window.focus(); launcher?.hide(); refreshMenus()
    return { ok: true }
  } catch (error) {
    candidate?.destroy()
    return { ok: false, error: controller.signal.aborted ? "连接已取消或超时，请重试。" : error instanceof Error ? error.message : "连接失败，请检查地址与网络。" }
  } finally { if (attempt === controller) attempt = undefined }
}
