import { describe, expect, it } from "vitest"
import {
  classifyMovieCodes,
  extractMovieNumber, extractMoviePartIndex,
  fileBaseName,
} from "./movie-number"

describe("extractMovieNumber", () => {
  // 扫描与 Mock 导入共用的番号解析必须兼容实际文件名及单字母分部。
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
    ["STAR-380A.mp4", "STAR-380"],
    ["STAR-380B.mp4", "STAR-380"],
    ["STAR-684A-C.mp4", "STAR-684"],
    ["STAR-684B-C.mp4", "STAR-684"],
    ["star-684b-c.mp4", "STAR-684"],
    ["ABC-123AB.mp4", ""],
    ["ABC-1231080p.mp4", ""],
    ["holiday.mp4", ""],
  ])("parses %s", (filename, expected) => {
    // 核对各命名形式归一后的作品番号，拒绝没有明确边界的尾缀。
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

// 番号与数字、字母分部独立解析；版本及质量标记不能成为分部。
it("extracts catalog and ordered parts independently", () => {
  // 验证普通番号、FC2 别名和字母分部与后端保持相同的序号含义。
  for (const [name, code, part] of [
    ["FC2PPV-1234567_2.mp4", "FC2-1234567", 2],
    ["ABC-123-part10.mp4", "ABC-123", 10],
    ["ABC-123_1.mp4", "ABC-123", 1],
    ["FC2-PPV-4942041-1.mp4", "FC2-4942041", 1],
    ["FC2-PPV-4942041-2.mp4", "FC2-4942041", 2],
    ["STAR-380A.mp4", "STAR-380", 1],
    ["STAR-380B.mp4", "STAR-380", 2],
    ["STAR-684A-C.mp4", "STAR-684", 1],
    ["STAR-684B-C.mp4", "STAR-684", 2],
    ["star-684b-c.mp4", "STAR-684", 2],
    ["STAR-684C.mp4", "STAR-684", 3],
    ["STAR-684Z.mp4", "STAR-684", 26],
    ["SSIS-562-C.mp4", "SSIS-562", 0],
    ["SSIS-588-C.mp4", "SSIS-588", 0],
    ["STAR-684-UC.mp4", "STAR-684", 0],
    ["ABC-123-1080p.mp4", "ABC-123", 0],
  ] as const) {
    expect(extractMovieNumber(name)).toBe(code)
    expect(extractMoviePartIndex(name)).toBe(part)
  }
})
