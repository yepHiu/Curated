import { describe, expect, it } from "vitest"

import {
  defaultFrontendBaseUrl,
  resolveFrontendBaseUrl,
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
