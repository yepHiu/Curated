import { describe, expect, it } from "vitest"
import { isLocalUpdateTarget } from "./app-update-target"
import type { DesktopInfo } from "../../electron/desktop-contract"

const info: DesktopInfo = { version: "0.1.0", buildStamp: "20260926.123433", distribution: "legacy", development: false, platform: "windows", arch: "x64", serverOrigin: "http://127.0.0.1:8080" }

describe("update target", () => {
  it("allows matching local legacy Desktop and direct local browser connections", () => {
    expect(isLocalUpdateTarget("http://127.0.0.1:8080/api", "http://127.0.0.1:5173", true, info)).toBe(true)
    for (const origin of ["http://127.0.0.1:8081", "http://localhost:8081", "http://[::1]:8081"]) {
      expect(isLocalUpdateTarget("/api", origin, false, null)).toBe(true)
    }
  })

  it("allows standalone Desktop to update only a matching local standalone Server", () => {
    const desktop = { ...info, distribution: "desktop" as const }
    expect(isLocalUpdateTarget("http://127.0.0.1:8080/api", "http://127.0.0.1:5173", true, desktop, "curated-server-stable")).toBe(true)
    expect(isLocalUpdateTarget("http://127.0.0.1:8080/api", "http://127.0.0.1:5173", true, desktop, "github-releases")).toBe(false)
    expect(isLocalUpdateTarget("https://nas.example/api", "https://nas.example", true, desktop, "curated-server-stable")).toBe(false)
  })

  it("rejects remote, ambiguous, old-bridge and standalone update targets", () => {
    for (const target of ["https://nas.example/api", "http://192.168.1.20:8081/api", "http://localhost.evil/api", "file:///api"]) {
      expect(isLocalUpdateTarget(target, "http://localhost:5173", false, null)).toBe(false)
    }
    expect(isLocalUpdateTarget("/api", "https://nas.example", false, null)).toBe(false)
    for (const candidate of [null, { ...info, serverOrigin: undefined }, { ...info, serverOrigin: "https://nas.example" }, { ...info, distribution: "desktop" as const }]) {
      expect(isLocalUpdateTarget("http://127.0.0.1:8080/api", "http://127.0.0.1:5173", true, candidate)).toBe(false)
    }
  })
})
