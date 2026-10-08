import path from "node:path"
import { existsSync } from "node:fs"
import type { DesktopPlaybackCapabilities } from "./playback-contract.js"

/** 不从 PATH 搜索任意引擎；固定安装位置或主进程显式绝对路径。 */
export function playbackResources(options: { directory: string; appPath: string; packaged: boolean },
  platform = process.platform, env = process.env, exists: (file: string) => boolean = existsSync) {
  const mac = platform === "darwin"
  const supported = mac || platform === "win32"
  const helper = path.join(options.directory, mac ? "native-player-macos.node" : "native-player-host.exe")
  const page = path.join(options.directory, "player", "index.html")
  const candidates = mac ? [
    env.CURATED_NATIVE_LIBMPV,
    "/opt/homebrew/opt/mpv/lib/libmpv.dylib",
    "/usr/local/opt/mpv/lib/libmpv.dylib",
  ] : [options.packaged ? path.join(options.appPath, "native-player", "mpv.exe")
    : env.CURATED_NATIVE_MPV || path.join(options.appPath, ".workspace", "native-player", "mpv.exe")]
  // 显式覆盖无效时失败，避免误用其它版本掩盖配置问题。
  const override = mac ? env.CURATED_NATIVE_LIBMPV : !options.packaged ? env.CURATED_NATIVE_MPV : undefined
  const executable = override || candidates.find(candidate => candidate && path.isAbsolute(candidate) && exists(candidate)) || ""
  const reason: DesktopPlaybackCapabilities["reason"] = !supported ? "platform" : !path.isAbsolute(executable) || !exists(executable) ? "engine"
    : !exists(helper) ? "host" : !exists(page) ? "page" : undefined
  return { executable, helper, page, available: reason === undefined, reason }
}
