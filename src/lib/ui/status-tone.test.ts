import { describe, expect, it } from "vitest"
import {
  statusBadgeClass,
  statusDotClass,
  statusPanelClass,
  statusTextClass,
} from "./status-tone"

describe("status tone classes", () => {
  it("maps status tones to semantic color tokens", () => {
    expect(statusTextClass("success")).toBe("text-success")
    expect(statusTextClass("warning")).toBe("text-warning")
    expect(statusTextClass("danger")).toBe("text-danger")
    expect(statusTextClass("info")).toBe("text-info")

    expect(statusDotClass("success")).toBe("bg-success shadow-sm ring-1 ring-success/40")
    expect(statusDotClass("warning")).toBe("bg-warning shadow-sm ring-1 ring-warning/40")
    expect(statusDotClass("danger")).toBe("bg-danger shadow-sm ring-1 ring-danger/40")
    expect(statusDotClass("info")).toBe("bg-info shadow-sm ring-1 ring-info/40")

    expect(statusBadgeClass("success")).toBe("border-success/35 bg-success/10 text-success")
    expect(statusBadgeClass("warning")).toBe("border-warning/35 bg-warning/10 text-warning")
    expect(statusBadgeClass("danger")).toBe("border-danger/35 bg-danger/10 text-danger")
    expect(statusBadgeClass("info")).toBe("border-info/35 bg-info/10 text-info")

    expect(statusPanelClass("info")).toBe(
      "rounded-2xl border border-info/25 border-l-[3px] border-l-info/60 bg-info/[0.07] px-4 py-3",
    )
  })
})
