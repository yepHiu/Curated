import {
  app,
  BrowserWindow,
  dialog,
  ipcMain,
  net,
  Menu,
  shell,
  Tray,
  type MenuItemConstructorOptions,
  type OpenDialogOptions,
} from "electron"
import path from "node:path"
import { existsSync, readFileSync } from "node:fs"
import { fileURLToPath, pathToFileURL } from "node:url"
import type { DesktopInfo, DesktopUpdateResult } from "./desktop-contract.js"
import { checkDesktopUpdate, desktopInfoChannel, desktopUpdateChannel, isTrustedDesktopSender, isVersion } from "./desktop-updates.js"

import { normalizeServerUrl, probeServer, serverSessionPartition, ServerConnectionStore, type SavedServer } from "./server-connections.js"

import { defaultBackendBaseUrl, startBackend, type ManagedBackend } from "./backend-process.js"
import {
  buildTrayMenuModel,
  shouldMarkCuratedDesktopRequest,
  pickDirectoryChannel,
  resolveAppIconPath,
  selectedDirectoryFromOpenDialogResult,
  shouldHideWindowOnClose,
  withCuratedDesktopVersion,
  withCuratedDesktopRequestHeaders,
  shouldStopBackendOnQuit,
  shouldUseApplicationMenu,
  type TrayMenuActionId,
  type TrayMenuModelItem,
} from "./desktop-shell.js"
import {
  shouldStartDevFrontend,
  shouldStopFrontendOnQuit,
  startFrontend,
  type ManagedFrontend,
} from "./frontend-process.js"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const desktopRelease = JSON.parse(readFileSync(path.join(__dirname, "desktop-release.json"), "utf8"))
if (desktopRelease.schema !== 1 || !isVersion(desktopRelease.version) || typeof desktopRelease.buildStamp !== "string" || !/^\d{8}\.\d{6}$/.test(desktopRelease.buildStamp) || !["legacy", "desktop"].includes(desktopRelease.distribution) || !(desktopRelease.updateFeed === null || typeof desktopRelease.updateFeed === "string")) {
  throw new Error("Invalid Desktop release metadata")
}
const desktopInfo: DesktopInfo = {
  version: desktopRelease.version,
  buildStamp: desktopRelease.buildStamp,
  development: !app.isPackaged,
  distribution: desktopRelease.distribution,
  platform: process.platform === "darwin" ? "macos" : process.platform === "win32" ? "windows" : process.platform,
  arch: process.arch,
}
let pendingDesktopCheck: Promise<DesktopUpdateResult> | undefined

let mainWindow: BrowserWindow | undefined
let connectionWindow: BrowserWindow | undefined
let loadingConnection: { window: BrowserWindow; renderer: string; server: string } | undefined
let serverStore: ServerConnectionStore | undefined
let connectionError = ""
let connecting = false
let currentServerUrl: string | undefined
const connectionPageUrl = pathToFileURL(path.join(__dirname, "connections.html")).href
let managedBackend: ManagedBackend | undefined
let managedFrontend: ManagedFrontend | undefined
let rendererBaseUrl: string | undefined
let appIconPath: string | undefined
let appTray: Tray | undefined
let isQuitting = false
let isSystemSessionEnding = false

