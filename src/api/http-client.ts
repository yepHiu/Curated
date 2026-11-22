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

interface RequestOptions {
  /** Milliseconds before aborting; 0 disables the deadline for long-running operations. */
  timeoutMs?: number
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

/** 将响应正文消费纳入同一次请求的超时、取消和性能记录生命周期。 */
async function monitoredRequest<T>(
  method: string,
  path: string,
  init: RequestInit,
  consume: (response: Response) => Promise<T>,
  params?: Record<string, string | number | undefined>,
  options?: RequestOptions,
): Promise<T> {
  const controller = new AbortController()
  const externalSignal = init.signal
  // 调用方取消应一直转发到正文读取结束。
  const abort = () => controller.abort()
  if (externalSignal?.aborted) abort()
  externalSignal?.addEventListener("abort", abort, { once: true })
  const timeoutMs = options?.timeoutMs ?? DEFAULT_TIMEOUT_MS
  const timeoutId = timeoutMs > 0 ? setTimeout(() => {
    // 同一截止时间覆盖响应头及完整正文。
    controller.abort()
  }, timeoutMs) : undefined
  const requestId = DEV_REQUEST_MONITOR_ENABLED
    ? devRequestMonitor.startRequest({
      method,
      path: buildMonitorPath(path, params),
    })
    : null

  let status: number | null = null
  try {
    const response = await fetch(buildUrl(path, params), {
      method,
      ...init,
      signal: controller.signal,
      credentials: "include",
    })
    status = response.status
    const result = await consume(response)
    if (requestId) {
      devRequestMonitor.finishRequest(requestId, {
        status: response.status,
      })
    }
    return result
  } catch (error) {
    if (requestId) {
      devRequestMonitor.finishRequest(requestId, {
        status,
        failed: true,
      })
    }
    // 以本次控制器状态区分超时和主动取消，兼容跨 realm 的 DOMException。
    if (controller.signal.aborted) {
      if (externalSignal?.aborted) throw error
      throw new HttpClientError(0, {
        code: "COMMON_TIMEOUT",
        message: "Request timed out",
        retryable: true,
      })
    }
    throw error
  } finally {
    clearTimeout(timeoutId)
    externalSignal?.removeEventListener("abort", abort)
  }
}

/** 读取完整 JSON 正文；兼容无正文的成功响应。 */
async function parseJsonBody<T>(response: Response): Promise<T> {
  const text = await response.text()
  if (!text.trim()) {
    return undefined as T
  }
  return JSON.parse(text) as T
}

/** 在请求生命周期内解析成功响应或服务端错误。 */
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) await handleErrorResponse(response)
  return parseJsonBody<T>(response)
}

/** 容忍非 JSON 错误页，但保留正文读取的取消和网络异常。 */
async function handleErrorResponse(response: Response): Promise<never> {
  let apiError: ApiError | undefined
  try {
    const text = await response.text()
    if (text.trim()) {
      apiError = JSON.parse(text) as ApiError
    }
  } catch (error) {
    if (!(error instanceof SyntaxError)) throw error
  }
  throw new HttpClientError(response.status, apiError)
}

/** 有界读取字幕正文；声明超限时立即释放未读取的响应体。 */
async function handleTextResponse(response: Response): Promise<string> {
  if (!response.ok) await handleErrorResponse(response)
  const length = Number(response.headers.get("Content-Length"))
  if (length > 16 * 1024 * 1024) {
    await response.body?.cancel()
    throw new Error("Subtitle file exceeds size limit")
  }
  const text = await response.text()
  if (text.length > 16 * 1024 * 1024) throw new Error("Subtitle file exceeds size limit")
  return text
}

/** 在请求截止时间内完成二进制下载，并保留下载文件名头。 */
async function handleBlobResponse(response: Response): Promise<{ blob: Blob; contentDisposition: string | null }> {
  if (!response.ok) await handleErrorResponse(response)
  return { blob: await response.blob(), contentDisposition: response.headers.get("Content-Disposition") }
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
  /** 获取完整字幕文本。 */
  async getText(path: string): Promise<string> {
    return monitoredRequest("GET", path, { headers: { Accept: "text/plain" } }, handleTextResponse)
  },
  /** 获取 JSON；调用方可取消响应头或正文读取。 */
  async get<T>(path: string, params?: Record<string, string | number | undefined>, signal?: AbortSignal): Promise<T> {
    return monitoredRequest("GET", path, {
      headers: { "Accept": "application/json" },
      signal,
    }, handleResponse<T>, params)
  },

  /** 提交 JSON，允许显式调整长任务的整次请求期限。 */
  async post<T>(path: string, body?: unknown, signal?: AbortSignal, options?: RequestOptions): Promise<T> {
    return monitoredRequest("POST", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal,
    }, handleResponse<T>, undefined, options)
  },

  /** 提交 multipart 并读取 JSON 结果。 */
  async postForm<T>(path: string, body: FormData): Promise<T> {
    return monitoredRequest("POST", path, {
      headers: { "Accept": "application/json" },
      body,
    }, handleResponse<T>)
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

  /** 局部更新资源并读取 JSON 结果。 */
  async patch<T>(path: string, body?: unknown): Promise<T> {
    return monitoredRequest("PATCH", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }, handleResponse<T>)
  },

  /** 替换资源并兼容无正文的成功响应。 */
  async put<T = void>(path: string, body?: unknown): Promise<T> {
    return monitoredRequest("PUT", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }, handleResponse<T>)
  },

  /** 删除资源并等待响应读取完成。 */
  async delete<T = void>(path: string): Promise<T> {
    return monitoredRequest("DELETE", path, {
      headers: { "Accept": "application/json" },
    }, handleResponse<T>)
  },

  /** 请求导出资源；下载正文同样受请求期限约束。 */
  async postBlob(
    path: string,
    body?: unknown,
  ): Promise<{ blob: Blob; contentDisposition: string | null }> {
    return monitoredRequest("POST", path, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "image/webp,application/zip,application/json;q=0.1,*/*;q=0.05",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }, handleBlobResponse)
  },
}
