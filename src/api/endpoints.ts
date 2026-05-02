import { HttpClientError, httpClient } from "./http-client"
import {
  assertApiResponse,
  isHealthDTO,
  isMovieDetailDTO,
  isMoviesPageDTO,
} from "./guards"

function filenameFromContentDisposition(h: string | null): string {
  if (!h) {
    return "curated-export.webp"
  }
  const star = /filename\*=UTF-8''([^;\s]+)/i.exec(h)
  if (star?.[1]) {
    try {
      return decodeURIComponent(star[1])
    } catch {
      return star[1]
    }
  }
  const q = /filename="([^"]+)"/i.exec(h)
  if (q?.[1]) {
    return q[1]
  }
  return "curated-export.webp"
}
import type {
  ActorListItemDTO,
  ActorProfileDTO,
  ActorsListDTO,
  AddLibraryPathBody,
  AddLibraryPathResultDTO,
  AddPlaybackWatchTimeBody,
  AppUpdateStatusDTO,
  CreateCuratedFrameBody,
  CreateMovieImportUploadBody,
  CuratedFrameFacetListDTO,
  CuratedFrameStatsDTO,
  CreatePlaybackSessionBody,
  CuratedFramesListDTO,
  DevPerformanceSummaryDTO,
  HealthDTO,
  HomepageDailyRecommendationsDTO,
  RefreshHomepageDailyRecommendationsBody,
  LibraryPathDTO,
  UpdateLibraryPathBody,
  ListActorsParams,
  ListCuratedFramesParams,
  ListMoviesParams,
  MetadataRefreshQueuedDTO,
  MovieImportUploadProgress,
  NativePlaybackLaunchDTO,
  MetadataScrapeByPathsBody,
  MovieImportUploadDTO,
  MovieCommentDTO,
  MovieDetailDTO,
  MoviesPageDTO,
  PlaybackDescriptorDTO,
  PatchActorExternalLinksBody,
  PatchCuratedFrameTagsBody,
  PostCuratedFramesExportBody,
  PatchMovieBody,
  PatchSettingsBody,
  PlayedMoviesListDTO,
  PlaybackProgressListDTO,
  PlaybackWatchTimeDailyListDTO,
  ProxyJavBusPingRequestBody,
  ProxyJavBusPingResponse,
  PutMovieCommentBody,
  PutPlaybackProgressBody,
  SettingsDTO,
  StartScanBody,
  RecentTasksDTO,
  TaskDTO,
} from "./types"

const DEFAULT_RESUMABLE_IMPORT_THRESHOLD_BYTES = 512 * 1024 * 1024
const DEFAULT_UPLOAD_CHUNK_MAX_ATTEMPTS = 3
const DEFAULT_UPLOAD_CHUNK_RETRY_DELAY_MS = 500

interface MovieImportApiOptions {
  onUploadProgress?: (progress: MovieImportUploadProgress) => void
  resumableThresholdBytes?: number
  resumableChunkMaxAttempts?: number
  resumableChunkRetryDelayMs?: number
}

function relativePathForFile(file: File): string {
  const candidate = (file as File & { webkitRelativePath?: string }).webkitRelativePath
  return candidate?.trim() || file.name
}

function shouldUseResumableImport(
  files: File[],
  thresholdBytes = DEFAULT_RESUMABLE_IMPORT_THRESHOLD_BYTES,
): boolean {
  return thresholdBytes > 0 && files.some((file) => file.size >= thresholdBytes)
}

function movieImportUploadManifest(files: File[]): CreateMovieImportUploadBody {
  return {
    files: files.map((file) => ({
      relativePath: relativePathForFile(file),
      size: file.size,
      lastModified: file.lastModified,
    })),
  }
}

function findUploadFileSource(files: File[], relativePath: string, fallbackIndex: number): File {
  const fallback = files[fallbackIndex]
  const matched = files.find((file) => relativePathForFile(file) === relativePath)
  if (matched) return matched
  if (fallback) return fallback
  throw new Error(`Missing source file for upload path ${relativePath}`)
}

function isRetryableUploadError(error: unknown): boolean {
  return error instanceof HttpClientError && error.retryable
}

