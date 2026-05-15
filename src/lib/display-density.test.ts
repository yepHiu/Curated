import { describe, expect, it } from "vitest"

import {
  RETINA_DESKTOP_DENSITY_QUERY,
  resolveMovieGridDensity,
} from "@/lib/display-density"

describe("display density", () => {
  it("keeps compact desktop density behind a DPR 2 media query", () => {
    expect(RETINA_DESKTOP_DENSITY_QUERY).toContain("min-resolution: 2dppx")
    expect(RETINA_DESKTOP_DENSITY_QUERY).not.toContain("1.5dppx")
  })

  it("leaves default grid density larger than Retina compact density", () => {
    const defaultDensity = resolveMovieGridDensity(false)
    const compactDensity = resolveMovieGridDensity(true)

    expect(defaultDensity.minTrackPx).toBe(188)
    expect(defaultDensity.gapPxEstimate).toBe(20)
    expect(compactDensity.minTrackPx).toBeLessThan(defaultDensity.minTrackPx)
    expect(compactDensity.gapPxEstimate).toBeLessThan(defaultDensity.gapPxEstimate)
    expect(compactDensity.minTrackWidth).toBe("var(--movie-grid-min-track)")
    expect(compactDensity.gap).toBe("var(--movie-grid-gap)")
  })
})
