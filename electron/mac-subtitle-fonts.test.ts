import { describe, expect, it, afterEach } from "vitest"
import { mkdtemp, readFile } from "node:fs/promises"
import { tmpdir } from "node:os"
import path from "node:path"
import { JSDOM } from "jsdom"
import { MAC_SUBTITLE_FONT, buildMacFontConfigBody, prepareMacSubtitleFonts } from "./mac-subtitle-fonts"

describe("macOS subtitle fontconfig preparation", () => {
  // 使用真正的 XML 解析器验证特殊字符往返，而非仅匹配转义后的字符串。
  it("round-trips special characters in the font cache path through valid XML", () => {
    const directory = '/Users/A&B/<字幕> "test"/fontconfig'
    const body = buildMacFontConfigBody(directory)
    const document = new JSDOM(body, { contentType: "application/xml" })
    try {
      expect(document.window.document.querySelector("cachedir")?.textContent).toBe(path.join(directory, "cache").replace(/\\/g, "/"))
      expect(document.window.document.querySelectorAll("dir")).toHaveLength(3)
      expect(document.window.document.querySelector("edit string")?.textContent).toBe(MAC_SUBTITLE_FONT)
    } finally { document.window.close() }
  })
  const previousFile = process.env.FONTCONFIG_FILE
  const previousPath = process.env.FONTCONFIG_PATH

  /** 恢复测试前的 fontconfig 环境，避免污染其它用例。 */
  afterEach(() => {
    if (previousFile === undefined) delete process.env.FONTCONFIG_FILE
    else process.env.FONTCONFIG_FILE = previousFile
    if (previousPath === undefined) delete process.env.FONTCONFIG_PATH
    else process.env.FONTCONFIG_PATH = previousPath
  })

  it("builds a system-only fonts.conf that prefers PingFang SC and never references Homebrew", () => {
    const body = buildMacFontConfigBody("/tmp/curated-fontconfig")
    expect(body).toContain("/System/Library/Fonts")
    expect(body).toContain("/System/Library/Fonts/Supplemental")
    expect(body).toContain("/Library/Fonts")
    expect(body).toContain(MAC_SUBTITLE_FONT)
    expect(body).not.toContain("/opt/homebrew")
    expect(body).not.toContain("/usr/local")
  })

  it("applies FONTCONFIG_FILE only on darwin and clears Homebrew FONTCONFIG_PATH", async () => {
    if (process.platform !== "darwin") {
      expect(await prepareMacSubtitleFonts()).toBeUndefined()
      return
    }
    const directory = await mkdtemp(path.join(tmpdir(), "curated-fontconfig-test-"))
    process.env.FONTCONFIG_PATH = "/opt/homebrew/etc/fonts"
    const file = await prepareMacSubtitleFonts(directory)
    expect(file).toBe(path.join(directory, "fonts.conf"))
    expect(process.env.FONTCONFIG_FILE).toBe(file)
    expect(process.env.FONTCONFIG_PATH).toBeUndefined()
    expect(await readFile(file!, "utf8")).toBe(buildMacFontConfigBody(directory))
  })
})
