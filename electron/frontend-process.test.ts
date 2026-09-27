import { describe, expect, it, vi } from "vitest"

import {
  defaultFrontendBaseUrl,
  resolveFrontendBaseUrl,
  resolveRendererBaseUrl,
  resolveFrontendLaunchPlan,
  shouldStartDevFrontend,
  shouldStopFrontendOnQuit,
} from "./frontend-process"

describe("Electron frontend launch planning", () => {
  it("uses the loopback Vite dev server as the desktop renderer in development", () => {
    expect(defaultFrontendBaseUrl()).toBe("http://127.0.0.1:5173")
    expect(shouldStartDevFrontend({ isPackaged: false })).toBe(true)
    expect(shouldStartDevFrontend({ isPackaged: true })).toBe(false)
  })

  it("normalizes an explicit frontend URL override", () => {
    expect(
      resolveFrontendBaseUrl({
        CURATED_ELECTRON_FRONTEND_URL: " http://localhost:5174/some/path ",
      }),
    ).toBe("http://127.0.0.1:5174")
  })

  it("starts Vite from the repository root on the selected loopback port", () => {
    const plan = resolveFrontendLaunchPlan({
      appPath: "C:/repo",
      baseUrl: "http://127.0.0.1:5173",
      isWindows: true,
    })

    expect(plan.command).toBe("cmd.exe")
    expect(plan.args).toEqual([
      "/d",
      "/s",
      "/c",
      "pnpm.cmd",
      "exec",
      "vite",
      "--host",
      "127.0.0.1",
      "--port",
      "5173",
    ])
    expect(plan.cwd).toBe("C:/repo")
  })

  it("points the Vite renderer at the Electron-managed backend API", () => {
    const plan = resolveFrontendLaunchPlan({
      appPath: "C:/repo",
      baseUrl: "http://127.0.0.1:5173",
      backendBaseUrl: "http://127.0.0.1:18080",
      env: { VITE_API_BASE_URL: undefined },
      isWindows: true,
    })

    expect(plan.env).toMatchObject({
      VITE_USE_WEB_API: "true",
      VITE_API_BASE_URL: "http://127.0.0.1:18080/api",
      BROWSER: "none",
    })
  })

  it("only stops a dev frontend that Electron spawned itself", () => {
    expect(shouldStopFrontendOnQuit({ attachedToExistingFrontend: false })).toBe(true)
    expect(shouldStopFrontendOnQuit({ attachedToExistingFrontend: true })).toBe(false)
  })
})

describe("standalone development renderer selection", () => {
  const backend = "http://127.0.0.1:8080"
  const response = (body: string, type = "application/json") => new Response(body, { status: 200, headers: { "content-type": type } })

  it("loads the running Vite UI for the local development Server", async () => {
    const fetchImpl = vi.fn()
      .mockResolvedValueOnce(response('{"name":"curated-dev"}'))
      .mockResolvedValueOnce(response('<script type="module" src="/src/main.ts?t=123"></script>', "text/html"))
    expect(await resolveRendererBaseUrl({ backendBaseUrl: backend, isPackaged: false, fetchImpl })).toBe("http://127.0.0.1:5173")
    expect(fetchImpl).toHaveBeenCalledTimes(2)
  })

  it("recognizes localhost as the same local development Server", async () => {
    const fetchImpl = vi.fn()
      .mockResolvedValueOnce(response('{"name":"curated-dev"}'))
      .mockResolvedValueOnce(response('<script type="module" src="/src/main.ts"></script>', "text/html"))
    expect(await resolveRendererBaseUrl({ backendBaseUrl: "http://localhost:8080", isPackaged: false, fetchImpl })).toBe("http://127.0.0.1:5173")
  })

  it("never redirects packaged or remote Servers to the local Vite UI", async () => {
    const fetchImpl = vi.fn()
    expect(await resolveRendererBaseUrl({ backendBaseUrl: backend, isPackaged: true, fetchImpl })).toBe(backend)
    expect(await resolveRendererBaseUrl({ backendBaseUrl: "http://192.168.1.20:8080", isPackaged: false, fetchImpl })).toBe("http://192.168.1.20:8080")
    expect(fetchImpl).not.toHaveBeenCalled()
  })

  it("reports an unavailable Vite UI before opening an empty Server page", async () => {
    const fetchImpl = vi.fn()
      .mockResolvedValueOnce(response('{"name":"curated-dev"}'))
      .mockRejectedValueOnce(new Error("Connection refused"))
    await expect(resolveRendererBaseUrl({ backendBaseUrl: backend, isPackaged: false, fetchImpl })).rejects.toThrow("pnpm dev")
  })

  it("keeps a non-development Server on its own hosted UI", async () => {
    const fetchImpl = vi.fn().mockResolvedValue(response('{"name":"curated"}'))
    expect(await resolveRendererBaseUrl({ backendBaseUrl: backend, isPackaged: false, fetchImpl })).toBe(backend)
  })
})
