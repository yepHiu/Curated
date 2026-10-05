import type { NativeLabControl, NativeLabMovie, NativePlayerState } from "./native-player-contract.js"

export interface DesktopPlaybackPreferences { preferNative: boolean; volume: number; speed: number }
export interface DesktopPlaybackCapabilities {
  protocol: 1
  available: boolean
  reason?: "platform" | "engine" | "host" | "page"
  preferences: DesktopPlaybackPreferences
}
export type DesktopPlaybackSourceQuery = Record<string, string | null | (string | null)[]>
export interface DesktopPlaybackOpen {
  movieId: string
  fileId?: string
  startSec?: number
  seekExisting?: boolean
  autoplay: boolean
  locale?: "zh-CN" | "en-US" | "ja-JP"
  queue?: string[]
  autoAdvance?: boolean
  sourceQuery?: DesktopPlaybackSourceQuery
}
export interface DesktopPlaybackSnapshot {
  captureRevision?: number
  sessionId: string
  revision: number
  engine: "native"
  windowOpen: boolean
  movie?: NativeLabMovie
  state: NativePlayerState
  fullscreen: boolean
  maximized?: boolean
  queue: string[]
  autoAdvance: boolean
  locale: "zh-CN" | "en-US" | "ja-JP"
  sourceQuery?: DesktopPlaybackSourceQuery
}
export interface DesktopPlaybackCapture {
  id: string
  movieId: string
  fileId: string
  code: string
  positionSec: number
  capturedAt: string
  preview: string
  phase: "saved" | "error"
  error?: string
}
export interface DesktopPlayerBridge extends Pick<DesktopPlaybackBridge, "snapshot" | "command" | "subscribe"> {
  capture(sessionId: string, retryId?: string): Promise<DesktopPlaybackCapture>
  capturePreferences(): Promise<{ keyCode: string; feedbackSoundEnabled: boolean }>
}
export type DesktopPlaybackCommand =
  | NativeLabControl
  | { [Action in "focus" | "fullscreen" | "minimize" | "close" | "web" | "replay"]: { action: Action } }["focus" | "fullscreen" | "minimize" | "close" | "web" | "replay"]
  | { action: "part"; fileId: string }
  | { action: "movie"; movieId: string }
  | { action: "autoAdvance"; enabled: boolean }
export interface DesktopPlaybackBridge {
  capabilities(): Promise<DesktopPlaybackCapabilities>
  setPreferences(input: Pick<DesktopPlaybackPreferences, "preferNative">): Promise<DesktopPlaybackCapabilities>
  open(input: DesktopPlaybackOpen): Promise<DesktopPlaybackSnapshot>
  snapshot(): Promise<DesktopPlaybackSnapshot>
  command(sessionId: string, input: DesktopPlaybackCommand): Promise<void>
  subscribe(callback: (snapshot: DesktopPlaybackSnapshot) => void): () => void
  onWebFallback(callback: (input: { movieId: string; fileId?: string; startSec: number }) => void): () => void
}