const singleInstanceLock = app.requestSingleInstanceLock()
if (!singleInstanceLock) {
  app.quit()
} else {
  if (process.platform === "win32") {
    app.setAppUserModelId("com.curated.desktop")
  }

  app.on("second-instance", () => {
    showMainWindow()
  })

  app.whenReady()
    .then(async () => {
      if (!shouldUseApplicationMenu(process.platform)) {
        Menu.setApplicationMenu(null)
      }
      appIconPath = resolveAppIconPath(app.getAppPath())
      if (process.platform === "darwin" && appIconPath) {
        // Dock artwork needs transparent margins to match other macOS app icons.
        const dockIconPath = path.join(app.getAppPath(), "public", "Curated-icon-macos.png")
        app.dock?.setIcon(existsSync(dockIconPath) ? dockIconPath : appIconPath)
      }
      registerDesktopIpc()
      createAppTray()
      installApplicationMenu()
      showConnections()
      try {
        serverStore = new ServerConnectionStore(path.join(app.getPath("userData"), "servers.json"))
        const state = serverStore.snapshot()
        const last = state.servers.find(s => s.id === state.lastServerId)
        const explicitUrl = process.env.CURATED_ELECTRON_BACKEND_URL || process.env.CURATED_BACKEND_URL
        const useLegacyLocal = desktopInfo.distribution === "legacy" && !explicitUrl && (last?.url === defaultBackendBaseUrl(app.isPackaged) || state.servers.length === 0)
        if (!useLegacyLocal && (last || explicitUrl)) {
          const target = last ?? serverStore.save({ name: "Curated Server", url: normalizeServerUrl(explicitUrl) })
          await connectServer(target, false)
        } else if (useLegacyLocal) {
          connecting = true
          managedBackend = await startBackend({ appPath: app.getAppPath(), env: process.env, isPackaged: app.isPackaged })
          if (shouldStartDevFrontend({ isPackaged: app.isPackaged })) {
            managedFrontend = await startFrontend({ appPath: app.getAppPath(), backendBaseUrl: managedBackend.baseUrl, env: process.env })
          }
          const local = last ?? serverStore.save({ name: "本机服务器", url: managedBackend.baseUrl })
          connecting = false
          await connectServer(local, false)
        }
      } catch (error) {
        connecting = false
        connectionError = error instanceof Error ? error.message : "连接失败，请重试。"
        showConnections()
      }
    })
    .catch((error: unknown) => {
      console.error("Curated Electron startup failed", error)
      app.quit()
    })
}

app.on("activate", () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    showMainWindow()
  }
})

app.on("window-all-closed", () => {
  if (process.platform !== "darwin" && isQuitting) {
    app.quit()
  }
})

app.on("before-quit", () => {
  isQuitting = true
})

app.on("will-quit", async (event) => {
  if (!managedBackend && !managedFrontend) {
    return
  }
  event.preventDefault()
  const backend = managedBackend
  const frontend = managedFrontend
  managedBackend = undefined
  managedFrontend = undefined
  rendererBaseUrl = undefined
  if (frontend && shouldStopFrontendOnQuit({ attachedToExistingFrontend: frontend.attachedToExisting })) {
    await frontend.stop()
  }
  if (backend && shouldStopBackendOnQuit({ attachedToExistingBackend: backend.attachedToExisting })) {
    await backend.stop()
  }
  app.exit(0)
})

function createMainWindow(
  baseUrl: string,
  iconPath?: string,
  backendBaseUrl = baseUrl,
): BrowserWindow {
  const window = new BrowserWindow({
    width: 1280,
    height: 820,
    minWidth: 960,
    minHeight: 640,
    autoHideMenuBar: true,
    show: false,
    title: "Curated",
    ...(process.platform === "darwin" ? {
      titleBarStyle: "hiddenInset" as const,
      trafficLightPosition: { x: 24, y: 20 },
    } : {}),
    ...(iconPath ? { icon: iconPath } : {}),
    webPreferences: {
      partition: serverSessionPartition(backendBaseUrl),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      preload: path.join(__dirname, "preload.cjs"),
    },
  })

  window.on("query-session-end", () => {
    isSystemSessionEnding = true
    isQuitting = true
  })
  window.on("session-end", () => {
    isSystemSessionEnding = true
    isQuitting = true
  })
  window.on("close", (event) => {
    if (!shouldHideWindowOnClose({ isQuitting, isSystemSessionEnding })) {
      return
    }
    event.preventDefault()
    window.hide()
  })
  window.on("closed", () => {
    if (mainWindow === window) {
      mainWindow = undefined
    }
  })

  window.webContents.setWindowOpenHandler(({ url }) => {
    openExternalUrl(url)
    return { action: "deny" }
  })
  const guardNavigation = (event: Electron.Event, url: string) => {
    if (isAllowedAppUrl(url, baseUrl)) {
      return
    }
    event.preventDefault()
    openExternalUrl(url)
  }
  window.webContents.on("will-navigate", guardNavigation)
  window.webContents.on("will-redirect", guardNavigation)
  window.webContents.on("did-fail-load", (_event, code, _description, _url, isMainFrame) => {
    if (isMainFrame && code !== -3 && mainWindow === window) {
      connectionError = "页面加载失败，请重试或连接其他服务器。"
      showConnections()
    }
  })

  installDesktopClientMarker(window, backendBaseUrl)
  return window
}

