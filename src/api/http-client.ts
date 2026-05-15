import type { ApiError } from "./types"
import { devRequestMonitor } from "@/lib/dev-performance/request-monitor"

export class HttpClientError extends Error {
  readonly status: number
  readonly apiError?: ApiError

  constructor(status: number, apiError?: ApiError) {
    super(apiError?.message ?? `HTTP ${status}`)
    this.name = "HttpClientError"
    this.status = status
    this.apiError = apiError
  }

  get retryable(): boolean {
    return this.apiError?.retryable ?? this.status >= 500
  }
}

export interface UploadProgress {
  loaded: number
  total: number
  percent: number
}

type ApiBaseUrlEnv = Pick<ImportMetaEnv, "VITE_API_BASE_URL" | "VITE_USE_WEB_API"> & {
  DEV?: boolean
}

interface UploadDiagnosticContext {
  uploadId?: string
  fileId?: string
  chunkIndex?: number
  offset?: number
}

const DEFAULT_DEV_BACKEND_PORT = "8080"

/** In local Web API development, bypass Vite's proxy; otherwise use same-origin /api. */
export function resolveApiBaseUrl(
  env: ApiBaseUrlEnv,
  origin = window.location.origin,
): string {
  const explicit = env.VITE_API_BASE_URL?.trim()
  if (explicit) {
    return trimApiBaseUrl(explicit)
  }

  const current = new URL(origin)
  if (env.DEV === true && env.VITE_USE_WEB_API === "true" && isLoopbackHost(current.hostname)) {
    return `${current.protocol}//${formatUrlHost(current.hostname)}:${DEFAULT_DEV_BACKEND_PORT}/api`
  }

  return "/api"
}

const BASE_URL = resolveApiBaseUrl(import.meta.env)
const DEV_REQUEST_MONITOR_ENABLED = import.meta.env.DEV
const DEFAULT_TIMEOUT_MS = 30_000

function trimApiBaseUrl(base: string): string {
  return base.replace(/\/$/, "")
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

function buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
  const url = new URL(`${BASE_URL}${path}`, window.location.origin)

  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== "") {
        url.searchParams.set(key, String(value))
      }
    }
  }

  return url.toString()
}

function buildMonitorPath(path: string, params?: Record<string, string | number | undefined>): string {
  const url = new URL(buildUrl(path, params), window.location.origin)
  return `${url.pathname}${url.search}`
}

async function monitoredFetch(
  method: string,
  path: string,
  init: RequestInit,
  params?: Record<string, string | number | undefined>,
): Promise<Response> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS)
  const requestId = DEV_REQUEST_MONITOR_ENABLED
    ? devRequestMonitor.startRequest({
      method,
      path: buildMonitorPath(path, params),
    })
    : null

  try {
    const response = await fetch(buildUrl(path, params), {
      method,
      ...init,
      signal: controller.signal,
      credentials: "include",
    })
    if (requestId) {
      devRequestMonitor.finishRequest(requestId, {
        status: response.status,
      })
    }
    return response
  } catch (error) {
    if (requestId) {
      devRequestMonitor.finishRequest(requestId, {
        status: null,
        failed: true,
      })
    }
    if (error instanceof Error && error.name === "AbortError") {
      throw new HttpClientError(0, {
        code: "COMMON_TIMEOUT",
        message: "Request timed out",
        retryable: true,
      })
    }
    throw error
  } finally {
    clearTimeout(timeoutId)
  }
}

async function parseJsonBody<T>(response: Response): Promise<T> {
  const text = await response.text()
  if (!text.trim()) {
    return undefined as T
  }
  return JSON.parse(text) as T
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let apiError: ApiError | undefined
    try {
      apiError = await parseJsonBody<ApiError>(response)
    } catch {
      // response body was not JSON
    }
    throw new HttpClientError(response.status, apiError)
  }
  return parseJsonBody<T>(response)
}

async function handleErrorResponse(response: Response): Promise<never> {
  let apiError: ApiError | undefined
  try {
    const text = await response.text()
    if (text.trim()) {
      apiError = JSON.parse(text) as ApiError
    }
  } catch {
    // ignore
  }
  throw new HttpClientError(response.status, apiError)
}

function parseXHRJsonBody<T>(text: string): T {
  if (!text.trim()) {
    return undefined as T
  }
  return JSON.parse(text) as T
}

function formatUploadDiagnosticContext(context?: UploadDiagnosticContext): string {
  if (!context) {
    return ""
  }

  const parts: string[] = []
  if (context.uploadId) {
    parts.push(`uploadId=${context.uploadId}`)
  }
  if (context.fileId) {
    parts.push(`fileId=${context.fileId}`)
  }
  if (context.chunkIndex !== undefined) {
    parts.push(`chunkIndex=${context.chunkIndex}`)
  }
  if (context.offset !== undefined) {
    parts.push(`offset=${context.offset}`)
  }

  return parts.length > 0 ? ` (${parts.join(", ")})` : ""
}

