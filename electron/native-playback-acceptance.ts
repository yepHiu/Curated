/** 原生播放验收清单。Linux CI 只能覆盖假引擎，不能把跳过的实机项记成通过。 */
export type NativeAcceptanceStatus = "unit-covered" | "skipped" | "manual-windows"

export interface NativeAcceptanceInput {
  platform: string
  mpvPath?: string
  win32HostBuilt?: boolean
}

export interface NativeAcceptanceCase {
  id: string
  requirement: "REQ-0058" | "REQ-0059"
  status: NativeAcceptanceStatus
  note: string
}

/**
 * 按当前机器给出验收项状态。没有真实 Windows 引擎时，实机项必须是 skipped 或 manual-windows。
 */
export function nativePlaybackAcceptance(input: NativeAcceptanceInput): NativeAcceptanceCase[] {
  const enginePresent = Boolean(input.mpvPath?.trim()) && (input.platform === "win32" || input.platform === "darwin")
  const realPlayback = input.platform === "win32" && enginePresent && input.win32HostBuilt === true ? "manual-windows" : "skipped"
  return [
    { id: "engine-missing", requirement: "REQ-0059", status: "unit-covered", note: "空引擎在请求 Server 前返回 NATIVE_ENGINE_MISSING；资源探测区分 engine、host、page 和 platform。" },
    { id: "proxy-and-progress", requirement: "REQ-0058", status: "unit-covered", note: "代理 Range、认证隔离和协调器进度写入由假播放器覆盖。" },
    { id: "real-mpv-controls", requirement: "REQ-0058", status: realPlayback, note: "真实 mpv 的暂停、seek、倍速、音量和退出只在 Windows 且提供 CURATED_NATIVE_MPV 时才可能运行，本清单不把它记为已通过。" },
    { id: "win32-host-geometry", requirement: "REQ-0058", status: input.platform === "win32" && input.win32HostBuilt ? "manual-windows" : "skipped", note: "HWND 几何、最小化恢复和透明层命中需要 Windows 宿主。" },
    { id: "more-sources", requirement: "REQ-0059", status: "manual-windows", note: "除已记录的 CD1 外，还需要更多片源。Linux 不能代替。" },
    { id: "full-length-sync", requirement: "REQ-0059", status: "manual-windows", note: "整片音画同步需要在 Windows 上按验收步骤记录，不能用五分钟片段代替。" },
    { id: "mixed-dpi", requirement: "REQ-0059", status: "manual-windows", note: "跨屏混合 DPI 只能在真实多显示器的 Windows 上做。" },
    { id: "installer-lifecycle", requirement: "REQ-0059", status: "manual-windows", note: "干净安装、升级和卸载必须使用真实 Windows 安装包。" },
  ]
}
