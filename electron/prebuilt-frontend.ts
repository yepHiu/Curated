import { createReadStream } from "node:fs"
import { realpath, stat } from "node:fs/promises"
import { createServer, request, type IncomingMessage, type ServerResponse } from "node:http"
import path from "node:path"

export interface PrebuiltFrontend {
  baseUrl: string
  stop: () => Promise<void>
}

const contentTypes: Record<string, string> = {
  ".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8", ".json": "application/json; charset=utf-8",
  ".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg", ".webp": "image/webp", ".ico": "image/x-icon",
  ".woff": "font/woff", ".woff2": "font/woff2", ".ttf": "font/ttf",
}

/** 只有显式启用预编译页面的本机开发测试连接使用该托管器。 */
export function shouldUsePrebuiltFrontend(options: {
  isPackaged: boolean
  backendBaseUrl: string
  env?: NodeJS.ProcessEnv
}): boolean {
  if (options.isPackaged || !(options.env ?? process.env).CURATED_ELECTRON_FRONTEND_DIR?.trim()) return false
  const backend = new URL(options.backendBaseUrl)
  return backend.protocol === "http:" && ["127.0.0.1", "localhost"].includes(backend.hostname) && backend.port === "8080"
}

/** 在 Electron 主进程内提供构建产物和同源 API 转发，不启动编译器或文件监听。 */
export async function startPrebuiltFrontend(options: {
  directory: string
  backendBaseUrl: string
  baseUrl: string
}): Promise<PrebuiltFrontend> {
  const frontend = new URL(options.baseUrl)
  const backend = new URL(options.backendBaseUrl)
  if (frontend.protocol !== "http:" || frontend.hostname !== "127.0.0.1" || !frontend.port ||
      backend.protocol !== "http:" || !["127.0.0.1", "localhost"].includes(backend.hostname) || frontend.origin === backend.origin) {
    throw new Error("预编译测试页面必须使用独立的本机 HTTP 地址。")
  }
  const root = await realpath(options.directory)
  if (!(await stat(path.join(root, "index.html"))).isFile()) throw new Error("请先运行 pnpm desktop:test:build。")
  const server = createServer((req, res) => {
    // 限定 Host 和浏览器来源，避免其它网页借用本机测试代理访问 Server。
    if (req.headers.host !== frontend.host || (req.headers.origin && req.headers.origin !== frontend.origin)) {
      res.writeHead(403).end()
      return
    }
    if (req.url === "/api" || req.url?.startsWith("/api/") || req.url?.startsWith("/api?")) {
      proxyApi(req, res, backend, frontend.origin)
      return
    }
    void serveFile(req, res, root).catch(() => {
      // 文件在读取期间被替换或请求取消时，关闭该响应而不结束 Desktop。
      if (!res.headersSent) res.writeHead(500, { "Cache-Control": "no-store" }).end()
      else res.destroy()
    })
  })
  await new Promise<void>((resolve, reject) => {
    // 端口冲突直接报错，不复用来源不明的开发服务器。
    server.once("error", reject)
    server.listen(Number(frontend.port), frontend.hostname, () => {
      // 就绪后移除仅用于启动阶段的错误回调。
      server.removeListener("error", reject)
      const address = server.address()
      if (address && typeof address !== "string") frontend.port = String(address.port)
      resolve()
    })
  })
  return {
    baseUrl: frontend.origin,
    /** 真正退出 Desktop 时回收 HTTP 监听和未完成的长连接。 */
    stop: () => new Promise<void>((resolve, reject) => {
      // 先停止接收请求，再关闭剩余流连接，避免 SSE 阻止退出。
      server.close(error => { if (error) reject(error); else resolve() })
      server.closeAllConnections()
    }),
  }
}

/** 读取构建目录内的静态文件；页面路由回退入口，缺失资源保持 404。 */
async function serveFile(req: IncomingMessage, res: ServerResponse, root: string): Promise<void> {
  if (req.method !== "GET" && req.method !== "HEAD") {
    res.writeHead(405, { Allow: "GET, HEAD" }).end()
    return
  }
  let pathname: string
  try { pathname = decodeURIComponent((req.url ?? "/").split("?")[0]!) }
  catch { res.writeHead(400).end(); return }
  if (pathname.includes("\\") || pathname.includes("\0") || pathname.split("/").includes("..")) {
    res.writeHead(403).end()
    return
  }
  let filename = path.resolve(root, `.${pathname === "/" ? "/index.html" : pathname}`)
  if (!isInside(root, filename)) { res.writeHead(403).end(); return }
  let file = await stat(filename).catch(() => {
    // 缺失静态资源由下方决定是否允许 SPA 回退。
    return undefined
  })
  if (!file?.isFile()) {
    if (pathname.startsWith("/assets/") || path.extname(pathname)) {
      res.writeHead(404, { "Cache-Control": "no-store" }).end()
      return
    }
    filename = path.join(root, "index.html")
    file = await stat(filename)
  }
  if (!isInside(root, await realpath(filename))) { res.writeHead(403).end(); return }
  const isEntry = filename === path.join(root, "index.html")
  res.writeHead(200, {
    "Content-Type": contentTypes[path.extname(filename)] ?? "application/octet-stream",
    "Content-Length": file.size,
    "Cache-Control": isEntry ? "no-store" : pathname.startsWith("/assets/") ? "public, max-age=31536000, immutable" : "no-cache",
    "X-Content-Type-Options": "nosniff",
  })
  if (req.method === "HEAD") { res.end(); return }
  const stream = createReadStream(filename)
  // 流读取失败只关闭本次请求；客户端退出时停止读取文件。
  stream.on("error", () => { res.destroy() })
  // 客户端关闭页面后，不再读取剩余文件数据。
  res.on("close", () => { stream.destroy() })
  stream.pipe(res)
}

/** 限制真实路径留在构建目录内，包括 Windows 盘符和符号链接边界。 */
function isInside(root: string, filename: string): boolean {
  const relative = path.relative(root, filename)
  return relative !== ".." && !relative.startsWith(`..${path.sep}`) && !path.isAbsolute(relative)
}

/** 流式转发同源 API，保留登录 Cookie、上传、Range 和 SSE，避免整段媒体驻留内存。 */
function proxyApi(req: IncomingMessage, res: ServerResponse, backend: URL, frontendOrigin: string): void {
  const headers = { ...req.headers, host: backend.host }
  if (headers.origin === frontendOrigin) headers.origin = backend.origin
  if (headers.referer?.startsWith(`${frontendOrigin}/`)) headers.referer = backend.origin + headers.referer.slice(frontendOrigin.length)
  const upstream = request(new URL(req.url!, backend), { method: req.method, headers }, response => {
    // 状态、Cookie 和 Range 响应头原样保留；数据直接通过流传输。
    res.writeHead(response.statusCode ?? 502, response.headers)
    // 上游流断开时关闭对应下游，不能发送一个不完整的成功响应。
    response.on("error", () => { res.destroy() })
    response.pipe(res)
  })
  upstream.on("error", () => {
    // 后端断开时给出明确失败，不能回退到静态入口页面。
    if (!res.headersSent) res.writeHead(502, { "Cache-Control": "no-store" }).end()
    else res.destroy()
  })
  // 客户端取消上传时，连同上游请求一起取消。
  req.on("aborted", () => { upstream.destroy() })
  // 客户端停止播放或关闭 SSE 时，立即回收媒体/事件上游。
  res.on("close", () => { upstream.destroy() })
  req.pipe(upstream)
}
