import { readFileSync } from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import vm from "node:vm"

import { describe, expect, it } from "vitest"



const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

describe("Electron preload bridge", () => {
  it("exposes narrow server-page controls without a local filesystem picker", async () => {
    const exposed: Record<string, unknown> = {}
    const invokedChannels: string[] = []
    const code = readFileSync(path.join(__dirname, "preload.cjs"), "utf8")

    vm.runInNewContext(code, {
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

    expect(Object.keys(exposed)).toEqual(["curatedDesktop"])

    const api = exposed.curatedDesktop as { getInfo: () => Promise<unknown>; changeServer: () => Promise<unknown>; serverPaths: boolean }
    expect(api.serverPaths).toBe(true)
    await api.getInfo()
    await api.changeServer()
    expect(invokedChannels).toEqual(["curated:desktop-info", "curated:change-server"])
  })
})
