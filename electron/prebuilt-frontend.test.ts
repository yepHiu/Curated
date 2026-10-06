import { mkdtemp, mkdir, rm, writeFile } from "node:fs/promises"
import { createServer, type Server } from "node:http"
import os from "node:os"
import path from "node:path"
import { afterEach, describe, expect, it } from "vitest"
import { shouldUsePrebuiltFrontend, startPrebuiltFrontend, type PrebuiltFrontend } from "./prebuilt-frontend"

let frontend: PrebuiltFrontend | undefined
let directory: string | undefined
let backend: Server | undefined

// 每例回收真实 HTTP 监听和独立临时文件，不访问用户资料库。
afterEach(async () => {
  await frontend?.stop()
  frontend = undefined
  if (backend) await new Promise<void>(resolve => {
    // 上游流也必须退出，避免留下监听影响后续测试。
    backend!.close(() => { resolve() })
    backend!.closeAllConnections()
  })
  backend = undefined
  if (directory) await rm(directory, { recursive: true, force: true })
  directory = undefined
})

/** 使用真实临时构建目录启动轻量托管器，端口由系统分配。 */
async function start(backendBaseUrl = "http://127.0.0.1:8080") {
  directory = await mkdtemp(path.join(os.tmpdir(), "curated-prebuilt-"))
  await mkdir(path.join(directory, "assets"))
  await writeFile(path.join(directory, "index.html"), '<div id="app"></div><script src="/assets/app-hash.js"></script>')
  await writeFile(path.join(directory, "assets", "app-hash.js"), "window.compiled = true")
  frontend = await startPrebuiltFrontend({ directory, backendBaseUrl, baseUrl: "http://127.0.0.1:0" })
  return frontend.baseUrl
}

// 覆盖真实静态响应、页面路由、代理认证与失败隔离，而非只比对启动参数。
describe("prebuilt Desktop test frontend", () => {
  // 正式包和远端 Server 始终保留自己的页面来源。
  it("only enables explicitly configured local development tests", () => {
    const env = { CURATED_ELECTRON_FRONTEND_DIR: "dist" }
    expect(shouldUsePrebuiltFrontend({ backendBaseUrl: "http://127.0.0.1:8080", isPackaged: false, env })).toBe(true)
    expect(shouldUsePrebuiltFrontend({ backendBaseUrl: "http://127.0.0.1:8080", isPackaged: true, env })).toBe(false)
    expect(shouldUsePrebuiltFrontend({ backendBaseUrl: "http://remote:8080", isPackaged: false, env })).toBe(false)
    expect(shouldUsePrebuiltFrontend({ backendBaseUrl: "http://127.0.0.1:8080", isPackaged: false, env: {} })).toBe(false)
  })

  // 深链接直接刷新仍返回 SPA；旧资源必须得到 404，不能返回 HTML。
  it("serves compiled assets and SPA routes with correct cache and content types", async () => {
    const baseUrl = await start()
    const page = await fetch(`${baseUrl}/movies/123`)
    expect(page.status).toBe(200)
    expect(page.headers.get("cache-control")).toBe("no-store")
    expect(await page.text()).toContain("app-hash.js")
    const script = await fetch(`${baseUrl}/assets/app-hash.js`)
    expect(script.headers.get("content-type")).toContain("text/javascript")
    expect(script.headers.get("cache-control")).toContain("immutable")
    expect(await script.text()).toBe("window.compiled = true")
    const head = await fetch(`${baseUrl}/assets/app-hash.js`, { method: "HEAD" })
    expect(head.status).toBe(200)
    expect(await head.text()).toBe("")
    expect((await fetch(`${baseUrl}/assets/missing.js`)).status).toBe(404)
    expect((await fetch(`${baseUrl}/@vite/client`)).headers.get("content-type")).toContain("text/html")
  })

  // 来源和路径越界不能读取本机文件或借用 API 权限。
  it("rejects foreign origins, invalid paths and static writes", async () => {
    const baseUrl = await start()
    expect((await fetch(`${baseUrl}/api/auth/status`, { headers: { Origin: "https://foreign.example" } })).status).toBe(403)
    expect((await fetch(`${baseUrl}/%2e%2e%5csecret`)).status).toBe(403)
    expect((await fetch(`${baseUrl}/%ZZ`)).status).toBe(400)
    expect((await fetch(baseUrl, { method: "POST" })).status).toBe(405)
  })

  // 真实上游收到 Cookie、请求体及 Range，响应状态和 Cookie 保持完整。
  it("streams authenticated API requests without consuming them as SPA routes", async () => {
    const received: Record<string, string | undefined> = {}
    backend = createServer((req, res) => {
      // 记录真正经过代理的认证头、范围和上传内容。
      received.cookie = req.headers.cookie
      received.range = req.headers.range
      received.origin = req.headers.origin
      received.url = req.url
      let body = ""
      req.on("data", chunk => { body += String(chunk) })
      req.on("end", () => {
        // 让客户端先收到第一段，以检查代理没有缓存完整响应。
        received.body = body
        res.writeHead(206, { "Content-Type": "application/octet-stream", "Set-Cookie": "session=next; HttpOnly; Path=/", "Content-Range": "bytes 0-5/6" })
        res.write("first")
        setTimeout(() => { res.end("!") }, 100)
      })
    })
    await new Promise<void>(resolve => { backend!.listen(0, "127.0.0.1", () => { resolve() }) })
    const address = backend.address()
    if (!address || typeof address === "string") throw new Error("Missing backend address")
    const upstream = `http://127.0.0.1:${address.port}`
    const baseUrl = await start(upstream)
    const response = await fetch(`${baseUrl}/api/media?fileId=123`, {
      method: "POST", headers: { Cookie: "session=original", Range: "bytes=0-5", Origin: baseUrl }, body: "payload",
    })
    expect(response.status).toBe(206)
    expect(response.headers.get("set-cookie")).toContain("session=next")
    const reader = response.body!.getReader()
    expect(new TextDecoder().decode((await reader.read()).value)).toBe("first")
    expect(new TextDecoder().decode((await reader.read()).value)).toBe("!")
    expect(received).toEqual({ cookie: "session=original", range: "bytes=0-5", origin: upstream, url: "/api/media?fileId=123", body: "payload" })
  })

  // 前端托管退出后释放监听，缺失的后端返回 502。
  it("reports unavailable APIs and releases its listener on quit", async () => {
    const baseUrl = await start("http://127.0.0.1:1")
    expect((await fetch(`${baseUrl}/api/health`)).status).toBe(502)
    await frontend!.stop()
    frontend = undefined
    await expect(fetch(baseUrl)).rejects.toThrow()
  })

  // 缺失入口不能被当作正常测试版本启动。
  it("fails before listening when the build is missing", async () => {
    await expect(startPrebuiltFrontend({ directory: path.join(os.tmpdir(), "curated-missing-build"), backendBaseUrl: "http://127.0.0.1:8080", baseUrl: "http://127.0.0.1:0" })).rejects.toThrow()
  })
})
