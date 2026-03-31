type HlsInstance = {
  loadSource(src: string): void
  attachMedia(video: HTMLVideoElement): void
  destroy(): void
  on?(event: string, handler: (event: string, data?: unknown) => void): void
  off?(event: string, handler: (event: string, data?: unknown) => void): void
  currentLevel?: number
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
  new (): HlsInstance
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

export function canPlayHlsNatively(video: HTMLVideoElement): boolean {
  const result = video.canPlayType("application/vnd.apple.mpegurl")
  return result === "probably" || result === "maybe"
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

export type { HlsInstance, HlsLevel }
