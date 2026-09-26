import { afterEach, describe, expect, it, vi } from "vitest"
import { devRemoteSimulation, setDevRemoteSimulation } from "./dev-remote-simulation"

afterEach(() => { setDevRemoteSimulation(false); vi.unstubAllEnvs() })

describe("remote client simulation", () => {
  it("is disabled by default and reversible in development", () => {
    expect(devRemoteSimulation.value).toBe(false)
    setDevRemoteSimulation(true)
    expect(devRemoteSimulation.value).toBe(true)
    setDevRemoteSimulation(false)
    expect(devRemoteSimulation.value).toBe(false)
  })

  it("cannot be enabled in production", async () => {
    vi.stubEnv("DEV", false)
    vi.resetModules()
    const production = await import("./dev-remote-simulation")
    production.setDevRemoteSimulation(true)
    expect(production.devRemoteSimulation.value).toBe(false)
  })
})
