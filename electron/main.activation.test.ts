import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

// 隔离主进程状态，使用真实事件注册验证 Dock 激活入口。
const runtime = vi.hoisted(() => ({
  ready: true,
  savedServer: true,
}))

// 只替换构建元数据，不读取或修改用户配置。
vi.mock("node:fs", async (importOriginal) => {
  const original = await importOriginal<typeof import("node:fs")>()
  return {
    ...original,
    // 图标使用模拟资源，不依赖本机文件。
    existsSync: (file: Parameters<typeof original.existsSync>[0]) => String(file).endsWith(".png"),
    // 主入口需要合法的发行元数据才能注册应用事件。
    readFileSync: (...args: Parameters<typeof original.readFileSync>) => String(args[0]).endsWith("desktop-release.json")
      ? JSON.stringify({ schema: 1, version: "0.2.2", buildStamp: "20261002.020000", distribution: "desktop", updateFeed: null })
      : original.readFileSync(...args),
  }
})

// 连接流程使用固定服务器身份，不访问真实网络。
vi.mock("./connections.js", () => ({
  // 始终返回与保存记录一致的身份。
  probeServer: async () => ({ serverId: "test-server", name: "Test" }),
  // 当前测试不涉及身份变更。
  hasServerIdentityChanged: () => false,
}))

// renderer 地址直接使用测试服务器，避免启动 Vite。
vi.mock("./frontend-process.js", () => ({
  // 模拟服务器自带前端的地址解析。
  resolveRendererBaseUrl: async ({ backendBaseUrl }: { backendBaseUrl: string }) => backendBaseUrl,
}))

// 保留连接地址及 partition 的真实工具，仅替换配置持久化。
vi.mock("./server-connections.js", async (importOriginal) => {
  const original = await importOriginal<typeof import("./server-connections.js")>()
  return {
    ...original,
    ServerConnectionStore: class {
      /** 根据场景提供上次连接记录，无磁盘副作用。 */
      snapshot() {
        return { servers: runtime.savedServer ? [{ id: "test", name: "Test", url: "http://test.local:8081" }] : [], lastServerId: "test" }
      }
      /** 身份绑定在此测试中仅记录调用。 */
      bindIdentity = vi.fn()
      /** 恢复窗口无需再次写入连接记录。 */
      remember = vi.fn()
    },
  }
})

// 最小 Electron 模型：隐藏窗口仍存在，销毁后才触发 closed。
vi.mock("electron", async () => {
  const { EventEmitter } = await import("node:events")
  const app = Object.assign(new EventEmitter(), {
    // 模拟应用就绪与单实例启动。
    isReady: () => runtime.ready,
    // 就绪 Promise 让主入口执行真实启动流程。
    whenReady: () => Promise.resolve(),
    // 测试是唯一实例。
    requestSingleInstanceLock: () => true,
    // 返回不会写入的配置目录。
    getPath: () => "/tmp/curated-activation-test",
    // 提供模拟应用安装路径。
    getAppPath: () => "/tmp/curated-activation-test",
    // 提供 renderer 入口版本参数。
    getVersion: () => "0.2.2",
    setName: vi.fn(), setPath: vi.fn(), setAppUserModelId: vi.fn(), quit: vi.fn(),
    isPackaged: false,
  })
  const testSession = {
    setProxy: vi.fn().mockResolvedValue(undefined),
    setPermissionCheckHandler: vi.fn(), setPermissionRequestHandler: vi.fn(),
    webRequest: { onBeforeSendHeaders: vi.fn() },
  }
  class TestWindow extends EventEmitter {
    static windows: TestWindow[] = []
    /** 已销毁窗口不再计入 Electron 窗口列表。 */
    static getAllWindows() { return this.windows.filter(/* 仅保留存活实例。 */ window => !window.destroyed) }
    destroyed = false
    visible = false
    minimized = false
    url = ""
    webContents = Object.assign(new EventEmitter(), {
      id: TestWindow.windows.length + 1,
      session: testSession,
      // 返回本实例已加载的地址，供真实连接流程核验。
      getURL: () => this.url,
      setWindowOpenHandler: vi.fn(), send: vi.fn(),
    })
    /** 登记实例，模拟 Electron 隐藏窗口仍计数的行为。 */
    constructor() { super(); TestWindow.windows.push(this) }
    /** 模拟加载，不触发网络请求。 */
    async loadURL(url: string) { this.url = url }
    /** 本地管理页不需要读取真实资源。 */
    loadFile = vi.fn().mockResolvedValue(undefined)
    /** 查询窗口存活状态。 */
    isDestroyed() { return this.destroyed }
    /** 查询窗口最小化状态。 */
    isMinimized() { return this.minimized }
    /** 关闭业务窗口时由真实 close 监听器决定隐藏或销毁。 */
    close() {
      const event = { defaultPrevented: false, /** 模拟 Electron 阻止窗口关闭。 */ preventDefault() { this.defaultPrevented = true } }
      this.emit("close", event)
      if (!event.defaultPrevented) this.destroy()
    }
    /** 隐藏不销毁 renderer。 */
    hide() { this.visible = false }
    /** 恢复最小化状态。 */
    restore = vi.fn(() => { this.minimized = false })
    /** 显示已有窗口。 */
    show = vi.fn(() => { this.visible = true })
    focus = vi.fn()
    /** 销毁后通知真实主进程清理窗口引用。 */
    destroy() { this.destroyed = true; this.emit("closed") }
  }
  class TestTray extends EventEmitter {
    setToolTip = vi.fn()
    setContextMenu = vi.fn()
  }
  const icon = {
    // 内存图标足够触发真实托盘创建分支。
    isEmpty: () => false,
    // Retina 托盘缩放不需要真实栅格数据。
    resize: () => icon,
  }
  return {
    app, BrowserWindow: TestWindow,
    Tray: TestTray,
    nativeImage: { /** 返回不会访问磁盘的图标。 */ createFromPath: () => icon },
    ipcMain: { handle: vi.fn() },
    session: { /** 启动只使用内存会话。 */ fromPartition: () => testSession },
    Menu: { setApplicationMenu: vi.fn(), buildFromTemplate: vi.fn() },
  }
})

