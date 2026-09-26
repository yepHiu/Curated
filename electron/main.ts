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
import { readFileSync } from "node:fs"
import { fileURLToPath } from "node:url"
import type { DesktopInfo, DesktopUpdateResult } from "./desktop-contract.js"
import { checkDesktopUpdate, desktopInfoChannel, desktopUpdateChannel, isTrustedDesktopSender, isVersion } from "./desktop-updates.js"

import { startBackend, type ManagedBackend } from "./backend-process.js"
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
        app.dock?.setIcon(appIconPath)
      }
      registerDesktopIpc()
      managedBackend = await startBackend({
        appPath: app.getAppPath(),
        env: process.env,
        isPackaged: app.isPackaged,
      })
      if (shouldStartDevFrontend({ isPackaged: app.isPackaged })) {
        managedFrontend = await startFrontend({
          appPath: app.getAppPath(),
          backendBaseUrl: managedBackend.baseUrl,
          env: process.env,
        })
      }
      rendererBaseUrl = managedFrontend?.baseUrl ?? managedBackend.baseUrl
      createAppTray(managedBackend, rendererBaseUrl)
      mainWindow = createMainWindow(rendererBaseUrl, appIconPath, rendererBaseUrl, managedBackend.baseUrl)
    })
    .catch((error: unknown) => {
      console.error("Curated Electron startup failed", error)
      app.quit()
    })
}

app.on("activate", () => {
  if (BrowserWindow.getAllWindows().length === 0 && managedBackend) {
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
  initialUrl = baseUrl,
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
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      preload: path.join(__dirname, "preload.cjs"),
    },
  })

  window.once("ready-to-show", () => {
    window.show()
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
    void shell.openExternal(url)
    return { action: "deny" }
  })
  window.webContents.on("will-navigate", (event, url) => {
    if (isAllowedAppUrl(url, baseUrl)) {
      return
    }
    event.preventDefault()
    void shell.openExternal(url)
  })

  installDesktopClientMarker(window, backendBaseUrl)
  void window.loadURL(withCuratedDesktopVersion(initialUrl, app.getVersion()))
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

function createAppTray(backend: ManagedBackend, appUrl: string): void {
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
  refreshTrayMenu(backend, appUrl)
}

function refreshTrayMenu(backend: ManagedBackend, appUrl: string): void {
  if (!appTray) {
    return
  }
  const menuModel = buildTrayMenuModel({
    baseUrl: appUrl,
    attachedToExistingBackend: backend.attachedToExisting,
  })
  appTray.setContextMenu(Menu.buildFromTemplate(menuModel.map((item) => toElectronTrayMenuItem(item))))
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
        void shell.openExternal(url)
      }
      break
    case "open-settings":
      showMainWindow(url)
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
  if (!managedBackend || !rendererBaseUrl) {
    return
  }
  if (!mainWindow || mainWindow.isDestroyed()) {
    mainWindow = createMainWindow(rendererBaseUrl, appIconPath, initialUrl ?? rendererBaseUrl, managedBackend.baseUrl)
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
    if (!isTrustedDesktopSender(event.sender.id, mainWindow?.webContents.id, event.senderFrame?.url, event.senderFrame === event.sender.mainFrame, rendererBaseUrl)) {
      throw new Error("Untrusted Desktop IPC sender")
    }
  }
  ipcMain.handle(desktopInfoChannel, (event) => {
    assertSender(event)
    return desktopInfo
  })
  ipcMain.handle(desktopUpdateChannel, (event) => {
    assertSender(event)
    pendingDesktopCheck ??= checkDesktopUpdate(desktopInfo, desktopRelease.updateFeed, net.fetch.bind(net)).finally(() => {
      pendingDesktopCheck = undefined
    })
    return pendingDesktopCheck
  })
  ipcMain.handle(pickDirectoryChannel, async () => {
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