function installDesktopClientMarker(window: BrowserWindow, backendBaseUrl: string): void {
  window.webContents.session.webRequest.onBeforeSendHeaders((details, callback) => {
    if (!shouldMarkCuratedDesktopRequest(details.url, backendBaseUrl)) {
      callback({ requestHeaders: details.requestHeaders })
      return
    }
    callback({
      requestHeaders: withCuratedDesktopRequestHeaders(details.requestHeaders, desktopInfo.version),
    })
  })
}

function createAppTray(): void {
  if (appTray || !appIconPath) {
    return
  }
  appTray = new Tray(appIconPath)
  appTray.setToolTip("Curated - local media library")
  appTray.on("click", () => {
    showMainWindow()
  })
  appTray.on("double-click", () => {
    showMainWindow()
  })
  refreshTrayMenu()
}

function refreshTrayMenu(): void {
  if (!appTray) {
    return
  }
  const menuModel = buildTrayMenuModel({
    baseUrl: rendererBaseUrl ?? "http://127.0.0.1:8081",
    attachedToExistingBackend: !managedBackend || managedBackend.attachedToExisting,
  })
  appTray.setContextMenu(Menu.buildFromTemplate(menuModel.map((item) => {
    const result = toElectronTrayMenuItem(item)
    if (item.type !== "separator" && ["open-browser", "open-settings"].includes(item.id)) result.enabled = Boolean(rendererBaseUrl)
    if (item.type !== "separator" && item.id === "backend-status") result.label = currentServerUrl ? `当前服务器：${currentServerUrl}` : "未连接服务器"
    return result
  })))
}

function toElectronTrayMenuItem(item: TrayMenuModelItem): MenuItemConstructorOptions {
  if (item.type === "separator") {
    return { type: "separator" }
  }
  return {
    label: item.label,
    enabled: item.enabled,
    click: item.enabled === false ? undefined : () => handleTrayMenuAction(item.id, item.url),
  }
}

function handleTrayMenuAction(actionId: TrayMenuActionId, url?: string): void {
  switch (actionId) {
    case "open-curated":
      showMainWindow()
      break
    case "open-browser":
      if (url) {
        openExternalUrl(url)
      }
      break
    case "open-settings":
      showMainWindow(url)
      break
    case "open-servers":
      showConnections()
      break
    case "quit":
      quitFromTray()
      break
    case "app-title":
    case "backend-status":
      break
  }
}

function showMainWindow(initialUrl?: string): void {
  if (!mainWindow || mainWindow.isDestroyed() || !rendererBaseUrl) {
    showConnections()
    return
  }
  if (initialUrl) {
    void mainWindow.loadURL(withCuratedDesktopVersion(initialUrl, app.getVersion()))
  }
  if (mainWindow.isMinimized()) {
    mainWindow.restore()
  }
  mainWindow.show()
  mainWindow.focus()
}

function quitFromTray(): void {
  isQuitting = true
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.destroy()
  }
  app.quit()
}

