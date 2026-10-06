import { readFileSync } from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import vm from "node:vm"

import { describe, expect, it } from "vitest"

import { pickDirectoryChannel } from "./desktop-shell"
import type { DesktopPlaybackBridge } from "./playback-contract"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

describe("Electron preload bridge", () => {
  it.each(["darwin", "win32", "linux"])("exposes the directory picker and a read-only chrome capability on %s", async (platform) => {
    const exposed: Record<string, unknown> = {}
    const invokedChannels: string[] = []
    const invokedArgs: unknown[][] = []
    const listeners = new Map<string, (...args: unknown[]) => void>()
    const code = readFileSync(path.join(__dirname, "preload.cjs"), "utf8")

    vm.runInNewContext(code, {
      process: { platform },
      require: (moduleName: string) => {
        if (moduleName !== "electron") {
          throw new Error(`Unexpected preload require: ${moduleName}`)
        }
        return {
          contextBridge: {
            exposeInMainWorld: (name: string, api: unknown) => {
              exposed[name] = api
            },
          },
          ipcRenderer: {
            on: (channel: string, listener: (...args: unknown[]) => void) => { listeners.set(channel, listener) },
            removeListener: (channel: string, listener: (...args: unknown[]) => void) => { if (listeners.get(channel) === listener) listeners.delete(channel) },
            invoke: async (channel: string, ...args: unknown[]) => {
              invokedArgs.push(args)
              invokedChannels.push(channel)
              return { path: "D:/Media" }
            },
          },
        }
      },
    })

    expect(Object.keys(exposed)).toEqual(["javLibrary"])

    const api = exposed.javLibrary as { playback: DesktopPlaybackBridge; addServer: (input: { name: string; url: string }) => Promise<unknown>; windowChrome: string; getServerConnections: () => Promise<unknown>; openServerConnections: (serverId?: string) => Promise<unknown>; pickDirectory: () => Promise<unknown>; getDesktopInfo: () => Promise<unknown>; checkDesktopUpdate: () => Promise<unknown> }
    expect(Object.keys(api).sort()).toEqual(["addServer", "checkDesktopUpdate", "getDesktopInfo", "getServerConnections", "openServerConnections", "pickDirectory", "playback", "windowChrome"])
    expect(api.windowChrome).toBe(platform === "darwin" ? "macos" : "native")
    await expect(api.pickDirectory()).resolves.toEqual({ path: "D:/Media" })
    await api.getDesktopInfo()
    await api.checkDesktopUpdate()
    await api.addServer({ name: "NAS", url: "http://nas.local:8081" })
    await api.getServerConnections()
    await api.openServerConnections("saved-server-id")
    expect(invokedArgs.at(-1)).toEqual(["saved-server-id"])
    expect(invokedChannels).toEqual([pickDirectoryChannel, "curated:desktop-info", "curated:desktop-check-update", "curated:add-server", "curated:server-connections", "curated:open-servers"])
    const received: unknown[] = []
    const unsubscribe = api.playback.subscribe(value => { received.push(value) })
    listeners.get("curated:playback-state")?.({ sensitiveIpcEvent: true }, { sessionId: "current" })
    expect(received).toEqual([{ sessionId: "current" }])
    unsubscribe()
    expect(listeners.size).toBe(0)
    await api.playback.command("current", { action: "pause" })
    expect(invokedArgs.at(-1)).toEqual(["current", { action: "pause" }])
  })
})
