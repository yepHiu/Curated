import http from "node:http";
import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createScanStore } from "./scan-store.mjs";
import { renderOfflineHtml, root } from "./code-atlas.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const dist = path.join(root, ".workspace/code-atlas-dist");
const mime = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".svg": "image/svg+xml",
  ".woff2": "font/woff2",
};

export async function startAtlasServer({
  port = 8795,
  scanner,
  preview = false,
  withVite = true,
} = {}) {
  const store = createScanStore(scanner);
  let vite;
  let boundPort;
  function json(res, code, body) {
    res.writeHead(code, {
      "Content-Type": "application/json; charset=utf-8",
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
    });
    res.end(JSON.stringify(body));
  }
  const server = http.createServer(async (req, res) => {
    try {
      const allowedHosts = [`127.0.0.1:${boundPort}`, `localhost:${boundPort}`];
      if (!allowedHosts.includes(req.headers.host))
        return json(res, 403, { error: "仅允许本机地址访问" });
      if (
        req.headers.origin &&
        !allowedHosts.some((host) => req.headers.origin === `http://${host}`)
      )
        return json(res, 403, { error: "不接受跨来源请求" });
      const url = new URL(req.url, `http://127.0.0.1:${boundPort}`);
      if (url.pathname.startsWith("/api/atlas/")) {
        if (req.method === "GET" && url.pathname === "/api/atlas/status")
          return json(res, 200, store.status());
        if (req.method === "POST" && url.pathname === "/api/atlas/scan") {
          if (req.headers["x-code-atlas"] !== "1")
            return json(res, 403, { error: "缺少本地操作标识" });
          req.resume();
          void store.scan();
          return json(res, 202, store.status());
        }
        if (
          req.method === "GET" &&
          ["/api/atlas/snapshot", "/api/atlas/export"].includes(url.pathname)
        ) {
          const data = store.snapshot();
          if (!data)
            return json(res, 503, {
              error: "还没有成功的扫描结果",
              ...store.status(),
            });
          if (url.pathname.endsWith("/snapshot"))
            return json(res, 200, {
              sessionId: store.status().sessionId,
              revision: store.status().revision,
              snapshot: data,
            });
          res.writeHead(200, {
            "Content-Type": "text/html; charset=utf-8",
            "Content-Disposition":
              'attachment; filename="curated-code-atlas.html"',
            "Cache-Control": "no-store",
          });
          return res.end(renderOfflineHtml(data));
        }
        return json(res, 404, { error: "分析工具接口不存在" });
      }
      if (req.method !== "GET" && req.method !== "HEAD")
        return json(res, 405, { error: "Method not allowed" });
      if (vite)
        return vite.middlewares(req, res, () =>
          json(res, 404, { error: "Not found" }),
        );
      if (preview) {
        const relative = decodeURIComponent(
          url.pathname === "/" ? "/index.html" : url.pathname,
        );
        const target = path.resolve(dist, `.${relative}`);
        if (!target.startsWith(dist + path.sep))
          return json(res, 403, { error: "无效路径" });
        try {
          const body = await fs.readFile(target);
          res.writeHead(200, {
            "Content-Type":
              mime[path.extname(target)] || "application/octet-stream",
            "Cache-Control": "no-store",
          });
          return res.end(req.method === "HEAD" ? undefined : body);
        } catch (error) {
          if (error.code !== "ENOENT" && error.code !== "EISDIR") throw error;
          return json(res, 404, {
            error: "页面不存在；请先运行 pnpm code:atlas:build",
          });
        }
      }
      return json(res, 404, { error: "Not found" });
    } catch (error) {
      if (!res.headersSent)
        json(res, 500, {
          error: error instanceof Error ? error.message : String(error),
        });
      else res.end();
    }
  });
  try {
    if (preview) await fs.access(path.join(dist, "index.html"));
    if (withVite && !preview) {
      const { createServer } = await import("vite");
      vite = await createServer({
        configFile: path.join(here, "vite.config.ts"),
        server: {
          middlewareMode: true,
          hmr: { server },
          fs: { allow: [root] },
        },
      });
    }
    await new Promise((resolve, reject) => {
      server.once("error", reject);
      server.listen(port, "127.0.0.1", () => {
        server.removeListener("error", reject);
        resolve();
      });
    });
    boundPort = server.address().port;
  } catch (error) {
    store.dispose();
    await vite?.close();
    throw error;
  }
  void store.scan();
  return {
    url: `http://127.0.0.1:${boundPort}`,
    store,
    async close() {
      store.dispose();
      await vite?.close();
      const closed = new Promise((resolve) => server.close(resolve));
      server.closeAllConnections();
      await closed;
    },
  };
}

if (
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  const args = process.argv.slice(2);
  const index = args.indexOf("--port");
  const port = index < 0 ? 8795 : Number(args[index + 1]);
  if (!Number.isInteger(port) || port < 1 || port > 65535)
    throw new Error("--port 必须是 1–65535 的整数");
  try {
    const app = await startAtlasServer({
      port,
      preview: args.includes("--preview"),
    });
    console.log(
      `Curated Code Atlas: ${app.url}\n正在扫描当前仓库。按 Ctrl+C 停止。`,
    );
    let closing = false;
    const close = async () => {
      if (closing) return;
      closing = true;
      await app.close();
      process.exit(0);
    };
    process.on("SIGINT", close);
    process.on("SIGTERM", close);
  } catch (error) {
    console.error(
      error.code === "EADDRINUSE"
        ? `端口 ${port} 已被占用；可运行 pnpm code:atlas --port 8796`
        : error.message,
    );
    process.exitCode = 1;
  }
}
