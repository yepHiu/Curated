import { describe, expect, it } from "vitest"
import { parseResumeSecondsFromQuery } from "@/lib/playback-progress-storage"

describe("parseResumeSecondsFromQuery", () => {
  it("parses fractional second route queries", () => {
    expect(parseResumeSecondsFromQuery("123.456")).toBe(123.456)
  })
})
