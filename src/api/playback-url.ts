type PlaybackUrlEnv = Pick<ImportMetaEnv, "VITE_API_BASE_URL" | "VITE_USE_WEB_API"> & {
  DEV?: boolean
}

const DEFAULT_DEV_BACKEND_PORT = "8080"

function trimApiBase(base: string): string {
  return base.trim().replace(/\/$/, "")
}

function isLoopbackHost(hostname: string): boolean {
  const normalized = hostname.replace(/^\[|\]$/g, "").toLowerCase()
  return normalized === "localhost" || normalized === "127.0.0.1" || normalized === "::1"
}

function formatUrlHost(hostname: string): string {
  const normalized = hostname.replace(/^\[|\]$/g, "")
  if (normalized === "::1") return "[::1]"
  return normalized || "127.0.0.1"
}

function resolvePlaybackApiBaseUrl(env: PlaybackUrlEnv, origin: string): string {
  const explicit = env.VITE_API_BASE_URL?.trim()
  if (explicit) return trimApiBase(explicit)

  const current = new URL(origin)
  if (env.DEV === true && env.VITE_USE_WEB_API === "true" && isLoopbackHost(current.hostname)) {
    return `${current.protocol}//${formatUrlHost(current.hostname)}:${DEFAULT_DEV_BACKEND_PORT}/api`
  }

  return "/api"
}

function buildPlaybackApiUrl(
  path: string,
  options: {
    env?: PlaybackUrlEnv
    origin?: string
  } = {},
): string {
  const base = resolvePlaybackApiBaseUrl(
    options.env ?? import.meta.env,
    options.origin ?? window.location.origin,
  )
  const tail = path.startsWith("/") ? path : `/${path}`
  if (base.startsWith("http://") || base.startsWith("https://")) {
    return `${base}${tail}`
  }
  return new URL(`${base}${tail}`, options.origin ?? window.location.origin).href
}

function isPlaybackMediaPath(pathname: string, movieId: string): boolean {
  const directMovieStreamPath = `/api/library/movies/${encodeURIComponent(movieId)}/stream`
  return pathname === directMovieStreamPath || pathname.startsWith("/api/playback/sessions/")
}

export function buildMoviePlaybackUrl(
  movieId: string,
  options: {
    env?: PlaybackUrlEnv
    origin?: string
  } = {},
): string {
  return buildPlaybackApiUrl(`/library/movies/${encodeURIComponent(movieId)}/stream`, options)
}

export function resolveMoviePlaybackSourceUrl(
  movieId: string,
  descriptorUrl: string | null | undefined,
  options: {
    env?: PlaybackUrlEnv
    origin?: string
  } = {},
): string | null {
  const rawUrl = descriptorUrl?.trim()
  if (!rawUrl) return buildMoviePlaybackUrl(movieId, options)

  const origin = options.origin ?? window.location.origin
  const current = new URL(origin)
  const resolved = new URL(rawUrl, origin)
  if (resolved.origin === current.origin && isPlaybackMediaPath(resolved.pathname, movieId)) {
    const apiPath = `${resolved.pathname.replace(/^\/api/, "")}${resolved.search}`
    return buildPlaybackApiUrl(apiPath, options)
  }
  return resolved.href
}

/**
 * Absolute URL for GET /api/library/movies/{id}/stream.
 * In local Web API development it bypasses Vite's proxy for large media streams.
 */
export function moviePlaybackAbsoluteUrl(movieId: string): string {
  return buildMoviePlaybackUrl(movieId)
}