function waitForUploadRetry(delayMs: number): Promise<void> {
  if (delayMs <= 0) {
    return Promise.resolve()
  }
  return new Promise((resolve) => window.setTimeout(resolve, delayMs))
}

async function putMovieImportChunkWithRetry(
  path: string,
  chunk: Blob,
  options: Parameters<typeof httpClient.putBinaryWithProgress<MovieImportUploadDTO>>[2],
  retryOptions: {
    maxAttempts: number
    retryDelayMs: number
  },
): Promise<void> {
  const maxAttempts = Math.max(1, retryOptions.maxAttempts)
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      await httpClient.putBinaryWithProgress<MovieImportUploadDTO>(path, chunk, options)
      return
    } catch (error) {
      if (attempt >= maxAttempts || !isRetryableUploadError(error)) {
        throw error
      }
      await waitForUploadRetry(retryOptions.retryDelayMs * attempt)
    }
  }
}

async function uploadMovieFileChunks(
  upload: MovieImportUploadDTO,
  files: File[],
  options: {
    onUploadProgress?: (progress: MovieImportUploadProgress) => void
    chunkMaxAttempts?: number
    chunkRetryDelayMs?: number
  } = {},
): Promise<void> {
  const chunkSize = upload.chunkSize > 0 ? upload.chunkSize : 32 * 1024 * 1024
  let completedBytes = 0
  const chunkRetryOptions = {
    maxAttempts: options.chunkMaxAttempts ?? DEFAULT_UPLOAD_CHUNK_MAX_ATTEMPTS,
    retryDelayMs: options.chunkRetryDelayMs ?? DEFAULT_UPLOAD_CHUNK_RETRY_DELAY_MS,
  }

  for (let fileIndex = 0; fileIndex < upload.files.length; fileIndex += 1) {
    const uploadFile = upload.files[fileIndex]
    const sourceFile = findUploadFileSource(files, uploadFile.relativePath, fileIndex)
    for (let offset = 0, chunkIndex = 0; offset < sourceFile.size; chunkIndex += 1) {
      const end = Math.min(offset + chunkSize, sourceFile.size)
      const chunk = sourceFile.slice(offset, end)
      const chunkBytes = end - offset
      const chunkOffset = offset
      await putMovieImportChunkWithRetry(
        `/import/movies/uploads/${encodeURIComponent(upload.uploadId)}/files/${encodeURIComponent(uploadFile.fileId)}/chunks/${chunkIndex}`,
        chunk,
        {
          headers: {
            "Content-Type": "application/octet-stream",
            "X-Curated-Offset": String(chunkOffset),
            "X-Curated-Chunk-Size": String(chunkBytes),
          },
          diagnosticContext: {
            uploadId: upload.uploadId,
            fileId: uploadFile.fileId,
            chunkIndex,
            offset: chunkOffset,
          },
          onUploadProgress: (progress) => {
            const loaded = Math.min(upload.totalBytes, completedBytes + progress.loaded)
            options.onUploadProgress?.({
              loaded,
              total: upload.totalBytes,
              percent:
                upload.totalBytes > 0
                  ? Math.min(100, Math.max(0, Math.round((loaded / upload.totalBytes) * 100)))
                  : 0,
            })
          },
        },
        chunkRetryOptions,
      )
      completedBytes += chunkBytes
      offset = end
      options.onUploadProgress?.({
        loaded: Math.min(upload.totalBytes, completedBytes),
        total: upload.totalBytes,
        percent:
          upload.totalBytes > 0
            ? Math.min(100, Math.max(0, Math.round((completedBytes / upload.totalBytes) * 100)))
            : 0,
      })
    }
  }
}

