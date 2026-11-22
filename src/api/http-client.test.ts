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

/** 返回真实流式 Response，模拟已收响应头但正文仍在传输的 fetch。 */
function streamingFetch(status = 200) {
  let signal: AbortSignal | null | undefined
  let streamController: ReadableStreamDefaultController<Uint8Array>
  const fetch = vi.fn(async (_url: RequestInfo | URL, init?: RequestInit) => {
    // 模拟浏览器在请求取消时同时中止响应体读取。
    signal = init?.signal
    return new Response(new ReadableStream<Uint8Array>({
      /** 保持正文打开，交由测试显式完成或取消。 */
      start(controller) {
        streamController = controller
        signal?.addEventListener("abort", () => {
          // 与原生 fetch 的正文读取错误保持一致。
          controller.error(new DOMException("Aborted", "AbortError"))
        }, { once: true })
      },
    }), { status })
  })
  return {
    fetch,
    /** 暴露传输 signal，验证取消真正到达网络层。 */
    signal: () => signal,
    /** 完成正文传输，验证完成后的计时器回收。 */
    finish(text: string) {
      streamController.enqueue(new TextEncoder().encode(text))
      streamController.close()
    },
  }
}

describe("httpClient", () => {
  // 不同正文类型及错误响应都必须受相同截止时间保护。
  it.each(["json", "text", "blob", "error"])("times out a stalled %s body after headers arrive", async kind => {
    vi.useFakeTimers()
    const transport = streamingFetch(kind === "error" ? 500 : 200)
    vi.stubGlobal("fetch", transport.fetch)
    const request = kind === "text" ? httpClient.getText("/subtitles")
      : kind === "blob" ? httpClient.postBlob("/export") : httpClient.get("/slow-body")
    const result = expect(request).rejects.toMatchObject({ status: 0, apiError: { code: "COMMON_TIMEOUT", retryable: true } })
    await vi.advanceTimersByTimeAsync(30_000)
    await result
    expect(transport.signal()?.aborted).toBe(true)
  })

  // 关闭请求期限的长任务仍须允许用户在正文阶段取消。
  it.each([200, 500])("forwards cancellation while reading an HTTP %i body with no deadline", async status => {
    vi.useFakeTimers()
    const transport = streamingFetch(status)
    vi.stubGlobal("fetch", transport.fetch)
    const caller = new AbortController()
    const request = httpClient.post("/slow-body", {}, caller.signal, { timeoutMs: 0 })
    const result = expect(request).rejects.toMatchObject({ name: "AbortError" })
    await vi.advanceTimersByTimeAsync(60_000)
    expect(transport.signal()?.aborted).toBe(false)
    caller.abort()
    await result
    expect(transport.signal()?.aborted).toBe(true)
  })

  // 成功后的迟到取消不能再作用于已经结束的传输。
  it("releases the deadline and cancellation listener after consuming the body", async () => {
    vi.useFakeTimers()
    const transport = streamingFetch()
    vi.stubGlobal("fetch", transport.fetch)
    const caller = new AbortController()
    const request = httpClient.get("/complete", undefined, caller.signal)
    await vi.advanceTimersByTimeAsync(0)
    transport.finish('{"ok":true}')
    await expect(request).resolves.toEqual({ ok: true })
    caller.abort()
    await vi.advanceTimersByTimeAsync(30_000)
    expect(transport.signal()?.aborted).toBe(false)
  })

  // 非 JSON 服务端错误页保持 HTTP 状态；不能把解析错误当作网络故障。
  it("preserves the status of a non-JSON error response", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => {
      // 模拟代理返回的 HTML 错误页。
      return new Response("<html>unavailable</html>", { status: 502 })
    }))
    await expect(httpClient.get("/failed")).rejects.toMatchObject({ status: 502 })
  })

  it("waits for a long-running POST when its deadline is disabled", async () => {
    vi.useFakeTimers()
    let signal: AbortSignal | null | undefined
    vi.stubGlobal("fetch", vi.fn((_url: RequestInfo | URL, init?: RequestInit) => {
      signal = init?.signal
      return new Promise<Response>((resolve, reject) => {
        signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true })
        setTimeout(() => resolve(new Response(JSON.stringify({ ok: true }))), 120_000)
      })
    }))

    const request = httpClient.post("/slow-backup", {}, undefined, { timeoutMs: 0 })
    const result = expect(request).resolves.toEqual({ ok: true })
    await vi.advanceTimersByTimeAsync(120_000)
    await result
    expect(signal?.aborted).toBe(false)
  })

  it("still supports explicit cancellation when the POST deadline is disabled", async () => {
    vi.useFakeTimers()
    vi.stubGlobal("fetch", abortableNeverFetch())
    const controller = new AbortController()
    const request = httpClient.post("/slow-backup", {}, controller.signal, { timeoutMs: 0 })
    const result = expect(request).rejects.toMatchObject({ name: "AbortError" })
    await vi.advanceTimersByTimeAsync(60_000)
    controller.abort()
    await result
  })

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
