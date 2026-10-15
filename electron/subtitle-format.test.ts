import { describe, expect, it } from "vitest"
import { activeSubtitleText, defaultSubtitleDisplay, mergeSubtitleSRT, parseSRT, validateSubtitleDisplay } from "./subtitle-format"
describe("subtitle timeline and display boundaries", () => {
  it("parses CRLF/BOM safely, strips markup and handles overlapping and half-open cues", () => {
    const cues = parseSRT('\uFEFF1\r\n00:00:01,000 --> 00:00:04,000\r\n<b>你好</b>\r\n\r\n2\r\n00:00:02,000 --> 00:00:03,000\r\n{\\pos(0,0)}世界\r\n')
    expect(activeSubtitleText(cues, 0.9)).toBe("")
    expect(activeSubtitleText(cues, 2)).toBe("你好\n世界")
    expect(activeSubtitleText(cues, 3)).toBe("你好")
    expect(activeSubtitleText(cues, 4)).toBe("")
    expect(activeSubtitleText(cues, 2 - 1.5)).toBe("") // A positive offset delays the cue.
  })
  it("merges bilingual cues by absolute timestamps and rejects invalid settings", () => {
    const zh = "1\n01:10:00,100 --> 01:10:01,900\n你好\n"
    const ja = "1\n01:10:00,100 --> 01:10:01,900\nこんにちは\n"
    expect(parseSRT(mergeSubtitleSRT(zh, ja))[0]).toMatchObject({ begin: 4200.1, end: 4201.9, text: "你好\nこんにちは" })
    expect(() => validateSubtitleDisplay({ ...defaultSubtitleDisplay, offset: Infinity })).toThrow()
    expect(() => validateSubtitleDisplay({ ...defaultSubtitleDisplay, size: 200 })).toThrow()
  })
})