// 每例重新加载主入口，让窗口及退出状态互不干扰。
beforeEach(async () => {
  const { app, BrowserWindow } = await import("electron")
  // 清理 mock 的窗口及事件监听器，避免不同主入口实例互相响应。
  for (const window of BrowserWindow.getAllWindows()) window.destroy()
  app.removeAllListeners()
  vi.clearAllMocks()
  vi.resetModules()
  runtime.ready = true
  runtime.savedServer = true
})
// 环境变量不会泄漏到其他 Electron 测试。
afterEach(() => { vi.unstubAllEnvs() })

/** 启动真实主入口并等待连接页或业务窗口准备完毕。 */
async function startDesktop(savedServer = true) {
  runtime.savedServer = savedServer
  vi.stubEnv("CURATED_ELECTRON_BACKEND_URL", "")
  vi.stubEnv("CURATED_BACKEND_URL", "")
  await import("./main.js")
  const { app, BrowserWindow } = await import("electron")
  // 连接完成后只应剩下一个业务窗口；未连接时则是管理窗口。
  await vi.waitFor(() => {
    expect(BrowserWindow.getAllWindows()).toHaveLength(1)
    if (savedServer) expect(BrowserWindow.getAllWindows()[0]!.webContents.getURL()).toContain("test.local")
  })
  return { app, BrowserWindow }
}

// Dock 激活测试覆盖隐藏、最小化、未连接以及退出边界。
describe("Desktop Dock activation", () => {
  // 重现关闭后隐藏窗口仍计数、旧 activate 不响应的缺陷，并覆盖最小化。
  it.each(["hidden", "minimized"])("restores the %s main window without reloading or creating another window", async (state) => {
    const { app, BrowserWindow } = await startDesktop()
    const window = BrowserWindow.getAllWindows()[0]!
    const load = vi.spyOn(window, "loadURL")
    window.close()
    if (state === "minimized") {
      // 最小化与隐藏都保留同一个 renderer。
      const testWindow = window as unknown as { minimized: boolean }
      testWindow.minimized = true
    }
    expect(window.isDestroyed()).toBe(false)
    expect(BrowserWindow.getAllWindows()).toHaveLength(1)
    vi.mocked(window.show).mockClear()
    app.emit("activate")
    expect(window.show).toHaveBeenCalledOnce()
    expect(window.focus).toHaveBeenCalled()
    if (state === "minimized") expect(window.restore).toHaveBeenCalledOnce()
    expect(load).not.toHaveBeenCalled()
    expect(BrowserWindow.getAllWindows()).toEqual([window])
  })

  // 管理窗口也必须能够从最小化恢复，关闭后可重新创建。
  it("restores the minimized connection manager and recreates it after closing", async () => {
    const { app, BrowserWindow } = await startDesktop(false)
    const window = BrowserWindow.getAllWindows()[0]! as unknown as { minimized: boolean; restore: ReturnType<typeof vi.fn>; close: () => void }
    window.minimized = true
    app.emit("activate")
    expect(window.restore).toHaveBeenCalledOnce()
    window.close()
    app.emit("activate")
    expect(BrowserWindow.getAllWindows()).toHaveLength(1)
    expect(BrowserWindow.getAllWindows()[0]).not.toBe(window)
  })

  // 退出期间及尚未就绪时的激活不得拉起窗口。
  it("ignores activation before readiness and during quit", async () => {
    const { app, BrowserWindow } = await startDesktop(false)
    const window = BrowserWindow.getAllWindows()[0]!
    vi.mocked(window.show).mockClear()
    runtime.ready = false
    app.emit("activate")
    runtime.ready = true
    app.emit("before-quit")
    app.emit("activate")
    expect(window.show).not.toHaveBeenCalled()
  })
})
