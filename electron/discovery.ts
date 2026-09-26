import dgram from "node:dgram"
import { networkInterfaces } from "node:os"
import { probeServer } from "./connections.js"

export const serviceType = "urn:curated:service:Library:1"
export interface DiscoveredServer { serverId: string; name: string; version: string; urls: string[]; expiresAt: number }
export interface DiscoveryReply { url: string; serverId: string; maxAge: number }

export function parseDiscoveryReply(message: string, source: string): DiscoveryReply | undefined {
  if (message.length > 2048 || !message.startsWith("HTTP/1.1 200 OK\r\n")) return
  const headers = new Map<string, string>()
  for (const line of message.split("\r\n").slice(1)) {
    if (!line) break
    const colon = line.indexOf(":")
    if (colon < 1) return
    const key = line.slice(0, colon).toLowerCase()
    if (headers.has(key)) return
    headers.set(key, line.slice(colon + 1).trim())
  }
  if (headers.get("st") !== serviceType) return
  const usn = headers.get("usn")?.match(/^uuid:([a-f\d-]{36})::urn:curated:service:Library:1$/i)
  const age = headers.get("cache-control")?.match(/^max-age=(\d+)$/i)
  if (!usn || !age || Number(age[1]) < 1) return
  try {
    const location = new URL(headers.get("location") ?? "")
    // Never follow an advertised URL to an unrelated host or arbitrary path.
    if (location.protocol !== "http:" || location.hostname !== source || location.username || location.password || location.search || location.hash || location.pathname !== "/discovery/description.xml") return
    return { url: location.origin, serverId: usn[1].toLowerCase(), maxAge: Math.min(120, Number(age[1])) }
  } catch { return }
}

// Fresh bounded scan; no permanent background multicast listener is required.
// The public server-info handshake validates candidates without transferring cookies.
export async function discoverServers(signal: AbortSignal, fetcher: typeof fetch = fetch): Promise<DiscoveredServer[]> {
  const sockets: dgram.Socket[] = []
  const results = new Map<string, DiscoveredServer>()
  const pending: Promise<void>[] = []
  const seen = new Set<string>()
  const scan = new AbortController()
  const combined = AbortSignal.any([signal, scan.signal])
  const search = Buffer.from(`M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: "ssdp:discover"\r\nMX: 2\r\nST: ${serviceType}\r\n\r\n`)
  const addresses = Object.values(networkInterfaces()).flatMap(items => items ?? []).filter(item => item.family === "IPv4" && !item.internal).slice(0, 16)
  try {
    for (const address of addresses) {
      if (combined.aborted) break
      const socket = dgram.createSocket("udp4")
      sockets.push(socket)
      socket.on("error", () => {}) // A restricted interface must not prevent manual connections.
      socket.on("message", (data, remote) => {
        if (combined.aborted || seen.size >= 32) return
        const reply = parseDiscoveryReply(data.toString("utf8"), remote.address)
        if (!reply || seen.has(reply.url)) return
        seen.add(reply.url)
        pending.push((async () => {
          try {
            const info = await probeServer(reply.url, combined, fetcher)
            if (info.serverId !== reply.serverId || combined.aborted) return
            const existing = results.get(info.serverId)
            if (existing) existing.urls.push(reply.url)
            else results.set(info.serverId, { serverId: info.serverId, name: info.name, version: info.version, urls: [reply.url], expiresAt: Date.now() + reply.maxAge * 1000 })
          } catch { /* An unverified candidate is not offered to the user. */ }
        })())
      })
      await new Promise<void>(resolve => {
        socket.once("error", () => resolve())
        socket.bind(0, address.address, () => {
          try { socket.setMulticastInterface(address.address); socket.setMulticastTTL(2); socket.send(search, 1900, "239.255.255.250", () => resolve()) }
          catch { resolve() }
        })
      })
    }
    await new Promise<void>(resolve => {
      if (combined.aborted) { resolve(); return }
      const finish = () => { clearTimeout(timer); combined.removeEventListener("abort", finish); resolve() }
      const timer = setTimeout(finish, 2800)
      combined.addEventListener("abort", finish, { once: true })
    })
    sockets.forEach(socket => { try { socket.close() } catch { /* Already closed. */ } })
    await Promise.allSettled(pending)
    return [...results.values()].filter(item => item.expiresAt > Date.now()).sort((a, b) => a.name.localeCompare(b.name))
  } finally {
    scan.abort()
    sockets.forEach(socket => { try { socket.close() } catch { /* Already closed. */ } })
  }
}
