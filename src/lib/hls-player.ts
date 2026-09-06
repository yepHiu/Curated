type HlsInstance = {
  loadSource(src: string): void
  attachMedia(video: HTMLVideoElement): void
  startLoad?(startPosition?: number, skipSeekToStartPosition?: boolean): void
  recoverMediaError?(): void
  destroy(): void
  on?(event: string, handler: (event: string, data?: unknown) => void): void
  off?(event: string, handler: (event: string, data?: unknown) => void): void
  currentLevel?: number
  loadLevel?: number
  nextLevel?: number
  nextLoadLevel?: number
  autoLevelEnabled?: boolean
  bandwidthEstimate?: number
  levels?: HlsLevel[]
}

type HlsLevel = {
  bitrate?: number
  width?: number
  height?: number
  frameRate?: number | string
  attrs?: Record<string, unknown>
}

type HlsCtor = {
  new (config?: Record<string, unknown>): HlsInstance
  isSupported(): boolean
  Events?: Record<string, string>
}

declare global {
  interface Window {
    Hls?: HlsCtor
  }
}

let hlsLoaderPromise: Promise<HlsCtor> | null = null

export function buildHlsPlaybackConfig(): Record<string, unknown> {
  return {
    // Backend HLS sessions are event-style playlists while ffmpeg is still
    // writing segments. Start at the session origin instead of hls.js' live edge.
    autoStartLoad: false,
    startPosition: 0,
    startFragPrefetch: true,
    enableWorker: true,
    lowLatencyMode: false,
    maxBufferLength: 60,
    maxMaxBufferLength: 120,
    backBufferLength: 90,
    // Event playlists stay "live" while ffmpeg is still writing. Prefer waiting
    // for the next transcoded fragment over giving up and restarting playback.
    maxStarvationDelay: 12,
    xhrSetup(xhr: XMLHttpRequest) {
      xhr.withCredentials = true
    },
  }
}

export function startHlsLoadingAtSessionOrigin(player: Pick<HlsInstance, "startLoad">): void {
  player.startLoad?.(0)
}

export function canPlayHlsNatively(video: HTMLVideoElement): boolean {
  const ua = typeof navigator !== "undefined" ? navigator.userAgent : ""
  const vendor = typeof navigator !== "undefined" ? navigator.vendor : ""
  const canPlay = video.canPlayType("application/vnd.apple.mpegurl")
  const isApplePlatform = /iPad|iPhone|iPod|Macintosh/i.test(ua)
  const isSafariEngine = /Apple/i.test(vendor) && /Safari/i.test(ua)
  const isChromiumFamily = /Chrome|Chromium|Edg|OPR|Brave/i.test(ua)
  const isFirefox = /Firefox/i.test(ua)

  // Desktop Chrome/Edge may report HLS support loosely, but playback is not reliable there.
  if (!isApplePlatform || !isSafariEngine || isChromiumFamily || isFirefox) {
    return false
  }
  return canPlay === "probably" || canPlay === "maybe"
}

export async function loadHlsLibrary(): Promise<HlsCtor> {
  if (window.Hls) {
    return window.Hls
  }
  if (hlsLoaderPromise) {
    return hlsLoaderPromise
  }

  // Curated's backend emits a single local playback rendition and does not use
  // subtitle, EME/DRM, alternate-audio, or CMCD controllers from the full build.
  hlsLoaderPromise = import("hls.js/light")
    .then((mod) => {
      const Hls = (mod.default ?? mod) as HlsCtor
      window.Hls = Hls
      return Hls
    })
    .catch((error) => {
      hlsLoaderPromise = null
      throw error
    })

  return hlsLoaderPromise
}

export function preloadHlsLibrary(): void {
  void loadHlsLibrary().catch(() => {
    // Prewarming is best-effort. Playback startup will retry if needed.
  })
}

export type { HlsInstance, HlsLevel }
