import { createServer, type Server } from "node:http"
import { describe, expect, it, vi } from "vitest"
import { NativeMediaProxy } from "./native-media-proxy"

/** 在临时端口提供可验证的受保护媒体，绝不访问用户 Server。 */
async function fixture(handler: Parameters<typeof createServer>[0]): Promise<{ server: Server; origin: string }> {
  const server = createServer(handler)
  await new Promise<void>((resolve) => { server.listen(0, "127.0.0.1", resolve) })
  const address = server.address()
  if (!address || typeof address === "string") throw new Error("fixture bind failed")
  return { server, origin: `http://127.0.0.1:${address.port}` }
}

/** 关闭 fixture，包括异常测试遗留的连接。 */
async function close(server: Server): Promise<void> {
  await new Promise<void>((resolve) => { server.close(() => resolve()); server.closeAllConnections() })
}

describe("native media proxy", () => {
  it("counts actual media bytes including repeated ranges, excludes HEAD and resets on restart", async () => {
    const proxy = new NativeMediaProxy("https://user:password@server.test:8443/private?token=secret", async (_url, init) =>
      new Response(init?.method === "HEAD" ? null : "data", { status: 206, headers: { "Content-Type": "video/mp4; codecs=avc1" } }))
    try {
      const url = await proxy.start()
      await fetch(url, { method: "HEAD" })
      for (let i = 0; i < 2; i++) await (await fetch(url, { headers: { Range: "bytes=0-3" } })).arrayBuffer()
      expect(proxy.diagnostics()).toMatchObject({ sourceHost: "server.test:8443", protocol: "https", mime: "video/mp4", receivedBytes: 8, requests: 3, rangeRequests: 2 })
      expect(JSON.stringify(proxy.diagnostics())).not.toMatch(/password|private|secret|token/)
      await proxy.stop()
      await proxy.start()
      expect(proxy.diagnostics()).toMatchObject({ receivedBytes: 0, requests: 0, rangeRequests: 0, mime: null, history: [] })
    } finally { await proxy.stop() }
  })
  it("samples one-second rates, reaches zero when idle, bounds history and stops emitting", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "performance"] })
    const proxy = new NativeMediaProxy("http://server.test/media", async () => new Response("1234"))
    const events = vi.fn()
    proxy.on("diagnostics", events)
    try {
      const url = await proxy.start()
      await (await fetch(url)).arrayBuffer()
      vi.advanceTimersByTime(1000)
      expect(proxy.diagnostics().bytesPerSec).toBe(4)
      const snapshot = proxy.diagnostics()
      snapshot.history[0] = 999
      vi.advanceTimersByTime(31000)
      expect(proxy.diagnostics()).toMatchObject({ bytesPerSec: 0, receivedBytes: 4, history: Array(30).fill(0) })
      await proxy.stop()
      const count = events.mock.calls.length
      vi.advanceTimersByTime(5000)
      expect(events).toHaveBeenCalledTimes(count)
    } finally { await proxy.stop(); vi.useRealTimers() }
  })
  // Range 数据、认证和能力边界必须在真实 HTTP 传输中保留。
  it("streams exact ranges without exposing Server cookies and rejects other routes", async () => {
    let requestedPath = ""
    const upstream = await fixture((request, response) => {
      // fixture 要求认证，响应故意带 Set-Cookie 检查代理不泄露它。
      requestedPath = request.url ?? ""
      if (request.headers.cookie !== "fixture-session=secret") { response.writeHead(403).end(); return }
      expect(request.headers.range).toBe("bytes=2-5")
      response.writeHead(206, { "Content-Range": "bytes 2-5/10", "Content-Length": "4", "Set-Cookie": "must-not-leak=secret" })
      response.end("2345")
    })
    const proxy = new NativeMediaProxy(`${upstream.origin}/stream?fileId=part-2`, async (url, init) => {
      // 模拟 Electron session 的认证，仅在上游请求内补充。
      return await fetch(url, { ...init, headers: { ...init?.headers, Cookie: "fixture-session=secret" } })
    })
    try {
      const url = await proxy.start()
      const response = await fetch(url, { headers: { Range: "bytes=2-5" } })
      expect(response.status).toBe(206)
      expect(response.headers.get("content-range")).toBe("bytes 2-5/10")
      expect(response.headers.get("set-cookie")).toBeNull()
      expect(await response.text()).toBe("2345")
      expect(requestedPath).toBe("/stream?fileId=part-2")
      expect((await fetch(new URL("/wrong/media", url))).status).toBe(404)
      expect((await fetch(url, { method: "POST" })).status).toBe(404)
      await proxy.stop()
      await expect(fetch(url)).rejects.toThrow()
    } finally { await proxy.stop(); await close(upstream.server) }
  })

  // 重定向绝不能把认证带到第二个来源。
  it("rejects redirects before the destination is requested", async () => {
    let destinationRequests = 0
    const destination = await fixture((_request, response) => { destinationRequests++; response.end("bad") })
    const upstream = await fixture((_request, response) => { response.writeHead(302, { Location: destination.origin }).end() })
    const proxy = new NativeMediaProxy(upstream.origin, fetch)
    try {
      expect((await fetch(await proxy.start())).status).toBe(502)
      expect(destinationRequests).toBe(0)
    } finally { await proxy.stop(); await close(upstream.server); await close(destination.server) }
  })

  // 停止播放器必须取消正在传输的影片。
  it("aborts an active upstream stream when the proxy stops", async () => {
    let aborted = false
    const proxy = new NativeMediaProxy("http://server.invalid/stream", async (_url, init) => {
      // 不结束的 body 用来验证回收，不需要下载大文件。
      const stream = new ReadableStream<Uint8Array>({
        /** 先发送一个字节，使客户端完成响应头读取。 */
        start(controller) {
          controller.enqueue(new Uint8Array([1]))
          init?.signal?.addEventListener("abort", () => { aborted = true; controller.error(new Error("cancelled")) })
        },
      })
      return new Response(stream)
    })
    const response = await fetch(await proxy.start())
    await response.body!.getReader().read()
    await proxy.stop()
    expect(aborted).toBe(true)
  })
})
