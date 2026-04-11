import { describe, expect, it } from "vitest"
import {
  buildAutoFillMovieGridTemplate,
  buildMovieGridChunkStyle,
} from "@/lib/movie-grid-template"

function acceptedGridTemplateColumns(value: string): string {
  const el = document.createElement("div")
  el.style.gridTemplateColumns = value
  return el.style.gridTemplateColumns
}

describe("buildAutoFillMovieGridTemplate", () => {
  it("builds a browser-accepted responsive grid template", () => {
    const template = buildAutoFillMovieGridTemplate("clamp(11.25rem, 9vw + 8.75rem, 15rem)")

    expect(template).toBe(
      "repeat(auto-fill, minmax(min(100%, clamp(11.25rem, 9vw + 8.75rem, 15rem)), 1fr))",
    )
    expect(acceptedGridTemplateColumns(template)).not.toBe("")
  })
})

describe("buildMovieGridChunkStyle", () => {
  it("keeps chunk boundary spacing inside the measured grid content", () => {
    const style = buildMovieGridChunkStyle({
      minTrackWidth: "clamp(11.25rem, 9vw + 8.75rem, 15rem)",
      gap: "clamp(1rem, 2vw, 1.5rem)",
    })

    expect(style.paddingBottom).toBe("clamp(1rem, 2vw, 1.5rem)")
  })
})
