import { addWatchTimeDelta } from "@/lib/playback-watch-time-storage"
import { createPlaybackWatchTimeTracker as createCore, type PlaybackWatchTimeTrackerOptions } from "../../electron/playback-watch-time-core"
export type { PlaybackWatchTimeTracker, WatchTimeDeltaSink } from "../../electron/playback-watch-time-core"
export type BrowserWatchTimeTrackerOptions = Omit<PlaybackWatchTimeTrackerOptions, "addDelta"> & { addDelta?: PlaybackWatchTimeTrackerOptions["addDelta"] }
/** Web 默认注入现有存储；Desktop 使用同一纯计算并注入当前 Server 的 HTTP sink。 */
export function createPlaybackWatchTimeTracker(options: BrowserWatchTimeTrackerOptions) {
  return createCore({ ...options, addDelta: options.addDelta ?? addWatchTimeDelta })
}
