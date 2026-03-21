import type { ApiError } from "./types"

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

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api"

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

export const httpClient = {
  async get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
    const response = await fetch(buildUrl(path, params), {
      method: "GET",
      headers: { "Accept": "application/json" },
    })
    return handleResponse<T>(response)
  },

  async post<T>(path: string, body?: unknown): Promise<T> {
    const response = await fetch(buildUrl(path), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async patch<T>(path: string, body?: unknown): Promise<T> {
    const response = await fetch(buildUrl(path), {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    return handleResponse<T>(response)
  },

  async delete(path: string): Promise<void> {
    const response = await fetch(buildUrl(path), {
      method: "DELETE",
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
}
