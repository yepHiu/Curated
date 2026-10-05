import { BrowserWindow, ipcMain } from "electron"
import { existsSync } from "node:fs"
import path from "node:path"
import { pathToFileURL } from "node:url"
import { NativePlaybackCoordinator, type PlaybackContext, type PlaybackSurface } from "./playback-coordinator.js"
import { NativeMpvPlayer } from "./native-mpv-player.js"
import { NativePlayerWindow, type NativeWindowState } from "./native-player-window.js"
import { PlaybackPreferencesStore, validatePlaybackPreferences } from "./playback-preferences.js"
import type { DesktopPlaybackCapabilities, DesktopPlaybackCommand, DesktopPlaybackSnapshot } from "./playback-contract.js"
import { isTrustedDesktopSender } from "./desktop-updates.js"

export interface DesktopPlaybackOptions {
  directory: string
  appPath: string
  userData: string
  packaged: boolean
  current(): { window: BrowserWindow; renderer: string; origin: string; generation: string } | undefined
  focusMain(): void
}
/** 生产 bridge 只信任已提交的主窗口或本地控制页，候选连接不能启动播放。 */
export function installDesktopPlayback(options: DesktopPlaybackOptions) {
  const store = new PlaybackPreferencesStore(options.userData)
  let preferences = store.read()
  const page = pathToFileURL(path.join(options.directory, "player", "index.html")).href
  const helper = path.join(options.directory, "native-player-host.exe")
  const executable = options.packaged ? path.join(options.appPath, "native-player", "mpv.exe")
    : process.env.CURATED_NATIVE_MPV || path.join(options.appPath, ".workspace", "native-player", "mpv.exe")
  let overlay: BrowserWindow | undefined
  const capabilities = (): DesktopPlaybackCapabilities => ({
    protocol: 1, available: process.platform === "win32" && existsSync(executable) && existsSync(helper) && existsSync(path.join(options.directory, "player", "index.html")),
    reason: process.platform !== "win32" ? "platform" : !existsSync(executable) ? "engine" : !existsSync(helper) ? "host"
      : !existsSync(path.join(options.directory, "player", "index.html")) ? "page" : undefined,
    preferences: { ...preferences },
  })
  const coordinator = new NativePlaybackCoordinator(executable, createSurface, () => preferences, value => {
    store.write(value); preferences = value
  })
  async function createSurface(): Promise<PlaybackSurface> {
    const host = new NativePlayerWindow()
    let window: BrowserWindow | undefined
    let ready = false
    let disposing = false
    const sync = (state: NativeWindowState) => {
      if (!ready || !window || window.isDestroyed()) return
      if (!state.visible) window.hide()
      else if (state.width > 0 && state.height > 0 && !window.isVisible()) window.showInactive()
    }
    try {
      const bounds = await host.start(helper)
      host.on("bounds", sync)
      host.on("close-request", () => { void coordinator.stop().then(options.focusMain) })
      host.on("fault", () => { if (!disposing) void coordinator.stop("NATIVE_HOST_FAILED") })
      window = new BrowserWindow({ width: 1000, height: 800, title: "Curated",
        show: false, frame: false, transparent: true, backgroundColor: "#00000000", thickFrame: false,
        skipTaskbar: true, autoHideMenuBar: true, resizable: false, movable: false, maximizable: false,
        minimizable: false, fullscreenable: false, webPreferences: {
          preload: path.join(options.directory, "playback-preload.cjs"), sandbox: true,
          contextIsolation: true, nodeIntegration: false, backgroundThrottling: false,
        } })
      overlay = window
      const handle = window.getNativeWindowHandle()
      await host.command("attach", { handle: (handle.length === 8 ? handle.readBigUInt64LE() : BigInt(handle.readUInt32LE())).toString() })
      window.webContents.setWindowOpenHandler(() => ({ action: "deny" }))
      window.webContents.on("will-navigate", event => event.preventDefault())
      window.webContents.on("will-attach-webview", event => event.preventDefault())
      window.webContents.on("render-process-gone", () => { if (!disposing) void coordinator.stop("PLAYER_UI_FAILED") })
      window.webContents.session.setPermissionCheckHandler(() => false)
      window.webContents.session.setPermissionRequestHandler((_contents, _permission, callback) => callback(false))
      window.once("ready-to-show", () => { ready = true; sync(host.state ?? bounds) })
      window.on("close", event => { if (!disposing) { event.preventDefault(); void coordinator.stop().then(options.focusMain) } })
      window.on("will-move", event => event.preventDefault())
      window.on("will-resize", event => event.preventDefault())
      await window.loadURL(page)
      return {
        createPlayer: () => new NativeMpvPlayer(host.state!.handle),
        focus: async () => { await host.command("restore"); if (!window!.isDestroyed()) { window!.show(); window!.focus() } },
        action: action => host.command(action),
        fullscreen: () => host.state?.fullscreen === true,
        dispose: async () => {
          disposing = true
          if (overlay === window) overlay = undefined
          window?.destroy()
          await host.dispose()
        },
      }
    } catch (error) {
      disposing = true
      if (overlay === window) overlay = undefined
      window?.destroy()
      await host.dispose()
      throw error
    }
  }
  const assertMain = (event: Electron.IpcMainInvokeEvent) => {
    const current = options.current()
    if (!current || !isTrustedDesktopSender(event.sender.id, current.window.webContents.id, event.senderFrame?.url,
      event.senderFrame === event.sender.mainFrame, current.renderer)) throw new Error("UNTRUSTED_PLAYBACK_SENDER")
    return current
  }
  const assertController = (event: Electron.IpcMainInvokeEvent) => {
    if (overlay && event.sender === overlay.webContents && event.senderFrame === event.sender.mainFrame && event.senderFrame.url === page) return
    assertMain(event)
  }
  ipcMain.handle("curated:playback-capabilities", event => { assertMain(event); return capabilities() })
  ipcMain.handle("curated:playback-preferences", (event, input: unknown) => {
    assertMain(event)
    if (!input || typeof input !== "object") throw new Error("INVALID_PLAYBACK_PREFERENCES")
    // 偏好开关不能用设置页较早的快照覆盖播放中保存的音量/倍速。
    const value = validatePlaybackPreferences({ ...preferences, preferNative: (input as { preferNative?: unknown }).preferNative })
    store.write(value); preferences = value
    return capabilities()
  })
  ipcMain.handle("curated:playback-open", (event, input: unknown) => {
    const current = assertMain(event)
    if (!capabilities().available) throw new Error("NATIVE_PLAYER_UNAVAILABLE")
    const scoped = current.window.webContents.session
    const context: PlaybackContext = { origin: current.origin, generation: current.generation, fetch: async (url, init) => {
      if (new URL(url).origin !== new URL(current.origin).origin) throw new Error("INVALID_MEDIA_ORIGIN")
      return scoped.fetch(url, { ...init, credentials: "include", redirect: "error" })
    } }
    return coordinator.open(context, input)
  })
  ipcMain.handle("curated:playback-snapshot", event => { assertController(event); return coordinator.snapshot() })
  ipcMain.handle("curated:playback-command", async (event, sessionId: string, input: DesktopPlaybackCommand) => {
    assertController(event)
    await coordinator.command(sessionId, input)
    if (input.action === "close" || input.action === "stop") options.focusMain()
  })
  coordinator.on("snapshot", (snapshot: DesktopPlaybackSnapshot) => {
    const main = options.current()?.window
    for (const window of [main, overlay]) if (window && !window.isDestroyed()) window.webContents.send("curated:playback-state", snapshot)
  })
  coordinator.on("web-fallback", input => {
    const main = options.current()?.window
    if (main && !main.isDestroyed()) main.webContents.send("curated:playback-web", input)
    options.focusMain()
  })
  return { stop: (error?: string) => coordinator.stop(error), snapshot: () => coordinator.snapshot() }
}
