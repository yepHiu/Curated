import { mkdir, writeFile } from "node:fs/promises"
import { homedir, tmpdir } from "node:os"
import path from "node:path"

/** 正式包与开发态共用的 macOS 系统字体目录；禁止指向 Homebrew 前缀。 */
const SYSTEM_FONT_DIRS = [
  "/System/Library/Fonts",
  "/System/Library/Fonts/Supplemental",
  "/Library/Fonts",
]

/** libass/mpv 在缺少有效 fontconfig 时回退的系统 CJK 字体族。 */
export const MAC_SUBTITLE_FONT = "PingFang SC"

let prepared: Promise<string | undefined> | undefined

/**
 * 生成仅枚举系统字体目录的 fontconfig，并写入进程环境。
 * libmpv 在进程内加载，必须在引擎初始化前设置 FONTCONFIG_FILE。
 */
export async function prepareMacSubtitleFonts(cacheDirectory = defaultCacheDirectory()): Promise<string | undefined> {
  if (process.platform !== "darwin") return undefined
  prepared ??= writeMacFontConfig(cacheDirectory).catch(() => undefined)
  const file = await prepared
  if (file) {
    process.env.FONTCONFIG_FILE = file
    // 清空可能指向 Homebrew cellar 的路径，避免正式包读 /opt/homebrew。
    delete process.env.FONTCONFIG_PATH
  }
  return file
}

/** 选择可写缓存目录：优先用户主目录 Application Support，失败时回退临时目录。 */
function defaultCacheDirectory(): string {
  try {
    return path.join(homedir(), "Library", "Application Support", "Curated", "fontconfig")
  } catch {
    return path.join(tmpdir(), "curated-fontconfig")
  }
}

/**
 * 构造最小 fonts.conf 正文，仅允许系统字体目录并优先 PingFang SC。
 * 纯函数便于跨平台单测，不触碰环境变量。
 */
export function buildMacFontConfigBody(cacheDirectory: string): string {
  // 系统目录固定；缓存路径来自用户目录，必须作为 XML 文本写入。
  const dirs = SYSTEM_FONT_DIRS.map(dir => {
    // 每个目录独立形成 fontconfig 的字体扫描项。
    return `  <dir>${xmlText(dir)}</dir>`
  }).join("\n")
  const cache = path.join(cacheDirectory, "cache").replace(/\\/g, "/")
  return `<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "urn:fontconfig:fonts.dtd">
<fontconfig>
${dirs}
  <cachedir>${xmlText(cache)}</cachedir>
  <match target="pattern">
    <test name="family"><string>sans-serif</string></test>
    <edit name="family" mode="prepend" binding="strong"><string>${MAC_SUBTITLE_FONT}</string></edit>
  </match>
</fontconfig>
`
}

/** 转义 XML 文本节点，保证合法路径不会被解释为实体或标签。 */
function xmlText(value: string): string {
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
}

/** 写入最小 fonts.conf，仅允许系统字体目录。 */
async function writeMacFontConfig(cacheDirectory: string): Promise<string> {
  await mkdir(cacheDirectory, { recursive: true })
  const file = path.join(cacheDirectory, "fonts.conf")
  await writeFile(file, buildMacFontConfigBody(cacheDirectory), { encoding: "utf8", mode: 0o600 })
  return file
}
