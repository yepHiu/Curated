import { describe, expect, it } from "vitest"
import { formatMovieSummaryForDisplay } from "./format-movie-summary"

describe("formatMovieSummaryForDisplay", () => {
  it("converts br and newlines to spaces and strips other tags", () => {
    expect(
      formatMovieSummaryForDisplay("a<br><br>b<strong>x</strong>c"),
    ).toBe("a bxc")
  })

  it("collapses runs of newlines to a single space", () => {
    expect(formatMovieSummaryForDisplay("a\n\n\n\n\nb")).toBe("a b")
  })

  it("trims and normalizes CRLF", () => {
    expect(formatMovieSummaryForDisplay("  hi\r\n  ")).toBe("hi")
  })
})