function registerDesktopIpc(): void {
  const assertSender = (event: Electron.IpcMainInvokeEvent) => {
    const loading = loadingConnection?.window.webContents.id === event.sender.id ? loadingConnection : undefined
    if (!isTrustedDesktopSender(event.sender.id, loading?.window.webContents.id ?? mainWindow?.webContents.id, event.senderFrame?.url, event.senderFrame === event.sender.mainFrame, loading?.renderer ?? rendererBaseUrl)) {
      throw new Error("Untrusted Desktop IPC sender")
    }
  }
  ipcMain.handle(desktopInfoChannel, (event) => {
    assertSender(event)
    return { ...desktopInfo, serverOrigin: loadingConnection?.window.webContents.id === event.sender.id ? loadingConnection.server : currentServerUrl }
  })
  ipcMain.handle(desktopUpdateChannel, (event) => {
    assertSender(event)
    pendingDesktopCheck ??= checkDesktopUpdate(desktopInfo, desktopRelease.updateFeed, net.fetch.bind(net)).finally(() => {
      pendingDesktopCheck = undefined
    })
    return pendingDesktopCheck
  })
  ipcMain.handle("curated:open-servers", (event) => {
    assertSender(event)
    showConnections()
  })
  ipcMain.handle("curated:connections", async (event, action: unknown, value: unknown) => {
    if (event.sender.id !== connectionWindow?.webContents.id || event.senderFrame !== event.sender.mainFrame || event.senderFrame?.url !== connectionPageUrl) throw new Error("Untrusted connection manager")
    try {
      if (action === "list") return { ok: true, state: serverStore?.snapshot() ?? { schema: 1, servers: [] }, currentServerUrl, connecting, error: connectionError }
      if (!serverStore) throw new Error(connectionError || "无法读取服务器列表。")
      if (connecting) throw new Error("正在连接，请稍候。")
      connectionError = ""
      if (action === "save") {
        const input = value as Partial<SavedServer> | null
        const existing = serverStore.snapshot().servers.find(s => s.id === input?.id)
        if (existing && existing.url === currentServerUrl && normalizeServerUrl(input?.url) !== currentServerUrl) throw new Error("请先切换到其他服务器，再修改当前连接的地址。")
        serverStore.save(value)
      } else if (action === "remove") {
        const target = serverStore.snapshot().servers.find(s => s.id === value)
        if (!target) throw new Error("服务器记录不存在。")
        if (target.url === currentServerUrl) throw new Error("请先切换到其他服务器，再删除当前连接。")
        serverStore.remove(target.id)
      } else if (action === "connect") {
        const target = serverStore.snapshot().servers.find(s => s.id === value)
        if (!target) throw new Error("服务器记录不存在。")
        await connectServer(target, true)
      } else throw new Error("未知操作。")
      return { ok: true }
    } catch (error) {
      return { ok: false, error: error instanceof Error ? error.message : "操作失败，请重试。" }
    }
  })
  ipcMain.handle(pickDirectoryChannel, async (event) => {
    assertSender(event)
    if (event.sender.id !== mainWindow?.webContents.id || !currentServerUrl || !["localhost", "127.0.0.1", "[::1]"].includes(new URL(currentServerUrl).hostname)) throw new Error("远程连接不可选择本机目录。")
    const owner = BrowserWindow.getFocusedWindow() ?? mainWindow
    const options: OpenDialogOptions = {
      title: "Select folder",
      properties: ["openDirectory"],
    }
    const result = owner ? await dialog.showOpenDialog(owner, options) : await dialog.showOpenDialog(options)
    return selectedDirectoryFromOpenDialogResult(result)
  })
}

function isAllowedAppUrl(candidate: string, baseUrl: string): boolean {
  try {
    return new URL(candidate).origin === new URL(baseUrl).origin
  } catch {
    return false
  }
}

