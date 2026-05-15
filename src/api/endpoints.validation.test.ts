import { afterEach, describe, expect, it, vi } from "vitest"
import { api } from "./endpoints"
import { HttpClientError, httpClient } from "./http-client"

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  })
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe("api endpoint response validation", () => {
  it("keeps valid health responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          name: "curated-dev",
          version: "20260430.120000",
          transport: "http",
          databasePath: "runtime/curated.db",
        }),
      ),
    )

    await expect(api.health()).resolves.toMatchObject({
      name: "curated-dev",
      transport: "http",
    })
  })

  it("rejects malformed health responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          name: "curated-dev",
          transport: "http",
        }),
      ),
    )

    await expect(api.health()).rejects.toThrow("Invalid API response for GET /health")
  })

  it("rejects malformed movie list pages", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          items: { id: "movie-1" },
          total: 1,
          limit: 500,
          offset: 0,
        }),
      ),
    )

    await expect(api.listMovies({ limit: 500, offset: 0 })).rejects.toThrow(
      "Invalid API response for GET /library/movies",
    )
  })

  it("rejects malformed movie details", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          title: "Missing id",
          code: "ABC-123",
        }),
      ),
    )

    await expect(api.getMovie("movie-1")).rejects.toThrow(
      "Invalid API response for GET /library/movies/:id",
    )
  })

  it("keeps valid connected clients responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          clients: [
            {
              key: "local-chrome",
              ip: "127.0.0.1",
              browser: "Chrome",
              os: "Windows",
              deviceType: "desktop",
              accessKind: "local",
              isLocalMachine: true,
              firstSeen: "2026-05-15T10:00:00Z",
              lastSeen: "2026-05-15T10:01:00Z",
              requestCount: 2,
            },
          ],
          total: 1,
          localCount: 1,
          remoteCount: 0,
          sampledAt: "2026-05-15T10:01:00Z",
        }),
      ),
    )

    await expect(api.listConnectedClients()).resolves.toMatchObject({
      total: 1,
      clients: [expect.objectContaining({ ip: "127.0.0.1" })],
    })
  })

  it("rejects malformed connected clients responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          clients: [{ ip: "127.0.0.1" }],
          total: "1",
          sampledAt: "2026-05-15T10:01:00Z",
        }),
      ),
    )

    await expect(api.listConnectedClients()).rejects.toThrow(
      "Invalid API response for GET /connected-clients",
    )
  })

  it("calls auth status endpoint", async () => {
    const status = {
      pinEnabled: true,
      unlocked: false,
      setupRequired: false,
      pinLength: 4,
      trustedForever: false,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    }
    const get = vi.spyOn(httpClient, "get").mockResolvedValueOnce(status)

    await expect(api.authStatus()).resolves.toEqual(status)

    expect(get).toHaveBeenCalledWith("/auth/status")
  })

  it("posts trustedForever when unlocking with permanent device trust", async () => {
    const status = {
      pinEnabled: true,
      unlocked: true,
      setupRequired: false,
      pinLength: 6,
      trustedForever: true,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(status)

    await expect(api.unlockPin({ pin: "123456", trustedForever: true })).resolves.toEqual(status)

    expect(post).toHaveBeenCalledWith("/auth/unlock", {
      pin: "123456",
      trustedForever: true,
    })
  })

  it("posts current and new PIN values when changing PIN", async () => {
    const status = {
      pinEnabled: true,
      unlocked: true,
      setupRequired: false,
      pinLength: 5,
      trustedForever: false,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(status)

    await expect(api.changePin({
      currentPin: "1234",
      newPin: "98765",
      confirmPin: "98765",
    })).resolves.toEqual(status)

    expect(post).toHaveBeenCalledWith("/auth/change-pin", {
      currentPin: "1234",
      newPin: "98765",
      confirmPin: "98765",
    })
  })

  it("keeps small movie imports on the multipart endpoint", async () => {
    const task = {
      taskId: "import.movies-1",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-02T00:00:00Z",
      progress: 100,
    }
    const postForm = vi.spyOn(httpClient, "postFormWithProgress").mockResolvedValueOnce(task)
    const post = vi.spyOn(httpClient, "post")
    const putBinary = vi.spyOn(httpClient as typeof httpClient & {
      putBinaryWithProgress: typeof httpClient.postFormWithProgress
    }, "putBinaryWithProgress")

    const file = new File(["small"], "IMP-SMALL.mp4", { type: "video/mp4" })

    await expect(
      api.importMovies([file], {
        resumableThresholdBytes: 1024,
      }),
    ).resolves.toEqual(task)

    expect(postForm).toHaveBeenCalledTimes(1)
    expect(postForm).toHaveBeenCalledWith("/import/movies", expect.any(FormData), {
      onUploadProgress: undefined,
    })
    expect(post).not.toHaveBeenCalledWith("/import/movies/uploads", expect.anything())
    expect(putBinary).not.toHaveBeenCalled()
  })

  it("uses resumable chunk upload for large movie imports", async () => {
    const task = {
      taskId: "import.movies-upload-1",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-02T00:00:00Z",
      progress: 100,
    }
    const createUpload = {
      uploadId: "upload_1",
      targetPath: "D:/Library",
      chunkSize: 4,
      bytesReceived: 0,
      totalBytes: 8,
      state: "uploading",
      files: [
        {
          fileId: "file_1",
          relativePath: "IMP-LARGE.mp4",
          size: 8,
          bytesReceived: 0,
          complete: false,
        },
      ],
      task,
    }
    const post = vi.spyOn(httpClient, "post")
    post.mockResolvedValueOnce(createUpload)
    post.mockResolvedValueOnce(task)
    const putBinary = vi.spyOn(httpClient as typeof httpClient & {
      putBinaryWithProgress: (
        path: string,
        body: Blob,
        options?: {
          headers?: Record<string, string>
          diagnosticContext?: {
            uploadId?: string
            fileId?: string
            chunkIndex?: number
            offset?: number
          }
          onUploadProgress?: (progress: { loaded: number; total: number; percent: number }) => void
        },
      ) => Promise<unknown>
    }, "putBinaryWithProgress")
    putBinary.mockImplementation(async (_path, _body, options) => {
      const total = Number(options?.headers?.["X-Curated-Chunk-Size"] ?? 0)
      options?.onUploadProgress?.({ loaded: total, total, percent: 100 })
      return createUpload
    })
    const postForm = vi.spyOn(httpClient, "postFormWithProgress")
    const onUploadProgress = vi.fn()
    const file = new File(["fake-mp4"], "IMP-LARGE.mp4", { type: "video/mp4" })

    await expect(
      api.importMovies([file], {
        onUploadProgress,
        resumableThresholdBytes: 1,
      }),
    ).resolves.toEqual(task)

    expect(post).toHaveBeenNthCalledWith(1, "/import/movies/uploads", {
      files: [{ relativePath: "IMP-LARGE.mp4", size: 8, lastModified: file.lastModified }],
    })
    expect(putBinary).toHaveBeenCalledTimes(2)
    expect(putBinary).toHaveBeenNthCalledWith(
      1,
      "/import/movies/uploads/upload_1/files/file_1/chunks/0",
      expect.any(Blob),
      expect.objectContaining({
        headers: {
          "Content-Type": "application/octet-stream",
          "X-Curated-Offset": "0",
          "X-Curated-Chunk-Size": "4",
        },
        diagnosticContext: {
          uploadId: "upload_1",
          fileId: "file_1",
          chunkIndex: 0,
          offset: 0,
        },
      }),
    )
    expect(putBinary).toHaveBeenNthCalledWith(
      2,
      "/import/movies/uploads/upload_1/files/file_1/chunks/1",
      expect.any(Blob),
      expect.objectContaining({
        headers: {
          "Content-Type": "application/octet-stream",
          "X-Curated-Offset": "4",
          "X-Curated-Chunk-Size": "4",
        },
        diagnosticContext: {
          uploadId: "upload_1",
          fileId: "file_1",
          chunkIndex: 1,
          offset: 4,
        },
      }),
    )
    expect(post).toHaveBeenNthCalledWith(2, "/import/movies/uploads/upload_1/commit")
    expect(postForm).not.toHaveBeenCalled()
    expect(onUploadProgress).toHaveBeenLastCalledWith({ loaded: 8, total: 8, percent: 100 })
  })

  it("retries retryable network failures during resumable chunk upload", async () => {
    const task = {
      taskId: "import.movies-upload-retry",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-02T00:00:00Z",
      progress: 100,
    }
    const createUpload = {
      uploadId: "upload_retry",
      targetPath: "D:/Library",
      chunkSize: 4,
      bytesReceived: 0,
      totalBytes: 8,
      state: "uploading",
      files: [
        {
          fileId: "file_retry",
          relativePath: "IMP-RETRY.mp4",
          size: 8,
          bytesReceived: 0,
          complete: false,
        },
      ],
      task,
    }
    const post = vi.spyOn(httpClient, "post")
    post.mockResolvedValueOnce(createUpload)
    post.mockResolvedValueOnce(task)
    const putBinary = vi.spyOn(httpClient as typeof httpClient & {
      putBinaryWithProgress: (
        path: string,
        body: Blob,
        options?: {
          headers?: Record<string, string>
          diagnosticContext?: {
            uploadId?: string
            fileId?: string
            chunkIndex?: number
            offset?: number
          }
          onUploadProgress?: (progress: { loaded: number; total: number; percent: number }) => void
        },
      ) => Promise<unknown>
    }, "putBinaryWithProgress")
    putBinary.mockImplementation(async (path, _body, options) => {
      if (path.endsWith("/chunks/1") && putBinary.mock.calls.filter(([calledPath]) => calledPath === path).length === 1) {
        throw new HttpClientError(0, {
          code: "COMMON_NETWORK_ERROR",
          message:
            "Network request failed (uploadId=upload_retry, fileId=file_retry, chunkIndex=1, offset=4)",
          retryable: true,
        })
      }
      const total = Number(options?.headers?.["X-Curated-Chunk-Size"] ?? 0)
      options?.onUploadProgress?.({ loaded: total, total, percent: 100 })
      return createUpload
    })
    const file = new File(["fake-mp4"], "IMP-RETRY.mp4", { type: "video/mp4" })

    await expect(
      api.importMovies([file], {
        resumableThresholdBytes: 1,
        resumableChunkRetryDelayMs: 0,
      }),
    ).resolves.toEqual(task)

    expect(putBinary).toHaveBeenCalledTimes(3)
    expect(putBinary.mock.calls.map(([path]) => path)).toEqual([
      "/import/movies/uploads/upload_retry/files/file_retry/chunks/0",
      "/import/movies/uploads/upload_retry/files/file_retry/chunks/1",
      "/import/movies/uploads/upload_retry/files/file_retry/chunks/1",
    ])
    expect(post).toHaveBeenNthCalledWith(2, "/import/movies/uploads/upload_retry/commit")
  })

})
