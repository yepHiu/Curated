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

  it("keeps valid versioned Saved Views and rejects malformed filter schemas", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(
          jsonResponse({
            items: [
              {
                id: "view-1",
                name: "Unwatched 4K",
                filters: {
                  schemaVersion: 1,
                  mode: "library",
                  playState: "unwatched",
                  resolution: "4k",
                },
                sortOrder: 0,
                createdAt: "2026-07-20T00:00:00Z",
                updatedAt: "2026-07-20T00:00:00Z",
              },
            ],
          }),
        )
        .mockResolvedValueOnce(
          jsonResponse({
            items: [
              {
                id: "view-2",
                name: "Future schema",
                filters: { schemaVersion: 2 },
                sortOrder: 0,
                createdAt: "2026-07-20T00:00:00Z",
                updatedAt: "2026-07-20T00:00:00Z",
              },
            ],
          }),
        ),
    )

    await expect(api.listSavedViews()).resolves.toMatchObject({
      items: [expect.objectContaining({ id: "view-1" })],
    })
    await expect(api.listSavedViews()).rejects.toThrow(
      "Invalid API response for GET /library/saved-views",
    )
  })

  it("validates explained recommendations and feedback responses", async () => {
    const snapshot = {
      dateUtc: "2026-07-21",
      generatedAt: "2026-07-21T00:00:00Z",
      generationVersion: "v7",
      heroMovieIds: ["m01"],
      recommendationMovieIds: ["m02"],
      recommendations: [{
        movieId: "m02",
        reasons: [{ code: "well_rated" }],
        feedbackEffects: [],
      }],
    }
    const feedback = {
      id: "feedback_1",
      action: "less",
      targetType: "actor",
      targetValue: "Actor A",
      sourceMovieId: "m02",
      createdAt: "2026-07-21T00:00:00Z",
      updatedAt: "2026-07-21T00:00:00Z",
    }
    vi.spyOn(httpClient, "get")
      .mockResolvedValueOnce(snapshot)
      .mockResolvedValueOnce({ ...snapshot, recommendationMovieIds: ["m03"] })
      .mockResolvedValueOnce({ items: [feedback] })
    const post = vi.spyOn(httpClient, "post")
      .mockResolvedValueOnce(snapshot)
      .mockResolvedValueOnce(feedback)
    const remove = vi.spyOn(httpClient, "delete").mockResolvedValueOnce(undefined)

    await expect(api.getHomepageDailyRecommendations()).resolves.toEqual(snapshot)
    await expect(api.getHomepageDailyRecommendations()).rejects.toThrow(
      "Invalid API response for GET /homepage/recommendations",
    )
    await expect(api.refreshHomepageDailyRecommendations()).resolves.toEqual(snapshot)
    await expect(api.listHomepageRecommendationFeedback()).resolves.toEqual({ items: [feedback] })
    await expect(api.createHomepageRecommendationFeedback({
      action: "less",
      targetType: "actor",
      targetValue: "Actor A",
      sourceMovieId: "m02",
    })).resolves.toEqual(feedback)
    await expect(api.deleteHomepageRecommendationFeedback("feedback/1")).resolves.toBeUndefined()
    expect(remove).toHaveBeenCalledWith("/homepage/recommendations/feedback/feedback%2F1")
    expect(post).toHaveBeenNthCalledWith(1, "/homepage/recommendations/refresh", undefined)
  })

  it("lists and revokes trusted sessions through safe public ids", async () => {
    const sessions = {
      items: [{
        publicId: "public-session-id",
        userAgent: "Test Browser",
        createdAt: "2026-07-19T10:00:00Z",
        lastSeenAt: "2026-07-19T11:00:00Z",
        trustedForever: true,
        current: false,
      }],
    }
    const get = vi.spyOn(httpClient, "get").mockResolvedValueOnce(sessions)
    const remove = vi.spyOn(httpClient, "delete").mockResolvedValueOnce({ items: [] })
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce({ items: [] })

    await expect(api.listTrustedAuthSessions()).resolves.toEqual(sessions)
    await expect(api.revokeTrustedAuthSession("public/session")).resolves.toEqual({ items: [] })
    await expect(api.revokeOtherTrustedAuthSessions()).resolves.toEqual({ items: [] })

    expect(get).toHaveBeenCalledWith("/auth/sessions")
    expect(remove).toHaveBeenCalledWith("/auth/sessions/public%2Fsession")
    expect(post).toHaveBeenCalledWith("/auth/sessions/revoke-others", {})
  })

  it("validates backup create verify and preflight responses", async () => {
    const manifest = {
      format: "curated-backup",
      formatVersion: 1,
      createdAt: "2026-07-20T02:00:00Z",
      appVersion: "1.4.11",
      appChannel: "dev",
      scope: {
        databaseIncluded: true,
        libraryConfigIncluded: true,
        userAssetsIncluded: false,
        mediaFilesIncluded: false,
      },
      schemaMigrations: ["0001_init.sql"],
      files: [
        {
          kind: "database",
          path: "database/curated.db",
          sizeBytes: 1024,
          sha256: "a".repeat(64),
        },
      ],
    }
    const verification = {
      valid: true,
      checkedAt: "2026-07-20T02:01:00Z",
      manifest,
      databaseIntegrity: { quickCheck: "ok", foreignKeyViolations: 0 },
      errors: [],
      warnings: [],
    }
    const preflight = {
      canRestore: true,
      checkedAt: "2026-07-20T02:02:00Z",
      verification,
      targetDatabase: "D:\\Curated\\curated.db",
      targetDatabaseExists: true,
      targetConfig: "D:\\Curated\\library-config.cfg",
      targetConfigExists: true,
      requiredBytes: 2048,
      availableBytes: 4096,
      availableBytesKnown: true,
      unsupportedMigrations: [],
      errors: [],
      warnings: [],
    }
    const post = vi.spyOn(httpClient, "post")
    post.mockResolvedValueOnce(manifest)
    post.mockResolvedValueOnce(verification)
    post.mockResolvedValueOnce(preflight)

    await expect(api.createBackup({ destinationPath: "D:\\Backups\\curated.curated-backup" })).resolves.toEqual(manifest)
    await expect(api.verifyBackup({ backupPath: "D:\\Backups\\curated.curated-backup" })).resolves.toEqual(verification)
    await expect(api.preflightBackupRestore({ backupPath: "D:\\Backups\\curated.curated-backup" })).resolves.toEqual(preflight)

    expect(post).toHaveBeenNthCalledWith(1, "/maintenance/backups", {
      destinationPath: "D:\\Backups\\curated.curated-backup",
    })
    expect(post).toHaveBeenNthCalledWith(2, "/maintenance/backups/verify", {
      backupPath: "D:\\Backups\\curated.curated-backup",
    })
    expect(post).toHaveBeenNthCalledWith(3, "/maintenance/backups/preflight", {
      backupPath: "D:\\Backups\\curated.curated-backup",
    })
  })

  it("rejects malformed backup verification responses", async () => {
    vi.spyOn(httpClient, "post").mockResolvedValueOnce({
      valid: true,
      checkedAt: "2026-07-20T02:01:00Z",
      databaseIntegrity: { quickCheck: "ok" },
      errors: [],
      warnings: [],
    })

    await expect(api.verifyBackup({ backupPath: "D:\\bad.curated-backup" })).rejects.toThrow(
      "Invalid API response for POST /maintenance/backups/verify",
    )
  })

  it("validates library health reports and persisted repair results", async () => {
    const report = {
      scannedAt: "2026-07-20T10:00:00Z",
      status: "attention",
      database: {
        quickCheckOk: true,
        quickCheckMessages: ["ok"],
        foreignKeyOk: true,
        foreignKeyCount: 0,
      },
      storageStatuses: [],
      summary: {
        totalFindings: 1,
        criticalFindings: 0,
        warningFindings: 1,
        infoFindings: 0,
        skippedOfflineFiles: 0,
        categoryCounts: { metadata_missing: 1 },
      },
      findings: [{
        id: "health-1",
        category: "metadata_missing",
        severity: "warning",
        entityType: "movie",
        entityId: "movie-1",
        label: "TEST-001",
        message: "metadata missing",
        repairActions: ["rescrape_metadata"],
      }],
      truncated: false,
    }
    const repair = {
      repairId: "repair-1",
      taskId: "task-1",
      action: "rescrape_metadata",
      categories: ["metadata_missing"],
      status: "completed",
      totalItems: 1,
      completedItems: 1,
      succeededItems: 1,
      failedItems: 0,
      createdAt: "2026-07-20T10:01:00Z",
      finishedAt: "2026-07-20T10:01:02Z",
      items: [{
        ordinal: 0,
        findingId: "health-1",
        category: "metadata_missing",
        movieId: "movie-1",
        label: "TEST-001",
        status: "succeeded",
        childTaskId: "scrape-1",
      }],
    }
    const post = vi.spyOn(httpClient, "post")
    post.mockResolvedValueOnce(report)
    post.mockResolvedValueOnce(repair)
    const get = vi.spyOn(httpClient, "get").mockResolvedValueOnce(repair)

    await expect(api.scanLibraryHealth(250)).resolves.toEqual(report)
    await expect(api.startLibraryHealthRepair({
      action: "rescrape_metadata",
      categories: ["metadata_missing"],
      confirm: true,
    })).resolves.toEqual(repair)
    await expect(api.getLibraryHealthRepair("repair-1")).resolves.toEqual(repair)

    expect(post).toHaveBeenNthCalledWith(1, "/library/health/scan?findingLimit=250")
    expect(get).toHaveBeenCalledWith("/library/health/repairs/repair-1")
  })

  it("posts exact confirmed library health cleanup findings", async () => {
    const task = {
      taskId: "cleanup-1",
      type: "library.health.cleanup",
      status: "running",
      createdAt: "2026-07-20T10:02:00Z",
      progress: 0,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(task)

    await expect(api.startLibraryHealthAction({
      action: "cleanup_orphan_state",
      findingIds: ["health-orphan"],
      confirm: true,
    })).resolves.toEqual(task)
    expect(post).toHaveBeenCalledWith("/library/health/actions", {
      action: "cleanup_orphan_state",
      findingIds: ["health-orphan"],
      confirm: true,
    })
  })

  it("rejects malformed library health category counts", async () => {
    vi.spyOn(httpClient, "post").mockResolvedValueOnce({
      scannedAt: "2026-07-20T10:00:00Z",
      status: "healthy",
      database: {
        quickCheckOk: true,
        quickCheckMessages: ["ok"],
        foreignKeyOk: true,
        foreignKeyCount: 0,
      },
      storageStatuses: [],
      summary: {
        totalFindings: 0,
        criticalFindings: 0,
        warningFindings: 0,
        infoFindings: 0,
        skippedOfflineFiles: 0,
        categoryCounts: { metadata_missing: "zero" },
      },
      findings: [],
      truncated: false,
    })

    await expect(api.scanLibraryHealth()).rejects.toThrow(
      "Invalid API response for POST /library/health/scan",
    )
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

  it("resumes an interrupted upload and only sends missing chunks", async () => {
    const task = {
      taskId: "import.movies-upload-resume",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-02T00:00:00Z",
      progress: 100,
    }
    const sessionStatus = {
      uploadId: "upload_resume",
      targetPath: "D:/Library",
      chunkSize: 4,
      bytesReceived: 4,
      totalBytes: 8,
      state: "uploading",
      files: [
        {
          fileId: "file_resume",
          relativePath: "IMP-RESUME.mp4",
          size: 8,
          bytesReceived: 4,
          complete: false,
          chunks: [{ index: 0, offset: 0, size: 4 }],
        },
      ],
      task,
    }
    const get = vi.spyOn(httpClient, "get")
    get.mockResolvedValueOnce(sessionStatus)
    const post = vi.spyOn(httpClient, "post")
    post.mockResolvedValueOnce(task)
    const putBinary = vi.spyOn(httpClient as typeof httpClient & {
      putBinaryWithProgress: (
        path: string,
        body: Blob,
        options?: {
          headers?: Record<string, string>
          diagnosticContext?: Record<string, unknown>
          onUploadProgress?: (progress: { loaded: number; total: number; percent: number }) => void
        },
      ) => Promise<unknown>
    }, "putBinaryWithProgress")
    putBinary.mockImplementation(async (_path, _body, options) => {
      const total = Number(options?.headers?.["X-Curated-Chunk-Size"] ?? 0)
      options?.onUploadProgress?.({ loaded: total, total, percent: 100 })
      return sessionStatus
    })
    const onUploadProgress = vi.fn()
    const file = new File(["fake-mp4"], "IMP-RESUME.mp4", { type: "video/mp4" })

    await expect(
      api.importMovies([file], { onUploadProgress, resumeUploadId: "upload_resume" }),
    ).resolves.toEqual(task)

    expect(get).toHaveBeenCalledWith("/import/movies/uploads/upload_resume")
    expect(putBinary).toHaveBeenCalledTimes(1)
    expect(putBinary).toHaveBeenCalledWith(
      "/import/movies/uploads/upload_resume/files/file_resume/chunks/1",
      expect.any(Blob),
      expect.objectContaining({
        headers: {
          "Content-Type": "application/octet-stream",
          "X-Curated-Offset": "4",
          "X-Curated-Chunk-Size": "4",
        },
      }),
    )
    expect(post).toHaveBeenCalledTimes(1)
    expect(post).toHaveBeenCalledWith("/import/movies/uploads/upload_resume/commit")
    // 进度基线从服务端已接收的 4 字节起算
    expect(onUploadProgress).toHaveBeenLastCalledWith({ loaded: 8, total: 8, percent: 100 })
  })

  it("rejects resuming a session that is no longer uploading", async () => {
    const get = vi.spyOn(httpClient, "get")
    get.mockResolvedValueOnce({
      uploadId: "upload_stale",
      targetPath: "D:/Library",
      chunkSize: 4,
      bytesReceived: 4,
      totalBytes: 8,
      state: "expired",
      files: [],
      task: {},
    })
    const post = vi.spyOn(httpClient, "post")
    const file = new File(["fake-mp4"], "IMP-STALE.mp4", { type: "video/mp4" })

    await expect(
      api.importMovies([file], { resumeUploadId: "upload_stale" }),
    ).rejects.toThrow("Upload session upload_stale is no longer accepting chunks")
    expect(post).not.toHaveBeenCalled()
  })

  it("validates actor merge preview, apply audit, and audit list responses", async () => {
    const association = {
      sourceCount: 1,
      targetCount: 2,
      duplicateCount: 0,
      resultCount: 3,
    }
    const preview = {
      previewToken: "token",
      source: { id: 1, name: "Source", aliases: [] },
      target: { id: 2, name: "Target", aliases: ["Target Alias"] },
      movies: association,
      userTags: { source: [], target: [], result: [] },
      externalLinks: { source: [], target: [], result: [] },
      recommendationFeedback: { ...association, targetCount: 0, resultCount: 1 },
      curatedFramesAffected: 0,
      aliasesToMove: ["Source"],
      profileFields: [
        {
          field: "summary",
          sourceValue: "source",
          targetValue: "target",
          defaultSelection: "target",
          conflict: true,
        },
      ],
      canApply: true,
      blockingReasons: [],
      requiredDecisions: ["summary"],
    }
    const audit = {
      id: "amrg_1",
      sourceActorId: 1,
      targetActorId: 2,
      sourceName: "Source",
      targetName: "Target",
      previewToken: "token",
      appliedAt: "2026-07-21T00:00:00Z",
      summary: {
        movies: association,
        userTags: [],
        externalLinks: [],
        aliases: ["Source"],
        recommendationFeedback: { ...association, targetCount: 0, resultCount: 1 },
        curatedFramesAffected: 0,
        profileDecisions: { summary: "target" },
      },
    }
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(jsonResponse(preview))
        .mockResolvedValueOnce(jsonResponse(audit))
        .mockResolvedValueOnce(
          jsonResponse({ items: [audit], total: 1, limit: 50, offset: 0 }),
        ),
    )

    await expect(
      api.previewActorMerge({ sourceName: "Source", targetName: "Target" }),
    ).resolves.toMatchObject({ previewToken: "token" })
    await expect(
      api.applyActorMerge({
        sourceName: "Source",
        targetName: "Target",
        previewToken: "token",
        confirm: true,
        profileDecisions: { summary: "target" },
      }),
    ).resolves.toMatchObject({ id: "amrg_1" })
    await expect(api.listActorMergeAudits()).resolves.toMatchObject({ total: 1 })
  })

  it("rejects malformed actor merge responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        jsonResponse({
          previewToken: "token",
          source: { id: 1, name: "Source", aliases: [] },
          target: { id: 2, name: "Target", aliases: [] },
          movies: { sourceCount: 1 },
        }),
      ),
    )

    await expect(
      api.previewActorMerge({ sourceName: "Source", targetName: "Target" }),
    ).rejects.toThrow("Invalid API response for POST /library/actors/merge-preview")
  })

  it("validates personal insights responses and encodes IANA timezone queries", async () => {
    const overview = {
      range: "30d",
      from: "2026-06-23",
      to: "2026-07-22",
      timezone: "Asia/Shanghai",
      generatedAt: "2026-07-21T16:30:00Z",
      dataSince: null,
      watchedSeconds: 0,
      startedMovies: 0,
      completedMovies: 0,
      completionRate: null,
      completionThreshold: 0.9,
      ratedMovies: 0,
      averageUserRating: null,
    }
    const breakdown = {
      range: "30d",
      dimension: "actor",
      from: "2026-06-23",
      to: "2026-07-22",
      timezone: "Asia/Shanghai",
      generatedAt: "2026-07-21T16:30:00Z",
      dataSince: "2026-07-01",
      totalWatchedSeconds: 120,
      attribution: "full-per-entity",
      items: [{ name: "Actor", watchedSeconds: 120, movieCount: 1, shareOfTotal: 1 }],
      limit: 10,
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(overview))
      .mockResolvedValueOnce(jsonResponse(breakdown))
    vi.stubGlobal("fetch", fetchMock)

    await expect(api.getPersonalInsightsOverview({ range: "30d", timezone: "Asia/Shanghai" }))
      .resolves.toEqual(overview)
    await expect(api.getPersonalInsightsBreakdown({
      range: "30d",
      timezone: "Asia/Shanghai",
      dimension: "actor",
      limit: 10,
    })).resolves.toEqual(breakdown)

    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("range=30d&timezone=Asia%2FShanghai")
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain("dimension=actor&limit=10")
  })

  it("rejects misleading or malformed personal insights responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(jsonResponse({
          range: "30d",
          from: "2026-06-23",
          to: "2026-07-22",
          timezone: "UTC",
          generatedAt: "2026-07-21T16:30:00Z",
          dataSince: null,
          watchedSeconds: 0,
          startedMovies: 0,
          completedMovies: 0,
          completionRate: 0,
          completionThreshold: 0.9,
          ratedMovies: 0,
          averageUserRating: null,
        }))
        .mockResolvedValueOnce(jsonResponse({
          range: "30d",
          dimension: "tag",
          from: "2026-06-23",
          to: "2026-07-22",
          timezone: "UTC",
          generatedAt: "2026-07-21T16:30:00Z",
          dataSince: null,
          totalWatchedSeconds: 0,
          attribution: "exclusive",
          items: [],
          limit: 10,
        })),
    )

    await expect(api.getPersonalInsightsOverview({ range: "30d", timezone: "UTC" }))
      .rejects.toThrow("Invalid API response for GET /insights/overview")
    await expect(api.getPersonalInsightsBreakdown({
      range: "30d",
      timezone: "UTC",
      dimension: "tag",
    })).rejects.toThrow("Invalid API response for GET /insights/breakdown")
  })

})
