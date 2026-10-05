import type { NativeLabControl, NativeLabMovie, NativePlayerState } from "./native-player-contract.js"

export interface DesktopPlaybackPreferences { preferNative: boolean; volume: number; speed: number }
export interface DesktopPlaybackCapabilities {
  protocol: 1
  available: boolean
  reason?: "platform" | "engine" | "host" | "page"
  preferences: DesktopPlaybackPreferences
}
export interface DesktopPlaybackOpen {
  movieId: string
  fileId?: string
  startSec?: number
  autoplay: boolean
  locale?: "zh-CN" | "en-US" | "ja-JP"
  queue?: string[]
  autoAdvance?: boolean
}
export interface DesktopPlaybackSnapshot {
  sessionId: string
  revision: number
  engine: "native"
  windowOpen: boolean
  movie?: NativeLabMovie
  state: NativePlayerState
  fullscreen: boolean
  queue: string[]
  autoAdvance: boolean
  locale: "zh-CN" | "en-US" | "ja-JP"
}
export type DesktopPlaybackCommand =
  | NativeLabControl
  | { [Action in "focus" | "fullscreen" | "minimize" | "close" | "web"]: { action: Action } }["focus" | "fullscreen" | "minimize" | "close" | "web"]
  | { action: "part"; fileId: string }
  | { action: "movie"; movieId: string }
  | { action: "autoAdvance"; enabled: boolean }
export interface DesktopPlaybackBridge {
  capabilities(): Promise<DesktopPlaybackCapabilities>
  setPreferences(input: DesktopPlaybackPreferences): Promise<DesktopPlaybackCapabilities>
  open(input: DesktopPlaybackOpen): Promise<DesktopPlaybackSnapshot>
  snapshot(): Promise<DesktopPlaybackSnapshot>
  command(sessionId: string, input: DesktopPlaybackCommand): Promise<void>
  subscribe(callback: (snapshot: DesktopPlaybackSnapshot) => void): () => void
  onWebFallback(callback: (input: { movieId: string; fileId?: string; startSec: number }) => void): () => void
}
