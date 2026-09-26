import { readFileSync } from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import vm from "node:vm"

import { describe, expect, it } from "vitest"

import { pickDirectoryChannel } from "./desktop-shell"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

describe("Electron preload bridge", () => {
  it.each(["darwin", "win32", "linux"])("exposes the directory picker and a read-only chrome capability on %s", async (platform) => {
    const exposed: Record<string, unknown> = {}
    const invokedChannels: string[] = []
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
            invoke: async (channel: string) => {
              invokedChannels.push(channel)
              return { path: "D:/Media" }
            },
          },
        }
      },
    })

    expect(Object.keys(exposed)).toEqual(["javLibrary"])

    const api = exposed.javLibrary as { windowChrome: string; openServerConnections: () => Promise<unknown>; pickDirectory: () => Promise<unknown>; getDesktopInfo: () => Promise<unknown>; checkDesktopUpdate: () => Promise<unknown> }
    expect(Object.keys(api).sort()).toEqual(["checkDesktopUpdate", "getDesktopInfo", "openServerConnections", "pickDirectory", "windowChrome"])
    expect(api.windowChrome).toBe(platform === "darwin" ? "macos" : "native")
    await expect(api.pickDirectory()).resolves.toEqual({ path: "D:/Media" })
    await api.getDesktopInfo()
    await api.checkDesktopUpdate()
    await api.openServerConnections()
    expect(invokedChannels).toEqual([pickDirectoryChannel, "curated:desktop-info", "curated:desktop-check-update", "curated:open-servers"])
  })
})
