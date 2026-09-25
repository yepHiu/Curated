import { createHash } from "node:crypto"
import { mkdirSync, readFileSync, renameSync, writeFileSync } from "node:fs"
import path from "node:path"
import { networkInterfaces } from "node:os"

export interface ServerInfo {
  product: "curated-server"
  serverId: string
  name: string
  version: string
  protocolVersion: number
  desktopBridgeVersion: number
  legacy?: boolean
}
export interface SavedConnection { url: string; serverId: string; name: string }
export interface ConnectionSettings { version: 1; lastUrl?: string; connections: SavedConnection[] }

export function normalizeServerUrl(input: string): string {
  const raw = input.trim()
  if (!raw || raw.length > 2048) throw new Error("请输入服务器地址。")
  const url = new URL(raw.includes("://") ? raw : `http://${raw}`)
  if (!["http:", "https:"].includes(url.protocol) || !url.hostname || url.username || url.password) {
    throw new Error("仅支持不含账号密码的 HTTP 或 HTTPS 地址。")
  }
  if (url.pathname !== "/" || url.search || url.hash) throw new Error("请填写服务器根地址，暂不支持子路径、查询参数或片段。")
  // No implicit localhost rewrite: the chosen endpoint is also a session boundary.
  return url.origin
}

export function connectionPartition(connection: SavedConnection): string {
  return `persist:curated-server-${createHash("sha256").update(`${connection.url}\n${connection.serverId}`).digest("hex")}`
}

async function readSmallJSON(response: Response): Promise<unknown> {
  if (!response.body) throw new Error("服务器返回了空响应。")
  const reader = response.body.getReader()
  const chunks: Uint8Array[] = []
  let size = 0
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      size += value.byteLength
      if (size > 32768) throw new Error("服务器身份响应过大。")
      chunks.push(value)
    }
    return JSON.parse(Buffer.concat(chunks).toString("utf8")) as unknown
  } finally { await reader.cancel().catch(() => {}) }
}

export async function probeServer(input: string, signal?: AbortSignal, fetchImpl = fetch): Promise<ServerInfo> {
  const url = normalizeServerUrl(input)
  const combined = AbortSignal.any([AbortSignal.timeout(5000), ...(signal ? [signal] : [])])
  const request = (endpoint: string) => fetchImpl(url + endpoint, {
    signal: combined, redirect: "error", credentials: "omit", headers: { Accept: "application/json" },
  })
  const response = await request("/api/server-info")
  if (response.status === 404) {
    const health = await request("/api/health")
    if (!health.ok) throw new Error("服务器不可用。")
    const payload = await readSmallJSON(health)
    if (!payload || typeof payload !== "object" || !("name" in payload) || !["curated", "curated-dev"].includes(String(payload.name))) {
      throw new Error("此地址不是 Curated 服务器。")
    }
    return { product: "curated-server", serverId: `legacy:${url}`, name: "Curated（旧版）", version: "", protocolVersion: 0, desktopBridgeVersion: 0, legacy: true }
  }
  if (!response.ok) throw new Error(`连接检查失败（HTTP ${response.status}）。`)
  const payload = await readSmallJSON(response)
  if (!payload || typeof payload !== "object") throw new Error("服务器身份无效。")
  const info = payload as Record<string, unknown>
  if (info.product !== "curated-server" || typeof info.serverId !== "string" || !/^[a-f\d]{8}(?:-[a-f\d]{4}){3}-[a-f\d]{12}$/i.test(info.serverId) || typeof info.name !== "string" || info.name.length > 256 || typeof info.version !== "string") {
    throw new Error("此地址未提供有效的 Curated 服务器身份。")
  }
  if (info.protocolVersion !== 1 || info.desktopBridgeVersion !== 1) throw new Error("此服务器需要其他版本的 Curated Desktop，请更新客户端。")
  return { product: "curated-server", serverId: info.serverId, name: info.name, version: info.version, protocolVersion: 1, desktopBridgeVersion: 1 }
}

export class ConnectionStore {
  private readonly file: string
  constructor(directory: string) { this.file = path.join(directory, "connections.json") }
  read(): ConnectionSettings {
    let content: string
    try { content = readFileSync(this.file, "utf8") } catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") return { version: 1, connections: [] }
      throw error
    }
    const value = JSON.parse(content) as ConnectionSettings
    if (value.version !== 1 || !Array.isArray(value.connections) || value.connections.length > 100) throw new Error("连接配置损坏，请保留文件后修复。")
    for (const item of value.connections) {
      if (typeof item.url !== "string" || normalizeServerUrl(item.url) !== item.url || typeof item.serverId !== "string" || typeof item.name !== "string") throw new Error("连接配置损坏。")
    }
    if (value.lastUrl && !value.connections.some(item => item.url === value.lastUrl)) throw new Error("上次连接配置无效。")
    return value
  }
  save(connection: SavedConnection): void {
    const settings = this.read()
    settings.connections = [connection, ...settings.connections.filter(item => item.url !== connection.url)].slice(0, 20)
    settings.lastUrl = connection.url
    this.write(settings)
  }
  forget(url: string): void {
    const settings = this.read()
    settings.connections = settings.connections.filter(item => item.url !== url)
    if (settings.lastUrl === url) delete settings.lastUrl
    this.write(settings)
  }
  private write(settings: ConnectionSettings): void {
    mkdirSync(path.dirname(this.file), { recursive: true })
    writeFileSync(`${this.file}.tmp`, JSON.stringify(settings, null, 2), { mode: 0o600 })
    renameSync(`${this.file}.tmp`, this.file)
  }
}

export function localServerSuggestion(localAppData = process.env.LOCALAPPDATA): string | undefined {
  if (!localAppData) return
  try {
    const hint = JSON.parse(readFileSync(path.join(localAppData, "Curated", "server-connection.json"), "utf8")) as { url?: unknown }
    if (typeof hint.url !== "string") return
    const normalized = normalizeServerUrl(hint.url)
    const host = new URL(normalized).hostname.replace(/^\[|\]$/g, "")
    if (["127.0.0.1", "::1"].includes(host) || Object.values(networkInterfaces()).some(items => items?.some(item => item.address === host))) return normalized
  } catch { /* No installed Server hint is a normal client-only startup. */ }
}
