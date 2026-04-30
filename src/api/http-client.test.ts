import { afterEach, describe, expect, it, vi } from "vitest"
import { HttpClientError, httpClient, resolveApiBaseUrl } from "./http-client"

function abortableNeverFetch() {
  return vi.fn((_url: RequestInfo | URL, init?: RequestInit) => {
    return new Promise<Response>((_resolve, reject) => {
      const signal = init?.signal
      if (!signal) return
      signal.addEventListener("abort", () => {
        const error = new Error("The operation was aborted.")
        error.name = "AbortError"
        reject(error)
      }, { once: true })
    })
  })
}

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe("resolveApiBaseUrl", () => {
  it("defaults to the same-origin API path", () => {
    expect(resolveApiBaseUrl({})).toBe("/api")
  })

  it("keeps an explicit API base URL override", () => {
    expect(resolveApiBaseUrl({ VITE_API_BASE_URL: "http://192.168.1.10:8081/api" })).toBe(
      "http://192.168.1.10:8081/api",
    )
  })
})

describe("httpClient", () => {
  it("aborts stalled requests and throws a retryable timeout error", async () => {
    vi.useFakeTimers()
    const fetchMock = abortableNeverFetch()
    vi.stubGlobal("fetch", fetchMock)

    let state:
      | { status: "resolved"; value: unknown }
      | { status: "rejected"; error: unknown }
      | undefined
    const request = httpClient.get("/slow")
    void request.then(
      (value) => {
        state = { status: "resolved", value }
      },
      (error: unknown) => {
        state = { status: "rejected", error }
      },
    )

    await vi.advanceTimersByTimeAsync(30_000)
    await Promise.resolve()

    expect(state?.status).toBe("rejected")
    if (state?.status !== "rejected") return
    expect(state.error).toBeInstanceOf(HttpClientError)
    expect(state.error).toMatchObject({
      status: 0,
      apiError: {
        code: "COMMON_TIMEOUT",
        message: "Request timed out",
        retryable: true,
      },
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const init = fetchMock.mock.calls[0]?.[1]
    expect(init?.signal).toBeInstanceOf(AbortSignal)
  })
})
