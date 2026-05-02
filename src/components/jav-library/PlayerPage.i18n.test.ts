import { describe, expect, it } from "vitest"
import source from "./PlayerPage.vue?raw"

describe("PlayerPage i18n", () => {
  it("uses locale keys for the detailed stats context menu labels", () => {
    expect(source).toMatch(/t\(["']player\.hideStats["']\)/)
    expect(source).toMatch(/t\(["']player\.showStats["']\)/)
    expect(source).not.toContain("关闭详细统计信息")
    expect(source).not.toContain("隐藏详细统计信息")
    expect(source).not.toContain('"详细统计信息"')
  })

  it("uses locale keys for immersive playback feedback labels", () => {
    expect(source).toMatch(/t\(["']player\.feedbackPlay["']\)/)
    expect(source).toMatch(/t\(["']player\.feedbackPause["']\)/)
  })
})
