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

/** 开发走 Vite 代理 /api；生产构建未设置 VITE_API_BASE_URL 时对齐 release 后端默认 :8081 */
const BASE_URL =
  import.meta.env.VITE_API_BASE_URL ??
  (import.meta.env.DEV ? "/api" : "http://127.0.0.1:8081/api")
const DEV_REQUEST_MONITOR_ENABLED = import.meta.env.DEV

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
    throw error
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

  async delete(path: string): Promise<void> {
    const response = await monitoredFetch("DELETE", path, {
      headers: { "Accept": "application/json" },
    })
    if (!response.ok) {
      let apiError: ApiError | undefined
      try {
        apiError = await response.json()
      } catch {
        // response body was not JSON
      }
      throw new HttpClientError(response.status, apiError)
    }
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
