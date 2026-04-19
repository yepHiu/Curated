import { describe, expect, it } from "vitest"
import {
  formatAboutBackendVersion,
  formatAboutInstallerVersion,
} from "@/lib/about-version"
import type { HealthDTO } from "@/api/types"

describe("formatAboutBackendVersion", () => {
  it("appends channel when present", () => {
    const health: HealthDTO = {
      name: "Curated",
      version: "20260412.165224",
      channel: "release",
      transport: "http",
      databasePath: "runtime/curated.db",
    }

    expect(formatAboutBackendVersion(health)).toBe("20260412.165224-release")
  })
})

describe("formatAboutInstallerVersion", () => {
  it("returns installer version when present", () => {
    const health: HealthDTO = {
      name: "Curated",
      version: "20260412.165224",
      channel: "release",
      transport: "http",
      databasePath: "runtime/curated.db",
      installerVersion: "1.1.3",
    }

    expect(formatAboutInstallerVersion(health)).toBe("1.1.3")
  })

  it("returns trimmed installer version for dev fallback values too", () => {
    const health: HealthDTO = {
      name: "Curated",
      version: "20260412.165224",
      channel: "dev",
      transport: "http",
      databasePath: "runtime/curated.db",
      installerVersion: " 0.0.0 ",
    }

    expect(formatAboutInstallerVersion(health)).toBe("0.0.0")
  })
})
