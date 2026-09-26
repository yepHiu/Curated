import { app, BrowserWindow, dialog, ipcMain, Menu, nativeImage, session, shell, Tray, type IpcMainInvokeEvent } from "electron"
import path from "node:path"
import { fileURLToPath, pathToFileURL } from "node:url"
import { checkDesktopUpdate } from "./updates.js"
import { DesktopPreferencesStore, proxyConfiguration, validatePreferences, type DesktopPreferences, type DesktopSettings } from "./settings.js"
import { discoverServers } from "./discovery.js"
import { ConnectionStore, connectionPartition, localServerSuggestion, normalizeServerUrl, probeServer, type SavedConnection } from "./connections.js"
import { resolveAppIconPath, withCuratedDesktopVersion, withCuratedDesktopRequestHeaders } from "./desktop-shell.js"

const directory = path.dirname(fileURLToPath(import.meta.url))
const launcherPath = path.join(directory, "launcher", "index.html")
const launcherURL = pathToFileURL(launcherPath).href
let launcher: BrowserWindow | undefined
let library: BrowserWindow | undefined
let tray: Tray | undefined
let appIcon: Electron.NativeImage | undefined
let store: ConnectionStore
let current: SavedConnection | undefined
let attempt: AbortController | undefined
let quitting = false
let discoveryScan: AbortController | undefined
let preferencesStore: DesktopPreferencesStore
let runningPreferences: DesktopPreferences
let networkSession: Electron.Session
let savingSettings = false

// 展示名称不应改变已使用的连接档案和 Electron 会话目录。
const existingUserData = app.getPath("userData")
app.setName("Curated")
app.setPath("userData", existingUserData)

// Desktop owns only its windows and discovery sockets, never Server processes.
if (!app.requestSingleInstanceLock()) app.quit()
else {
  app.on("second-instance", () => showWindow())
  app.whenReady().then(async () => {
    store = new ConnectionStore(app.getPath("userData"))
    preferencesStore = new DesktopPreferencesStore(app.getPath("userData"))
    runningPreferences = preferencesStore.read()
    networkSession = session.fromPartition("curated-connection-probes")
    await networkSession.setProxy(proxyConfiguration(runningPreferences))
    registerIPC()
    const icon = resolveAppIconPath(app.getAppPath())
    if (icon) {
      // 图标解码失败不应阻止离线连接页；macOS 菜单栏需使用小尺寸图标。
      const trayImage = nativeImage.createFromPath(icon)
      if (!trayImage.isEmpty()) {
        // Dock 使用原始品牌图标，不能复用菜单栏缩小后的位图。
        appIcon = trayImage
        if (process.platform === "darwin") app.dock?.setIcon(appIcon)
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
/** 打开本地连接窗口；macOS 原生红黄绿按钮叠放在应用内容顶部。 */
function showLauncher(): void {
  if (!launcher || launcher.isDestroyed()) {
    launcher = new BrowserWindow({ width: 520, height: 740, minWidth: 520, minHeight: 540, title: "Curated Desktop", icon: appIcon,
      ...(process.platform === "darwin" ? { titleBarStyle: "hidden" as const, trafficLightPosition: { x: 20, y: 16 } } : {}),
      webPreferences: {
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
/** Windows 开发启动需传项目目录，macOS 登录启动由开发 bundle 内入口承接。 */
function loginOptions(): Electron.LoginItemSettingsOptions {
  return { path: process.execPath, args: !app.isPackaged && process.platform === "win32" ? [app.getAppPath()] : [] }
}
/** 查询 OS 的真实登录启动状态，并提示网络配置是否仍待重启。 */
function readDesktopSettings(): DesktopSettings {
  const preferences = preferencesStore.read()
  const loginSupported = process.platform === "darwin" || process.platform === "win32"
  return { ...preferences, loginSupported, launchAtLogin: loginSupported && app.getLoginItemSettings(loginOptions()).openAtLogin,
    restartRequired: JSON.stringify(preferences) !== JSON.stringify(runningPreferences) }
}
/** 身份探测与更新检查共用 Chromium 代理，且不携带业务会话 Cookie。 */
const desktopFetch: typeof fetch = (input, init) => networkSession.fetch(input instanceof URL ? input.href : input, { ...init, credentials: "omit" })
function registerIPC(): void {
  // 仅本地连接窗口可读取或更改客户端配置。
  ipcMain.handle("curated:settings-read", event => { requireLauncher(event); return readDesktopSettings() })
  // 保存过程串行；网络配置下次启动应用，避免中断正在播放或上传的页面。
  ipcMain.handle("curated:settings-save", (event, value: unknown) => {
    requireLauncher(event)
    if (savingSettings) throw new Error("设置正在保存，请稍后重试。")
    const preferences = validatePreferences(value)
    const launchAtLogin = (value as Record<string, unknown>).launchAtLogin
    if (typeof launchAtLogin !== "boolean") throw new Error("无效的自启动设置。")
    const previous = readDesktopSettings()
    if (!previous.loginSupported && launchAtLogin) throw new Error("此平台暂不支持登录启动。")
    savingSettings = true
    try {
      if (previous.loginSupported && launchAtLogin !== previous.launchAtLogin) {
        app.setLoginItemSettings({ ...loginOptions(), openAtLogin: launchAtLogin })
        if (app.getLoginItemSettings(loginOptions()).openAtLogin !== launchAtLogin) throw new Error("系统未应用登录启动设置，请检查系统登录项。")
      }
      preferencesStore.write(preferences)
      return readDesktopSettings()
    } catch (error) {
      if (previous.loginSupported) app.setLoginItemSettings({ ...loginOptions(), openAtLogin: previous.launchAtLogin })
      throw error
    } finally { savingSettings = false }
  })
  ipcMain.handle("curated:desktop-update", async event => {
    requireLauncher(event)
    const update = await checkDesktopUpdate(desktopFetch)
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
    const info = await probeServer(url, controller.signal, desktopFetch)
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
    candidate = new BrowserWindow({ width: 1280, height: 820, minWidth: 960, minHeight: 640, show: false, title: "Curated", icon: appIcon, webPreferences: {
      partition: connectionPartition(selected), preload: path.join(directory, "preload.cjs"), sandbox: true, contextIsolation: true, nodeIntegration: false,
    } })
    const window = candidate
    await window.webContents.session.setProxy(proxyConfiguration(runningPreferences))
    controller.signal.throwIfAborted()
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