function openExternalUrl(url: string): void {
  try {
    if (["https:", "http:"].includes(new URL(url).protocol)) void shell.openExternal(url)
  } catch { /* Ignore malformed external destinations. */ }
}

function showConnections(): void {
  if (connectionWindow && !connectionWindow.isDestroyed()) {
    connectionWindow.show(); connectionWindow.focus()
    return
  }
  connectionWindow = new BrowserWindow({
    width: 760, height: 720, minWidth: 480, minHeight: 500,
    title: "Curated · 服务器", autoHideMenuBar: true,
    ...(appIconPath ? { icon: appIconPath } : {}),
    webPreferences: { contextIsolation: true, nodeIntegration: false, sandbox: true, preload: path.join(__dirname, "connections-preload.cjs") },
  })
  connectionWindow.webContents.setWindowOpenHandler(() => ({ action: "deny" }))
  connectionWindow.webContents.on("will-navigate", event => event.preventDefault())
  connectionWindow.on("closed", () => {
    connectionWindow = undefined
    if (!mainWindow && !connecting && !appTray) app.quit()
  })
  void connectionWindow.loadFile(path.join(__dirname, "connections.html"))
}

function installApplicationMenu(): void {
  if (!shouldUseApplicationMenu(process.platform)) return
  Menu.setApplicationMenu(Menu.buildFromTemplate([
    { label: "Curated", submenu: [{ label: "服务器…", click: () => showConnections() }, { type: "separator" }, { role: "quit" }] },
    { role: "editMenu" }, { role: "viewMenu" }, { role: "windowMenu" },
  ]))
}

async function connectServer(target: SavedServer, confirm: boolean): Promise<void> {
  if (connecting) throw new Error("正在连接，请稍候。")
  connecting = true
  connectionError = ""
  let candidate: BrowserWindow | undefined
  try {
    if (confirm && mainWindow) {
      const result = await dialog.showMessageBox(connectionWindow ?? mainWindow, {
        type: "question", title: "切换服务器", message: `连接到“${target.name}”？`,
        detail: "当前页面将关闭，播放、上传及未保存的编辑会中断。服务器已接收的后台任务继续运行。",
        buttons: ["取消", "切换服务器"], defaultId: 0, cancelId: 0,
      })
      if (result.response !== 1) return
    }
    await probeServer(target.url)
    const renderer = target.url === managedBackend?.baseUrl ? managedFrontend?.baseUrl ?? target.url : target.url
    candidate = createMainWindow(renderer, appIconPath, target.url)
    loadingConnection = { window: candidate, renderer, server: target.url }
    let pageStatus = 200
    candidate.webContents.on("did-navigate", (_event, _url, httpResponseCode) => { pageStatus = httpResponseCode })
    let timeout: ReturnType<typeof setTimeout> | undefined
    try {
      await Promise.race([
        candidate.loadURL(withCuratedDesktopVersion(renderer, app.getVersion())).catch(() => { throw new Error("无法加载服务器页面，请检查服务是否运行并重试。") }),
        new Promise<never>((_resolve, reject) => { timeout = setTimeout(() => reject(new Error("页面加载超时，请重试。")), 15000) }),
      ])
    } finally { clearTimeout(timeout) }
    if (pageStatus >= 400) throw new Error(`服务器页面返回 ${pageStatus}，请检查 Server 的 Web 界面是否已部署。`)
    if (!isAllowedAppUrl(candidate.webContents.getURL(), renderer)) throw new Error("服务器页面跳转到了其他地址。")
    serverStore!.remember(target.id)
    const previous = mainWindow
    mainWindow = candidate
    currentServerUrl = target.url
    rendererBaseUrl = renderer
    previous?.destroy()
    mainWindow.show()
    mainWindow.focus()
    connectionWindow?.close()
    refreshTrayMenu()
  } catch (error) {
    candidate?.destroy()
    throw error
  } finally { loadingConnection = undefined; connecting = false }
}
