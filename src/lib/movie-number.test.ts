import { describe, expect, it } from "vitest"
import {
  classifyMovieCodes,
  extractMovieNumber, extractMoviePartIndex,
  fileBaseName,
} from "./movie-number"

describe("extractMovieNumber", () => {
  it.each([
    ["489155.com@FC2PPV-4854489.mp4", "FC2-4854489"],
    ["489155.com@START-483.mp4", "START-483"],
    ["489155.com@EBWH-287-C.mp4", "EBWH-287"],
    ["IPZZ-708-C.mp4", "IPZZ-708"],
    ["abc123.mkv", "ABC-123"],
    ["ABC_123.avi", "ABC-123"],
    ["FC2PPV-123456.mp4", "FC2-123456"],
    ["heyzo_4321.mp4", "HEYZO-4321"],
    ["ABP-100-UC.mp4", "ABP-100"],
    ["folder\\SSIS-001.mkv", "SSIS-001"],
    ["holiday.mp4", ""],
  ])("parses %s", (filename, expected) => {
    expect(extractMovieNumber(filename)).toBe(expected)
  })
})

describe("classifyMovieCodes", () => {
  it("treats hyphen variants as exact", () => {
    expect(classifyMovieCodes("SSIS-001", "ssis001")).toBe("exact")
    expect(classifyMovieCodes("SSIS_001", "SSIS-001")).toBe("exact")
  })

  it("treats disc suffixes as similar", () => {
    expect(classifyMovieCodes("SSIS-001", "SSIS-001-CD1")).toBe("similar")
  })

  it("does not match neighboring numbers", () => {
    expect(classifyMovieCodes("ABC-12", "ABC-123")).toBe("")
  })
})

describe("fileBaseName", () => {
  it("strips windows and posix directories", () => {
    expect(fileBaseName("a\\b\\c.mp4")).toBe("c.mp4")
    expect(fileBaseName("a/b/c.mp4")).toBe("c.mp4")
  })
})

// 下划线分部标记不能导致 FC2 退化成 PPV 番号，质量数字也不能成为分部。
it("extracts catalog and ordered parts independently", () => {
    // 验证 extracts catalog and ordered parts independently。
 for (const [name, code, part] of [
  ["FC2PPV-1234567_2.mp4", "FC2-1234567", 2],
  ["ABC-123-part10.mp4", "ABC-123", 10],
  ["ABC-123_1.mp4", "ABC-123", 1],
  ["ABC-123-1080p.mp4", "ABC-123", 0],
 ] as const) {
  expect(extractMovieNumber(name)).toBe(code)
  expect(extractMoviePartIndex(name)).toBe(part)
 }
})
