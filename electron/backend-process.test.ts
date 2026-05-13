import { describe, expect, it } from "vitest"

import {
  defaultBackendBaseUrl,
  isCuratedHealthPayload,
  resolveBackendBaseUrl,
  resolveBackendLaunchPlan,
} from "./backend-process"

describe("Electron backend launch planning", () => {
  it("uses loopback dev and release defaults without exposing the desktop private server", () => {
    expect(defaultBackendBaseUrl(false)).toBe("http://127.0.0.1:8080")
    expect(defaultBackendBaseUrl(true)).toBe("http://127.0.0.1:8081")
  })

  it("normalizes an explicit backend URL override", () => {
    expect(
      resolveBackendBaseUrl({
        CURATED_ELECTRON_BACKEND_URL: " http://localhost:19090/ ",
      }, false),
    ).toBe("http://127.0.0.1:19090")
  })

  it("prefers an explicit backend executable before falling back to go run", () => {
    const plan = resolveBackendLaunchPlan({
      appPath: "C:/repo",
      env: { CURATED_ELECTRON_BACKEND_PATH: "C:/Curated/curated.exe" },
      isWindows: true,
      pathExists: (candidate) => candidate === "C:/Curated/curated.exe",
    })

    expect(plan.command).toBe("C:/Curated/curated.exe")
    expect(plan.args).toEqual(["-mode", "http"])
    expect(plan.cwd).toBe("C:/Curated")
  })

  it("runs the development backend binary from the backend directory", () => {
    const plan = resolveBackendLaunchPlan({
      appPath: "C:/repo",
      env: {},
      isWindows: true,
      pathExists: (candidate) => candidate === "C:/repo/backend/runtime/curated-dev.exe",
    })

    expect(plan.command).toBe("C:/repo/backend/runtime/curated-dev.exe")
    expect(plan.args).toEqual(["-mode", "http"])
    expect(plan.cwd).toBe("C:/repo/backend")
  })

  it("runs the packaged backend from the Electron app resources directory", () => {
    const plan = resolveBackendLaunchPlan({
      appPath: "C:/Program Files/Curated/resources/app",
      env: {},
      isWindows: true,
      pathExists: (candidate) => candidate === "C:/Program Files/Curated/resources/app/curated.exe",
    })

    expect(plan.command).toBe("C:/Program Files/Curated/resources/app/curated.exe")
    expect(plan.args).toEqual(["-mode", "http"])
    expect(plan.cwd).toBe("C:/Program Files/Curated/resources/app")
  })

  it("falls back to go run when no backend binary exists in development", () => {
    const plan = resolveBackendLaunchPlan({
      appPath: "C:/repo",
      env: {},
      isWindows: true,
      pathExists: () => false,
    })

    expect(plan.command).toBe("go")
    expect(plan.args).toEqual(["run", "./cmd/curated", "-mode", "http"])
    expect(plan.cwd).toBe("C:/repo/backend")
  })

  it("only treats Curated health payloads as attachable backends", () => {
    expect(isCuratedHealthPayload({ name: "curated" })).toBe(true)
    expect(isCuratedHealthPayload({ name: "curated-dev" })).toBe(true)
    expect(isCuratedHealthPayload({ name: "other-service" })).toBe(false)
    expect(isCuratedHealthPayload({})).toBe(false)
  })
})
