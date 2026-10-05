/** 原型状态不包含凭据、媒体能力地址或可执行参数。 */
export interface NativePlayerState {
  status: "idle" | "starting" | "playing" | "paused" | "ended" | "stopped" | "error"
  positionSec: number
  durationSec: number
  speed: number
  volume: number
  droppedFrames: number
  decoderDroppedFrames: number
  codec: string
  hwdec: string
  error?: string
  progressError?: string
  movieId?: string
  fileId?: string
}

export interface NativeLabConnection { origin: string; unlocked: boolean; requiresPin: boolean }
export interface NativeLabFile { id: string; fileName: string; resumePositionSec: number }
export interface NativeLabMovie { id: string; code: string; title: string; actors?: string[]; files: NativeLabFile[] }
export interface NativeLabStatus {
  engineReady: boolean
  connection?: NativeLabConnection
  state: NativePlayerState
  window?: { embedded: boolean; fullscreen: boolean }
}
export interface NativeLabStart { movieId: string; fileId?: string; startSec: number }
export interface NativeLabControl { action: "pause" | "resume" | "seek" | "speed" | "volume" | "stop"; value?: number }
/** renderer 只表达受限业务动作，不传入任意媒体 URL 或 shell 命令。 */
export interface NativeLabBridge {
  getStatus(): Promise<NativeLabStatus>
  selectExecutable(): Promise<boolean>
  connect(origin: string): Promise<NativeLabConnection>
  unlock(pin: string): Promise<NativeLabConnection>
  search(query: string): Promise<NativeLabMovie[]>
  detail(movieId: string): Promise<NativeLabMovie>
  resume(input: { movieId: string; fileId?: string }): Promise<number>
  start(input: NativeLabStart): Promise<void>
  control(input: NativeLabControl): Promise<void>
  windowAction(action: "fullscreen" | "minimize" | "close"): Promise<void>
}

/** 每次播放从独立状态开始，防止上一片的诊断和时间串入新片。 */
export function emptyNativePlayerState(): NativePlayerState {
  return { status: "idle", positionSec: 0, durationSec: 0, speed: 1, volume: 100,
    droppedFrames: 0, decoderDroppedFrames: 0, codec: "", hwdec: "" }
}
