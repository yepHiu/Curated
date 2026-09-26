import { createHash, randomUUID } from "node:crypto"
import { existsSync, readFileSync, mkdirSync, writeFileSync, renameSync } from "node:fs"
import path from "node:path"
import { isCuratedHealthPayload } from "./backend-process.js"

export interface SavedServer { id: string; name: string; url: string; serverId?: string; partition?: string }
export interface ServerConnections { schema: 1; servers: SavedServer[]; lastServerId?: string }

export function normalizeServerUrl(input: unknown): string {
  if (typeof input !== "string" || !input.trim() || input.length > 2048) throw new Error("请输入服务器地址。")
  let url: URL
  try { url = new URL(input.includes("://") ? input.trim() : `http://${input.trim()}`) }
  catch { throw new Error("服务器地址格式不正确。") }
  if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
    throw new Error("请使用 HTTP/HTTPS 根地址，不包含账号、路径、查询参数或片段。")
  }
  return url.origin
}

export function serverSessionPartition(url: string): string {
  return `persist:curated-server-${createHash("sha256").update(normalizeServerUrl(url)).digest("hex")}`
}

export class ServerConnectionStore {
  private state: ServerConnections = { schema: 1, servers: [] }
  constructor(private readonly file: string) {
    if (!existsSync(file)) {
      const previous = path.join(path.dirname(file), "connections.json")
      if (existsSync(previous)) {
        const old = JSON.parse(readFileSync(previous, "utf8"))
        if (old.version !== 1 || !Array.isArray(old.connections) || old.connections.length > 100) throw new Error("旧连接配置损坏，请保留文件后修复。")
        const seen = new Set<string>()
        const migrated: ServerConnections = { schema: 1, servers: [] }
        for (const entry of old.connections) {
          if (!entry || typeof entry.name !== "string" || !entry.name.trim() || entry.name.length > 80 || typeof entry.serverId !== "string" || !entry.serverId || entry.serverId.length > 2100) throw new Error("旧连接配置损坏。")
          const url = normalizeServerUrl(entry.url)
          if (seen.has(url)) throw new Error("旧连接配置损坏。")
          seen.add(url)
          const saved = { id: randomUUID(), name: entry.name.trim(), url, serverId: entry.serverId,
            partition: `persist:curated-server-${createHash("sha256").update(`${url}\n${entry.serverId}`).digest("hex")}` }
          migrated.servers.push(saved)
          if (old.lastUrl === url) migrated.lastServerId = saved.id
        }
        if (old.lastUrl !== undefined && !migrated.lastServerId) throw new Error("上次连接配置无效。")
        this.commit(migrated)
      }
      return
    }
    const data = JSON.parse(readFileSync(file, "utf8")) as ServerConnections
    if (data.schema !== 1 || !Array.isArray(data.servers) || data.servers.length > 100) throw new Error("服务器列表文件无效，请保留原文件并检查。")
    const ids = new Set<string>()
    const urls = new Set<string>()
    for (const item of data.servers) {
      if (!item || typeof item.id !== "string" || !item.id || typeof item.name !== "string" || !item.name.trim() || item.name.length > 80 || normalizeServerUrl(item.url) !== item.url || ids.has(item.id) || urls.has(item.url)) throw new Error("服务器列表文件无效，请保留原文件并检查。")
      if (item.serverId !== undefined && (typeof item.serverId !== "string" || !item.serverId || item.serverId.length > 2100)) throw new Error("服务器身份无效。")
      if (item.partition !== undefined && !/^persist:curated-server-[a-f0-9]{64}$/.test(item.partition)) throw new Error("服务器会话无效。")
      ids.add(item.id); urls.add(item.url)
    }
    if (data.lastServerId !== undefined && !ids.has(data.lastServerId)) throw new Error("上次连接记录无效。")
    this.state = data
  }
  snapshot(): ServerConnections { return structuredClone(this.state) }
  private commit(next: ServerConnections): void {
    mkdirSync(path.dirname(this.file), { recursive: true })
    writeFileSync(`${this.file}.tmp`, JSON.stringify(next, null, 2) + "\n", { mode: 0o600 })
    renameSync(`${this.file}.tmp`, this.file)
    this.state = next
  }
  add(input: unknown): SavedServer {
    if (!input || typeof input !== "object" || "id" in input) throw new Error("服务器信息无效。")
    const value = input as Partial<SavedServer>
    const url = normalizeServerUrl(value.url)
    if (this.state.servers.some(server => server.url === url)) throw new Error("该地址已保存在列表中。")
    return this.save({ name: value.name, url })
  }
  save(input: unknown): SavedServer {
    if (!input || typeof input !== "object") throw new Error("服务器信息无效。")
    const value = input as Partial<SavedServer>
    const url = normalizeServerUrl(value.url)
    if (typeof value.name !== "string" || !value.name.trim() || value.name.trim().length > 80) throw new Error("名称需为 1–80 个字符。")
    const next = this.snapshot()
    const existing = value.id === undefined ? next.servers.find(s => s.url === url) : next.servers.find(s => s.id === value.id)
    if (value.id !== undefined && !existing) throw new Error("服务器记录不存在。")
    if (next.servers.some(s => s.url === url && s.id !== existing?.id)) throw new Error("该地址已保存在列表中。")
    if (!existing && next.servers.length >= 100) throw new Error("最多保存 100 台服务器。")
    const saved = { ...(existing?.url === url ? existing : {}), id: existing?.id ?? randomUUID(), name: value.name.trim(), url }
    next.servers = existing ? next.servers.map(s => s.id === saved.id ? saved : s) : [...next.servers, saved]
    this.commit(next)
    return saved
  }
  bindIdentity(id: string, serverId: string, partition: string): void {
    const next = this.snapshot()
    const target = next.servers.find(server => server.id === id)
    if (!target) throw new Error("服务器记录不存在。")
    target.serverId = serverId
    target.partition = partition
    this.commit(next)
  }
  remove(id: string): void {
    const next = this.snapshot()
    next.servers = next.servers.filter(s => s.id !== id)
    if (next.lastServerId === id) delete next.lastServerId
    this.commit(next)
  }
  remember(id: string): void {
    const next = this.snapshot()
    if (!next.servers.some(s => s.id === id)) throw new Error("服务器记录不存在。")
    next.lastServerId = id
    this.commit(next)
  }
}

export async function probeServer(url: string, fetcher: typeof fetch = fetch): Promise<void> {
  const endpoint = `${normalizeServerUrl(url)}/api/health`
  let response: Response
  try {
    response = await fetcher(endpoint, {
      signal: AbortSignal.timeout(8000), redirect: "error", credentials: "omit", cache: "no-store",
    })
  } catch {
    throw new Error("无法连接服务器，请检查地址、端口、证书及网络，然后重试。")
  }
  if (!response.ok || !response.body) throw new Error("无法连接服务器，请检查地址、端口及网络。")
  const reader = response.body.getReader()
  const chunks: Uint8Array[] = []
  let size = 0
  try {
    while (true) {
      const chunk = await reader.read()
      if (chunk.done) break
      size += chunk.value.length
      if (size > 64 * 1024) throw new Error("服务器响应过大。")
      chunks.push(chunk.value)
    }
  } finally { await reader.cancel() }
  let payload: unknown
  try { payload = JSON.parse(Buffer.concat(chunks).toString("utf8")) }
  catch { throw new Error("该地址未返回有效的 Curated 服务信息。") }
  if (!isCuratedHealthPayload(payload)) throw new Error("该地址不是 Curated Server。")
}