export const api = {
  health(): Promise<HealthDTO> {
    return httpClient
      .get<unknown>("/health")
      .then((value) => assertApiResponse("GET /health", value, isHealthDTO))
  },

  getDevPerformanceSummary(): Promise<DevPerformanceSummaryDTO> {
    return httpClient.get<DevPerformanceSummaryDTO>("/dev/performance")
  },

  getAppUpdateStatus(): Promise<AppUpdateStatusDTO> {
    return httpClient.get<AppUpdateStatusDTO>("/app-update/status")
  },

  checkAppUpdateNow(): Promise<AppUpdateStatusDTO> {
    return httpClient.post<AppUpdateStatusDTO>("/app-update/check")
  },

  listPlayedMovies(): Promise<PlayedMoviesListDTO> {
    return httpClient.get<PlayedMoviesListDTO>("/library/played-movies")
  },

  getHomepageDailyRecommendations(): Promise<HomepageDailyRecommendationsDTO> {
    return httpClient.get<HomepageDailyRecommendationsDTO>("/homepage/recommendations")
  },

  refreshHomepageDailyRecommendations(
    body?: RefreshHomepageDailyRecommendationsBody,
  ): Promise<HomepageDailyRecommendationsDTO> {
    return httpClient.post<HomepageDailyRecommendationsDTO>("/homepage/recommendations/refresh", body)
  },

  recordPlayedMovie(movieId: string): Promise<void> {
    return httpClient.post<void>(`/library/played-movies/${encodeURIComponent(movieId)}`)
  },

  listMovies(params?: ListMoviesParams): Promise<MoviesPageDTO> {
    return httpClient
      .get<unknown>("/library/movies", params as Record<string, string | number | undefined>)
      .then((value) => assertApiResponse("GET /library/movies", value, isMoviesPageDTO))
  },

  getActorProfile(name: string): Promise<ActorProfileDTO> {
    return httpClient.get<ActorProfileDTO>("/library/actors/profile", { name })
  },

  listActors(params?: ListActorsParams): Promise<ActorsListDTO> {
    return httpClient.get<ActorsListDTO>("/library/actors", params as Record<string, string | number | undefined>)
  },

  patchActorUserTags(name: string, userTags: string[]): Promise<ActorListItemDTO> {
    const q = new URLSearchParams({ name })
    return httpClient.patch<ActorListItemDTO>(`/library/actors/tags?${q.toString()}`, { userTags })
  },

  patchActorExternalLinks(name: string, externalLinks: string[]): Promise<ActorProfileDTO> {
    const q = new URLSearchParams({ name })
    const body: PatchActorExternalLinksBody = { externalLinks }
    return httpClient.patch<ActorProfileDTO>(`/library/actors/external-links?${q.toString()}`, body)
  },

  scrapeActorProfile(name: string): Promise<TaskDTO> {
    const q = new URLSearchParams({ name })
    return httpClient.post<TaskDTO>(`/library/actors/scrape?${q.toString()}`)
  },

  getMovie(movieId: string): Promise<MovieDetailDTO> {
    return httpClient
      .get<unknown>(`/library/movies/${encodeURIComponent(movieId)}`)
      .then((value) => assertApiResponse("GET /library/movies/:id", value, isMovieDetailDTO))
  },

  getMoviePlayback(movieId: string): Promise<PlaybackDescriptorDTO> {
    return httpClient.get<PlaybackDescriptorDTO>(`/library/movies/${encodeURIComponent(movieId)}/playback`)
  },

  launchNativePlayback(movieId: string, startPositionSec?: number): Promise<NativePlaybackLaunchDTO> {
    return httpClient.post<NativePlaybackLaunchDTO>(
      `/library/movies/${encodeURIComponent(movieId)}/native-play`,
      startPositionSec !== undefined ? { startPositionSec } : {},
    )
  },

  createPlaybackSession(movieId: string, body: CreatePlaybackSessionBody): Promise<PlaybackDescriptorDTO> {
    return httpClient.post<PlaybackDescriptorDTO>(
      `/library/movies/${encodeURIComponent(movieId)}/playback-session`,
      body,
    )
  },

  deletePlaybackSession(sessionId: string): Promise<void> {
    return httpClient.delete(`/playback/sessions/${encodeURIComponent(sessionId)}`)
  },

  getMovieComment(movieId: string): Promise<MovieCommentDTO> {
    return httpClient.get<MovieCommentDTO>(`/library/movies/${encodeURIComponent(movieId)}/comment`)
  },

  putMovieComment(movieId: string, body: PutMovieCommentBody): Promise<MovieCommentDTO> {
    return httpClient.put<MovieCommentDTO>(`/library/movies/${encodeURIComponent(movieId)}/comment`, body)
  },

  patchMovie(movieId: string, body: PatchMovieBody): Promise<MovieDetailDTO> {
    return httpClient
      .patch<unknown>(`/library/movies/${encodeURIComponent(movieId)}`, body)
      .then((value) => assertApiResponse("PATCH /library/movies/:id", value, isMovieDetailDTO))
  },

  deleteMovie(movieId: string, opts?: { permanent?: boolean }): Promise<void> {
    const q =
      opts?.permanent === true ? `?${new URLSearchParams({ permanent: "true" }).toString()}` : ""
    return httpClient.delete(`/library/movies/${encodeURIComponent(movieId)}${q}`)
  },

  restoreMovie(movieId: string): Promise<void> {
    return httpClient.post<void>(`/library/movies/${encodeURIComponent(movieId)}/restore`)
  },

  getSettings(): Promise<SettingsDTO> {
    return httpClient.get<SettingsDTO>("/settings")
  },

  patchSettings(body: PatchSettingsBody): Promise<SettingsDTO> {
    return httpClient.patch<SettingsDTO>("/settings", body)
  },

  importMovies(
    files: File[],
    options?: MovieImportApiOptions,
  ): Promise<TaskDTO> {
    if (shouldUseResumableImport(files, options?.resumableThresholdBytes)) {
      return httpClient
        .post<MovieImportUploadDTO>("/import/movies/uploads", movieImportUploadManifest(files))
        .then(async (upload) => {
          await uploadMovieFileChunks(upload, files, {
            onUploadProgress: options?.onUploadProgress,
            chunkMaxAttempts: options?.resumableChunkMaxAttempts,
            chunkRetryDelayMs: options?.resumableChunkRetryDelayMs,
          })
          return httpClient.post<TaskDTO>(
            `/import/movies/uploads/${encodeURIComponent(upload.uploadId)}/commit`,
          )
        })
    }

    const form = new FormData()
    const totalBytes = files.reduce((sum, file) => sum + file.size, 0)
    if (totalBytes > 0) {
      form.set("totalBytes", String(totalBytes))
    }
    for (const file of files) {
      const relativePath = relativePathForFile(file)
      form.append("relativePath", relativePath)
      form.append("files", file, relativePath)
    }
    return httpClient.postFormWithProgress<TaskDTO>("/import/movies", form, {
      onUploadProgress: options?.onUploadProgress,
    })
  },

  addLibraryPath(body: AddLibraryPathBody): Promise<AddLibraryPathResultDTO> {
    return httpClient.post<AddLibraryPathResultDTO>("/library/paths", body)
  },

  deleteLibraryPath(id: string): Promise<void> {
    return httpClient.delete(`/library/paths/${encodeURIComponent(id)}`)
  },

  updateLibraryPathTitle(id: string, body: UpdateLibraryPathBody): Promise<LibraryPathDTO> {
    return httpClient.patch<LibraryPathDTO>(`/library/paths/${encodeURIComponent(id)}`, body)
  },

  startScan(body?: StartScanBody): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>("/scans", body)
  },

  refreshMovieMetadata(movieId: string): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>(`/library/movies/${encodeURIComponent(movieId)}/scrape`)
  },

  revealMovieInFileManager(movieId: string): Promise<void> {
    return httpClient.post<void>(`/library/movies/${encodeURIComponent(movieId)}/reveal`)
  },

  revealLibraryPathInFileManager(id: string): Promise<void> {
    return httpClient.post<void>(`/library/paths/${encodeURIComponent(id)}/reveal`)
  },

  startMetadataRefreshByPaths(body: MetadataScrapeByPathsBody): Promise<MetadataRefreshQueuedDTO> {
    return httpClient.post<MetadataRefreshQueuedDTO>("/library/metadata-scrape", body)
  },

  getTaskStatus(taskId: string): Promise<TaskDTO> {
    return httpClient.get<TaskDTO>(`/tasks/${encodeURIComponent(taskId)}`)
  },

  getRecentTasks(limit?: number): Promise<RecentTasksDTO> {
    return httpClient.get<RecentTasksDTO>("/tasks/recent", {
      limit: limit ?? undefined,
    } as Record<string, string | number | undefined>)
  },

  listPlaybackProgress(): Promise<PlaybackProgressListDTO> {
    return httpClient.get<PlaybackProgressListDTO>("/playback/progress")
  },

  putPlaybackProgress(movieId: string, body: PutPlaybackProgressBody): Promise<void> {
    return httpClient.put<void>(`/playback/progress/${encodeURIComponent(movieId)}`, body)
  },

  deletePlaybackProgress(movieId: string): Promise<void> {
    return httpClient.delete(`/playback/progress/${encodeURIComponent(movieId)}`)
  },

  listPlaybackWatchTimeDaily(days?: number): Promise<PlaybackWatchTimeDailyListDTO> {
    return httpClient.get<PlaybackWatchTimeDailyListDTO>("/playback/watch-time/daily", {
      days: days ?? undefined,
    })
  },

  addPlaybackWatchTimeDaily(body: AddPlaybackWatchTimeBody): Promise<void> {
    return httpClient.post<void>("/playback/watch-time/daily", body)
  },

  listCuratedFrames(params?: ListCuratedFramesParams): Promise<CuratedFramesListDTO> {
    return httpClient.get<CuratedFramesListDTO>("/curated-frames", params as Record<string, string | number | undefined>)
  },

  getCuratedFrameStats(): Promise<CuratedFrameStatsDTO> {
    return httpClient.get<CuratedFrameStatsDTO>("/curated-frames/stats")
  },

  listCuratedFrameTags(): Promise<CuratedFrameFacetListDTO> {
    return httpClient.get<CuratedFrameFacetListDTO>("/curated-frames/tags")
  },

  listCuratedFrameActors(): Promise<CuratedFrameFacetListDTO> {
    return httpClient.get<CuratedFrameFacetListDTO>("/curated-frames/actors")
  },

  createCuratedFrame(body: CreateCuratedFrameBody): Promise<void> {
    return httpClient.post<void>("/curated-frames", body)
  },

  createCuratedFrameUpload(body: CreateCuratedFrameBody, image: Blob): Promise<void> {
    const form = new FormData()
    const { imageBase64: _imageBase64, ...metadata } = body
    void _imageBase64
    form.set("metadata", JSON.stringify(metadata))
    form.set("image", image, "frame.png")
    return httpClient.postForm<void>("/curated-frames", form)
  },

  patchCuratedFrameTags(id: string, body: PatchCuratedFrameTagsBody): Promise<void> {
    return httpClient.patch<void>(`/curated-frames/${encodeURIComponent(id)}/tags`, body)
  },

  deleteCuratedFrame(id: string): Promise<void> {
    return httpClient.delete(`/curated-frames/${encodeURIComponent(id)}`)
  },

  async postCuratedFramesExport(body: PostCuratedFramesExportBody): Promise<{ blob: Blob; filename: string }> {
    const { blob, contentDisposition } = await httpClient.postBlob("/curated-frames/export", body)
    return { blob, filename: filenameFromContentDisposition(contentDisposition) }
  },

  pingProvider(name: string): Promise<import("./types").ProviderHealthDTO> {
    return httpClient.post<import("./types").ProviderHealthDTO>("/providers/ping", { name })
  },

  pingAllProviders(): Promise<import("./types").PingAllProvidersResponse> {
    return httpClient.post<import("./types").PingAllProvidersResponse>("/providers/ping-all")
  },

  pingProxyJavbus(body?: ProxyJavBusPingRequestBody): Promise<ProxyJavBusPingResponse> {
    return httpClient.post<ProxyJavBusPingResponse>("/proxy/ping-javbus", body ?? {})
  },

  pingProxyGoogle(body?: ProxyJavBusPingRequestBody): Promise<ProxyJavBusPingResponse> {
    return httpClient.post<ProxyJavBusPingResponse>("/proxy/ping-google", body ?? {})
  },
}