export const httpClient = {
  async get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
    const response = await monitoredFetch("GET", path, {
      headers: { "Accept": "application/json" },
    }, params)
    return handleResponse<T>(response)
  },

  async post<T>(path: string, body?: unknown): Promise<T> {
    const response = await monitoredFetch("POST", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async postForm<T>(path: string, body: FormData): Promise<T> {
    const response = await monitoredFetch("POST", path, {
      headers: { "Accept": "application/json" },
      body,
    })
    return handleResponse<T>(response)
  },

  async postFormWithProgress<T>(
    path: string,
    body: FormData,
    options: {
      onUploadProgress?: (progress: UploadProgress) => void
      timeoutMs?: number
    } = {},
  ): Promise<T> {
    return await new Promise<T>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open("POST", buildUrl(path))
      xhr.withCredentials = true
      xhr.setRequestHeader("Accept", "application/json")
      xhr.timeout = options.timeoutMs ?? 0

      xhr.upload.addEventListener("progress", (event) => {
        if (!event.lengthComputable || event.total <= 0) {
          return
        }
        const percent = Math.min(100, Math.max(0, Math.round((event.loaded / event.total) * 100)))
        options.onUploadProgress?.({
          loaded: event.loaded,
          total: event.total,
          percent,
        })
      })

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            resolve(parseXHRJsonBody<T>(xhr.responseText))
          } catch (error) {
            reject(error)
          }
          return
        }

        let apiError: ApiError | undefined
        try {
          apiError = parseXHRJsonBody<ApiError>(xhr.responseText)
        } catch {
          // response body was not JSON
        }
        reject(new HttpClientError(xhr.status, apiError))
      }

      xhr.onerror = () => {
        reject(new HttpClientError(0, {
          code: "COMMON_NETWORK_ERROR",
          message: "Network request failed",
          retryable: true,
        }))
      }

      xhr.ontimeout = () => {
        reject(new HttpClientError(0, {
          code: "COMMON_TIMEOUT",
          message: "Request timed out",
          retryable: true,
        }))
      }

      xhr.send(body)
    })
  },

  async putBinaryWithProgress<T>(
    path: string,
    body: Blob,
    options: {
      headers?: Record<string, string>
      onUploadProgress?: (progress: UploadProgress) => void
      timeoutMs?: number
      diagnosticContext?: UploadDiagnosticContext
    } = {},
  ): Promise<T> {
    return await new Promise<T>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      const diagnosticSuffix = formatUploadDiagnosticContext(options.diagnosticContext)
      xhr.open("PUT", buildUrl(path))
      xhr.withCredentials = true
      xhr.setRequestHeader("Accept", "application/json")
      for (const [key, value] of Object.entries(options.headers ?? {})) {
        xhr.setRequestHeader(key, value)
      }
      xhr.timeout = options.timeoutMs ?? 0

      xhr.upload.addEventListener("progress", (event) => {
        if (!event.lengthComputable || event.total <= 0) {
          return
        }
        const percent = Math.min(100, Math.max(0, Math.round((event.loaded / event.total) * 100)))
        options.onUploadProgress?.({
          loaded: event.loaded,
          total: event.total,
          percent,
        })
      })

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            resolve(parseXHRJsonBody<T>(xhr.responseText))
          } catch (error) {
            reject(error)
          }
          return
        }

        let apiError: ApiError | undefined
        try {
          apiError = parseXHRJsonBody<ApiError>(xhr.responseText)
        } catch {
          // response body was not JSON
        }
        reject(new HttpClientError(xhr.status, apiError))
      }

      xhr.onerror = () => {
        reject(new HttpClientError(0, {
          code: "COMMON_NETWORK_ERROR",
          message: `Network request failed${diagnosticSuffix}`,
          retryable: true,
        }))
      }

      xhr.ontimeout = () => {
        reject(new HttpClientError(0, {
          code: "COMMON_TIMEOUT",
          message: `Request timed out${diagnosticSuffix}`,
          retryable: true,
        }))
      }

      xhr.send(body)
    })
  },

  async patch<T>(path: string, body?: unknown): Promise<T> {
    const response = await monitoredFetch("PATCH", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async put<T = void>(path: string, body?: unknown): Promise<T> {
    const response = await monitoredFetch("PUT", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async delete<T = void>(path: string): Promise<T> {
    const response = await monitoredFetch("DELETE", path, {
      headers: { "Accept": "application/json" },
    })
    return handleResponse<T>(response)
  },

  async postBlob(
    path: string,
    body?: unknown,
  ): Promise<{ blob: Blob; contentDisposition: string | null }> {
    const response = await monitoredFetch("POST", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "image/webp,application/zip,application/json;q=0.1,*/*;q=0.05",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    if (!response.ok) {
      await handleErrorResponse(response)
    }
    const blob = await response.blob()
    return {
      blob,
      contentDisposition: response.headers.get("Content-Disposition"),
    }
  },
}
