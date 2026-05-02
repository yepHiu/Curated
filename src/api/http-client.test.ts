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

  it("bypasses the Vite proxy for loopback Web API dev requests", () => {
    expect(
      resolveApiBaseUrl(
        {
          DEV: true,
          VITE_USE_WEB_API: "true",
        },
        "http://127.0.0.1:5173",
      ),
    ).toBe("http://127.0.0.1:8080/api")
  })

  it("keeps same-origin API path for non-loopback Web API dev requests", () => {
    expect(
      resolveApiBaseUrl(
        {
          DEV: true,
          VITE_USE_WEB_API: "true",
        },
        "http://192.168.1.20:5173",
      ),
    ).toBe("/api")
  })

  it("keeps same-origin API path for release hosting on port 8081", () => {
    expect(
      resolveApiBaseUrl(
        {
          DEV: false,
          VITE_USE_WEB_API: "true",
        },
        "http://127.0.0.1:8081",
      ),
    ).toBe("/api")
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

  it("posts multipart forms with upload progress", async () => {
    let progressHandler: ((event: ProgressEvent) => void) | undefined

    class FakeXMLHttpRequest {
      status = 0
      responseText = ""
      timeout = 0
      upload = {
        addEventListener: vi.fn((event: string, handler: EventListenerOrEventListenerObject) => {
          if (event === "progress" && typeof handler === "function") {
            progressHandler = handler as (event: ProgressEvent) => void
          }
        }),
      }
      open = vi.fn()
      setRequestHeader = vi.fn()
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      ontimeout: (() => void) | null = null
      send = vi.fn(() => {
        progressHandler?.({
          lengthComputable: true,
          loaded: 5,
          total: 10,
        } as ProgressEvent)
        this.status = 202
        this.responseText = JSON.stringify({ ok: true })
        this.onload?.()
      })
    }

    vi.stubGlobal("XMLHttpRequest", FakeXMLHttpRequest)

    const form = new FormData()
    form.set("files", new Blob(["movie"]), "IMP-001.mp4")
    const onUploadProgress = vi.fn()

    await expect(
      httpClient.postFormWithProgress<{ ok: boolean }>("/import/movies", form, {
        onUploadProgress,
      }),
    ).resolves.toEqual({ ok: true })
    expect(onUploadProgress).toHaveBeenCalledWith({
      loaded: 5,
      total: 10,
      percent: 50,
    })
  })

  it("puts binary bodies with custom headers and upload progress", async () => {
    let progressHandler: ((event: ProgressEvent) => void) | undefined
    const xhrInstances: FakeXMLHttpRequest[] = []

    class FakeXMLHttpRequest {
      constructor() {
        xhrInstances.push(this)
      }
      status = 0
      responseText = ""
      timeout = 0
      upload = {
        addEventListener: vi.fn((event: string, handler: EventListenerOrEventListenerObject) => {
          if (event === "progress" && typeof handler === "function") {
            progressHandler = handler as (event: ProgressEvent) => void
          }
        }),
      }
      open = vi.fn()
      setRequestHeader = vi.fn()
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      ontimeout: (() => void) | null = null
      send = vi.fn(() => {
        progressHandler?.({
          lengthComputable: true,
          loaded: 4,
          total: 8,
        } as ProgressEvent)
        this.status = 200
        this.responseText = JSON.stringify({ ok: true })
        this.onload?.()
      })
    }

    vi.stubGlobal("XMLHttpRequest", FakeXMLHttpRequest)

    const onUploadProgress = vi.fn()
    await expect(
      httpClient.putBinaryWithProgress<{ ok: boolean }>("/import/movies/uploads/u/files/f/chunks/0", new Blob(["chunk"]), {
        headers: {
          "Content-Type": "application/octet-stream",
          "X-Curated-Offset": "0",
        },
        onUploadProgress,
      }),
    ).resolves.toEqual({ ok: true })

    expect(xhrInstances[0]?.open).toHaveBeenCalledWith(
      "PUT",
      expect.stringContaining("/api/import/movies/uploads/u/files/f/chunks/0"),
    )
    expect(xhrInstances[0]?.setRequestHeader).toHaveBeenCalledWith("Accept", "application/json")
    expect(xhrInstances[0]?.setRequestHeader).toHaveBeenCalledWith(
      "Content-Type",
      "application/octet-stream",
    )
    expect(xhrInstances[0]?.setRequestHeader).toHaveBeenCalledWith("X-Curated-Offset", "0")
    expect(onUploadProgress).toHaveBeenCalledWith({
      loaded: 4,
      total: 8,
      percent: 50,
    })
  })

  it("adds upload chunk diagnostics to binary network errors", async () => {
    class FakeXMLHttpRequest {
      status = 0
      responseText = ""
      timeout = 0
      upload = {
        addEventListener: vi.fn(),
      }
      open = vi.fn()
      setRequestHeader = vi.fn()
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      ontimeout: (() => void) | null = null
      send = vi.fn(() => {
        this.onerror?.()
      })
    }

    vi.stubGlobal("XMLHttpRequest", FakeXMLHttpRequest)

    await expect(
      httpClient.putBinaryWithProgress<{ ok: boolean }>(
        "/import/movies/uploads/upload_abc/files/file_def/chunks/7",
        new Blob(["chunk"]),
        {
          headers: {
            "Content-Type": "application/octet-stream",
            "X-Curated-Offset": "224",
            "X-Curated-Chunk-Size": "32",
          },
          diagnosticContext: {
            uploadId: "upload_abc",
            fileId: "file_def",
            chunkIndex: 7,
            offset: 224,
          },
        },
      ),
    ).rejects.toMatchObject({
      apiError: {
        code: "COMMON_NETWORK_ERROR",
        message:
          "Network request failed (uploadId=upload_abc, fileId=file_def, chunkIndex=7, offset=224)",
        retryable: true,
      },
    })
  })

  it("adds upload chunk diagnostics to binary timeout errors", async () => {
    class FakeXMLHttpRequest {
      status = 0
      responseText = ""
      timeout = 0
      upload = {
        addEventListener: vi.fn(),
      }
      open = vi.fn()
      setRequestHeader = vi.fn()
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      ontimeout: (() => void) | null = null
      send = vi.fn(() => {
        this.ontimeout?.()
      })
    }

    vi.stubGlobal("XMLHttpRequest", FakeXMLHttpRequest)

    await expect(
      httpClient.putBinaryWithProgress<{ ok: boolean }>(
        "/import/movies/uploads/upload_timeout/files/file_timeout/chunks/3",
        new Blob(["chunk"]),
        {
          timeoutMs: 1,
          diagnosticContext: {
            uploadId: "upload_timeout",
            fileId: "file_timeout",
            chunkIndex: 3,
            offset: 96,
          },
        },
      ),
    ).rejects.toMatchObject({
      apiError: {
        code: "COMMON_TIMEOUT",
        message:
          "Request timed out (uploadId=upload_timeout, fileId=file_timeout, chunkIndex=3, offset=96)",
        retryable: true,
      },
    })
  })
})
