type HlsInstance = {
  loadSource(src: string): void
  attachMedia(video: HTMLVideoElement): void
  startLoad?(startPosition?: number, skipSeekToStartPosition?: boolean): void
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

const HLS_SCRIPT_SRC = "https://cdn.jsdelivr.net/npm/hls.js@1.6.15/dist/hls.min.js"

export function buildHlsPlaybackConfig(): Record<string, unknown> {
  return {
    // Backend HLS sessions are event-style playlists while ffmpeg is still
    // writing segments. Start at the session origin instead of hls.js' live edge.
    autoStartLoad: false,
    startPosition: 0,
    startFragPrefetch: true,
    enableWorker: true,
    lowLatencyMode: false,
    maxBufferLength: 30,
    maxMaxBufferLength: 60,
    backBufferLength: 90,
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

  hlsLoaderPromise = new Promise<HlsCtor>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>('script[data-curated-hls="true"]')
    if (existing) {
      existing.addEventListener("load", () => {
        if (window.Hls) {
          resolve(window.Hls)
        } else {
          reject(new Error("HLS library did not initialize"))
        }
      }, { once: true })
      existing.addEventListener("error", () => reject(new Error("Failed to load HLS library")), { once: true })
      return
    }

    const script = document.createElement("script")
    script.src = HLS_SCRIPT_SRC
    script.async = true
    script.dataset.curatedHls = "true"
    script.onload = () => {
      if (window.Hls) {
        resolve(window.Hls)
        return
      }
      reject(new Error("HLS library did not initialize"))
    }
    script.onerror = () => reject(new Error("Failed to load HLS library"))
    document.head.appendChild(script)
  })

  return hlsLoaderPromise
}

export function preloadHlsLibrary(): void {
  void loadHlsLibrary().catch(() => {
    // Prewarming is best-effort. Playback startup will retry if needed.
  })
}

function extractPlaylistEntries(
  playlistUrl: string,
  playlistBody: string,
  limit: number,
): {
  childPlaylistUrls: string[]
  mediaResourceUrls: string[]
} {
  const base = new URL(playlistUrl, window.location.href)
  const variantPlaylistUrls: string[] = []
  const auxiliaryPlaylistUrls: string[] = []
  const mediaResourceUrls: string[] = []
  let nextUriIsChildPlaylist = false
  const add = (target: string[], candidate: string | null | undefined) => {
    const trimmed = candidate?.trim()
    if (!trimmed) return
    try {
      const absolute = new URL(trimmed, base).href
      if (!target.includes(absolute) && target.length < limit) {
        target.push(absolute)
      }
    } catch {
      // ignore malformed lines
    }
  }

  for (const rawLine of playlistBody.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line) continue
    if (line.startsWith("#EXT-X-MAP:")) {
      const uriMatch = /URI="([^"]+)"/i.exec(line)
      add(mediaResourceUrls, uriMatch?.[1])
      continue
    }
    if (line.startsWith("#EXT-X-STREAM-INF:")) {
      nextUriIsChildPlaylist = true
      continue
    }
    if (line.startsWith("#EXT-X-I-FRAME-STREAM-INF:") || line.startsWith("#EXT-X-MEDIA:")) {
      const uriMatch = /URI="([^"]+)"/i.exec(line)
      add(auxiliaryPlaylistUrls, uriMatch?.[1])
      continue
    }
    if (line.startsWith("#")) {
      nextUriIsChildPlaylist = false
      continue
    }
    if (nextUriIsChildPlaylist) {
      add(variantPlaylistUrls, line)
      nextUriIsChildPlaylist = false
      continue
    }
    add(mediaResourceUrls, line)
    if (mediaResourceUrls.length >= limit) break
  }

  return {
    childPlaylistUrls: [...variantPlaylistUrls, ...auxiliaryPlaylistUrls].slice(0, limit),
    mediaResourceUrls: mediaResourceUrls.slice(0, limit),
  }
}

async function fetchWithTimeout(url: string, timeoutMs: number, signal?: AbortSignal): Promise<Response> {
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), timeoutMs)
  const abortForwarder = () => controller.abort()
  signal?.addEventListener("abort", abortForwarder, { once: true })
  try {
    return await fetch(url, {
      credentials: "same-origin",
      cache: "default",
      signal: controller.signal,
    })
  } finally {
    window.clearTimeout(timer)
    signal?.removeEventListener("abort", abortForwarder)
  }
}

async function collectPrewarmMediaUrls(
  playlistUrl: string,
  timeoutMs: number,
  resourceCount: number,
  signal?: AbortSignal,
  visited: Set<string> = new Set(),
  depth = 0,
): Promise<string[]> {
  if (visited.has(playlistUrl) || depth > 2) return []
  visited.add(playlistUrl)

  const playlistResponse = await fetchWithTimeout(playlistUrl, timeoutMs, signal)
  if (!playlistResponse.ok) return []
  const playlistBody = await playlistResponse.text()
  const { childPlaylistUrls, mediaResourceUrls } = extractPlaylistEntries(
    playlistUrl,
    playlistBody,
    Math.max(resourceCount, 4),
  )

  if (mediaResourceUrls.length > 0) {
    return mediaResourceUrls.slice(0, resourceCount)
  }
  if (childPlaylistUrls.length === 0) {
    return []
  }

  const resolvedMediaUrls: string[] = []
  for (const childPlaylistUrl of childPlaylistUrls) {
    const childMediaUrls = await collectPrewarmMediaUrls(
      childPlaylistUrl,
      timeoutMs,
      resourceCount,
      signal,
      visited,
      depth + 1,
    )
    for (const mediaUrl of childMediaUrls) {
      if (!resolvedMediaUrls.includes(mediaUrl)) {
        resolvedMediaUrls.push(mediaUrl)
      }
      if (resolvedMediaUrls.length >= resourceCount) {
        return resolvedMediaUrls
      }
    }
  }

  return resolvedMediaUrls
}

export async function prewarmHlsResources(
  playlistUrl: string,
  options?: {
    resourceCount?: number
    timeoutMs?: number
    signal?: AbortSignal
    onProgress?: (progress: number) => void
  },
): Promise<void> {
  if (typeof window === "undefined" || typeof fetch !== "function") return
  const timeoutMs = Math.max(500, options?.timeoutMs ?? 3200)
  const resourceCount = Math.max(1, Math.min(3, options?.resourceCount ?? 2))
  const reportProgress = (progress: number) => {
    options?.onProgress?.(Math.max(0, Math.min(1, progress)))
  }

  reportProgress(0.08)
  const resourceUrls = await collectPrewarmMediaUrls(
    playlistUrl,
    timeoutMs,
    resourceCount,
    options?.signal,
  )
  if (resourceUrls.length === 0) {
    reportProgress(1)
    return
  }

  reportProgress(0.32)
  let completedCount = 0
  await Promise.allSettled(
    resourceUrls.map(async (resourceUrl) => {
      try {
        const response = await fetchWithTimeout(resourceUrl, timeoutMs, options?.signal)
        if (!response.ok) return
        await response.arrayBuffer()
      } finally {
        completedCount += 1
        reportProgress(0.32 + (completedCount / resourceUrls.length) * 0.68)
      }
    }),
  )
  reportProgress(1)
}

export type { HlsInstance, HlsLevel }
