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
  it("uses a bounded credential-free handshake and falls back for old Servers, including PIN-locked versions", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify(info)))
    expect(await probeServer("http://nas:8081", undefined, fetcher)).toMatchObject(info)
    expect(fetcher).toHaveBeenCalledWith("http://nas:8081/api/server-info", expect.objectContaining({ credentials: "omit", redirect: "error" }))
    const locked = vi.fn().mockResolvedValue(new Response("", { status: 423 }))
    await expect(probeServer("http://nas", undefined, locked)).rejects.toThrow("服务器不可用")
    expect(locked).toHaveBeenCalledTimes(2)
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

it("ignores stale local hints pointing to another machine", async () => {
  const { writeFileSync, mkdirSync } = await import("node:fs")
  const { localServerSuggestion } = await import("./connections")
  const dir = mkdtempSync(path.join(os.tmpdir(), "curated-hint-")); dirs.push(dir)
  mkdirSync(path.join(dir,"Curated"))
  const file = path.join(dir,"Curated","server-connection.json")
  writeFileSync(file,JSON.stringify({url:"http://127.0.0.1:9001"}))
  expect(localServerSuggestion(dir)).toBe("http://127.0.0.1:9001")
  writeFileSync(file,JSON.stringify({url:"https://example.com"}))
  expect(localServerSuggestion(dir)).toBeUndefined()
})


it("connects to a PIN-locked 1.6 Server while rejecting a locked new identity endpoint", async () => {
  for (const version of ["1.5.8", "1.6.0", "1.7.0"]) {
    const fetcher = vi.fn().mockResolvedValueOnce(new Response("", { status: 423 })).mockResolvedValueOnce(new Response(JSON.stringify({ name: "curated", version })))
    const result = probeServer("http://nas", undefined, fetcher)
    if (version === "1.7.0") await expect(result).rejects.toThrow()
    else expect(await result).toMatchObject({ legacy: true })
  }
})
it("keeps first identity binding compatible but detects a replaced known Server", async () => {
  const { hasServerIdentityChanged } = await import("./connections")
  expect(hasServerIdentityChanged(undefined, id, "http://nas")).toBe(false)
  expect(hasServerIdentityChanged("legacy:http://nas", id, "http://nas")).toBe(false)
  expect(hasServerIdentityChanged(id, id, "http://nas")).toBe(false)
  expect(hasServerIdentityChanged(id, "new-id", "http://nas")).toBe(true)
})
