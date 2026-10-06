import { createServer, type Server, type ServerResponse } from "node:http"
import { randomBytes } from "node:crypto"
import { Readable, Transform } from "node:stream"
import { pipeline } from "node:stream/promises"
import { EventEmitter } from "node:events"
import type { NativeNetworkDiagnostics } from "./native-player-contract.js"

export type MediaFetcher = (url: string, init?: RequestInit) => Promise<Response>

/** 仅代理一份已授权媒体；mpv 不接触 Server Cookie，响应按背压流式传输。 */
export class NativeMediaProxy extends EventEmitter {
  private server?: Server
  private readonly pending = new Set<AbortController>()
  private address?: string
  private timer?: ReturnType<typeof setInterval>
  private lastSample = 0
  private lastBytes = 0
  private network: NativeNetworkDiagnostics

  /** fetcher 必须绑定当前 Server session；上游媒体地址由主进程构造。 */
  constructor(private readonly source: string, private readonly fetcher: MediaFetcher) {
    super()
    const url = new URL(source)
    if (!["http:", "https:"].includes(url.protocol)) throw new Error("INVALID_MEDIA_ORIGIN")
    this.network = { sourceHost: url.host, protocol: url.protocol === "https:" ? "https" : "http", mime: null,
      bytesPerSec: 0, receivedBytes: 0, requests: 0, rangeRequests: 0, history: [] }
  }

  diagnostics(): NativeNetworkDiagnostics { return structuredClone(this.network) }

  /** 以单调时钟计算真实读取速率；闲置下一采样为零，历史有界。 */
  private sample(): void {
    const now = performance.now()
    const elapsed = now - this.lastSample
    this.network.bytesPerSec = elapsed > 0 ? (this.network.receivedBytes - this.lastBytes) * 1000 / elapsed : 0
    this.lastSample = now
    this.lastBytes = this.network.receivedBytes
    this.network.history = [...this.network.history.slice(-29), this.network.bytesPerSec]
    this.emit("diagnostics", this.diagnostics())
  }

  /** 在随机 loopback 端口创建不可预测的单媒体能力地址。 */
  async start(): Promise<string> {
    if (this.address) return this.address
    this.network = { ...this.network, mime: null, bytesPerSec: 0, receivedBytes: 0, requests: 0, rangeRequests: 0, history: [] }
    this.lastBytes = 0
    this.lastSample = performance.now()
    const route = `/${randomBytes(24).toString("hex")}/media`
    const server = createServer((request, response) => {
      // 路由及请求方法被限定，不能把它变成任意来源代理。
      if (request.url !== route || !["GET", "HEAD"].includes(request.method ?? "")) {
        response.writeHead(404).end()
        return
      }
      const abort = new AbortController()
      this.network.requests++
      if (request.headers.range) this.network.rangeRequests++
      this.pending.add(abort)
      const deadline = setTimeout(() => { // 限制无响应头的上游请求。
        abort.abort()
      }, 15000)
      response.on("close", () => { // mpv 放弃 seek 前的连接时取消上游下载。
        abort.abort()
      })
      void this.forward(request.method!, request.headers.range, response, abort, deadline)
    })
    this.server = server
    await new Promise<void>((resolve, reject) => {
      // 监听错误只在绑定阶段交给调用方。
      server.once("error", reject)
      server.listen(0, "127.0.0.1", () => {
        server.removeListener("error", reject)
        resolve()
      })
    })
    const address = server.address()
    if (!address || typeof address === "string") throw new Error("MEDIA_PROXY_START_FAILED")
    this.address = `http://127.0.0.1:${address.port}${route}`
    this.timer = setInterval(() => this.sample(), 1000)
    return this.address
  }

  /** 只转发媒体必要字段，拒绝重定向，响应体流量由下游消费控制。 */
  private async forward(method: string, range: string | undefined, output: ServerResponse,
    abort: AbortController, deadline: ReturnType<typeof setTimeout>): Promise<void> {
    try {
      const headers: Record<string, string> = {}
      if (range) headers.Range = range
      const upstream = await this.fetcher(this.source, { method, headers, redirect: "error", signal: abort.signal })
      clearTimeout(deadline)
      if (![200, 206, 416].includes(upstream.status)) {
        await upstream.body?.cancel()
        output.writeHead(upstream.status === 401 || upstream.status === 403 ? 403 : 502).end()
        return
      }
      const mime = upstream.headers.get("content-type")?.split(";")[0]?.trim().toLowerCase()
      if (mime && /^(?:video|audio|application)\/[a-z0-9.+-]+$/.test(mime)) this.network.mime = mime
      for (const name of ["content-type", "content-length", "content-range", "accept-ranges", "etag", "last-modified"]) {
        const value = upstream.headers.get(name)
        if (value) output.setHeader(name, value)
      }
      output.setHeader("Cache-Control", "no-store")
      output.writeHead(upstream.status)
      if (method === "HEAD" || !upstream.body) {
        await upstream.body?.cancel()
        output.end()
      } else {
        const counter = new Transform({ transform: (chunk: Buffer, _encoding, callback) => {
          this.network.receivedBytes += chunk.length
          callback(null, chunk)
        } })
        await pipeline(Readable.fromWeb(upstream.body as import("node:stream/web").ReadableStream), counter, output)
      }
    } catch {
      if (!output.headersSent && !output.destroyed) output.writeHead(502).end()
      else output.destroy()
    } finally {
      clearTimeout(deadline)
      this.pending.delete(abort)
    }
  }

  /** 停止本次代理以及所有下载，不影响系统上的其它媒体连接。 */
  async stop(): Promise<void> {
    clearInterval(this.timer); this.timer = undefined
    this.network.bytesPerSec = 0
    for (const request of this.pending) request.abort()
    this.pending.clear()
    const server = this.server
    this.server = undefined
    this.address = undefined
    if (!server) return
    await new Promise<void>((resolve) => {
      // 关闭包括正在读媒体的连接，避免退出等待整部影片。
      server.close(() => resolve())
      server.closeAllConnections()
    })
  }
}
