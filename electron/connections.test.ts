import { mkdtempSync, rmSync } from "node:fs"
import os from "node:os"
import path from "node:path"
import { afterEach, describe, expect, it, vi } from "vitest"
import { ConnectionStore, connectionPartition, normalizeServerUrl, probeServer } from "./connections"

const id = "18a31111-36e0-424b-9f9f-d1e9cfb74ed2"
const info = { product: "curated-server", serverId: id, name: "NAS", version: "2", protocolVersion: 1, desktopBridgeVersion: 1 }
const dirs: string[] = []
afterEach(() => dirs.splice(0).forEach(dir => rmSync(dir, { recursive: true, force: true })))
describe("server connections", () => {
  it("accepts custom ports and IPv6 without silently dropping paths or credentials", () => {
    expect(normalizeServerUrl("192.168.1.2:8081")).toBe("http://192.168.1.2:8081")
    expect(normalizeServerUrl("https://[::1]:8443/")).toBe("https://[::1]:8443")
    for (const input of ["file:///tmp", "https://x/curated", "https://u:p@x", "https://x/?key=a"]) expect(() => normalizeServerUrl(input)).toThrow()
  })
  it("uses a bounded credential-free handshake and only falls back on 404", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify(info)))
    expect(await probeServer("http://nas:8081", undefined, fetcher)).toMatchObject(info)
    expect(fetcher).toHaveBeenCalledWith("http://nas:8081/api/server-info", expect.objectContaining({ credentials: "omit", redirect: "error" }))
    const locked = vi.fn().mockResolvedValue(new Response("", { status: 423 }))
    await expect(probeServer("http://nas", undefined, locked)).rejects.toThrow("423")
    expect(locked).toHaveBeenCalledTimes(1)
    const old = vi.fn().mockResolvedValueOnce(new Response("", { status: 404 })).mockResolvedValueOnce(new Response('{"name":"curated"}'))
    expect(await probeServer("http://nas", undefined, old)).toMatchObject({ legacy: true })
  })
  it("rejects incompatible, arbitrary and oversized replies", async () => {
    for (const payload of [{ ...info, protocolVersion: 2 }, {}, { ...info, name: "x".repeat(40000) }]) {
      await expect(probeServer("http://nas", undefined, vi.fn().mockResolvedValue(new Response(JSON.stringify(payload))))).rejects.toThrow()
    }
  })
  it("remembers and forgets endpoints; partitions differ even on the same host", () => {
    const dir = mkdtempSync(path.join(os.tmpdir(), "curated-connections-")); dirs.push(dir)
    const store = new ConnectionStore(dir)
    const a = { url: "http://nas:8081", serverId: id, name: "A" }
    const b = { ...a, url: "http://nas:8082" }
    store.save(a); store.save(b)
    expect(new ConnectionStore(dir).read().lastUrl).toBe(b.url)
    expect(connectionPartition(a)).not.toBe(connectionPartition(b))
    store.forget(b.url)
    expect(store.read()).toEqual({ version: 1, connections: [a] })
  })
})
