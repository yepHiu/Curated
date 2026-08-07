import { HttpClientError, httpClient } from "./http-client"
import {
  assertApiResponse,
  isConnectedClientsDTO,
  isBackupManifestDTO,
  isBackupRestorePreflightDTO,
  isBackupVerificationDTO,
  isHealthDTO,
  isHomepageDailyRecommendationsDTO,
  isLibraryHealthRepairDTO,
  isLibraryHealthReportDTO,
  isMovieDetailDTO,
  isMoviesPageDTO,
  isSavedViewDTO,
  isSavedViewsDTO,
  isRecommendationFeedbackDTO,
  isRecommendationFeedbackListDTO,
  isActorMergeAuditDTO,
  isActorMergeAuditListDTO,
  isActorMergePreviewDTO,
  isPersonalInsightsBreakdownDTO,
  isPersonalInsightsOverviewDTO,
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
  ActorMergeAuditDTO,
  ActorMergeAuditListDTO,
  ActorMergePreviewDTO,
  ActorMergePreviewRequest,
  ApplyActorMergeRequest,
  ActorProfileDTO,
  ActorsListDTO,
  AddLibraryPathBody,
  AddLibraryPathResultDTO,
  AddPlaybackWatchTimeBody,
  AppUpdateStatusDTO,
  AppUpdateInstallBody,
  BackupCreateBody,
  BackupManifestDTO,
  BackupPathBody,
  BackupRestorePreflightDTO,
  BackupVerificationDTO,
  AuthStatusDTO,
  AuthSessionsDTO,
  ChangePinBody,
  ConnectedClientsDTO,
  CreateCuratedFrameBody,
  CreateMovieImportUploadBody,
  CuratedFrameFacetListDTO,
  CuratedFrameStatsDTO,
  CreatePlaybackSessionBody,
  CreateSavedViewBody,
  CreateMovieClipBody,
  CuratedFramesListDTO,
  DevPerformanceSummaryDTO,
  HealthDTO,
  HomepageDailyRecommendationsDTO,
  CreateRecommendationFeedbackBody,
  RecommendationFeedbackDTO,
  RecommendationFeedbackListDTO,
  RefreshHomepageDailyRecommendationsBody,
  CheckLibraryPathStorageStatusBody,
  LibraryPathStorageStatusDTO,
  LibraryPathStorageStatusListDTO,
  LibraryHealthRepairDTO,
  LibraryHealthReportDTO,
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
  PatchAuthSettingsBody,
  PatchCuratedFrameTagsBody,
  PostCuratedFramesExportBody,
  PatchMovieBody,
  StartLibraryHealthRepairBody,
  StartLibraryHealthActionBody,
  PatchSettingsBody,
  PlayedMoviesListDTO,
  PlaybackProgressListDTO,
  PlaybackWatchTimeDailyListDTO,
  PersonalInsightsBreakdownDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
  ProxyJavBusPingRequestBody,
  ProxyJavBusPingResponse,
  PutMovieCommentBody,
  PutPlaybackProgressBody,
  PatchSavedViewBody,
  ReorderSavedViewsBody,
  SavedViewDTO,
  SavedViewsDTO,
  SettingsDTO,
  SetupPinBody,
  StartScanBody,
  RecentTasksDTO,
  TaskDTO,
  UnlockPinBody,
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

  listConnectedClients(): Promise<ConnectedClientsDTO> {
    return httpClient
      .get<unknown>("/connected-clients")
      .then((value) => assertApiResponse("GET /connected-clients", value, isConnectedClientsDTO))
  },

  authStatus(): Promise<AuthStatusDTO> {
    return httpClient.get<AuthStatusDTO>("/auth/status")
  },

  setupPin(body: SetupPinBody): Promise<AuthStatusDTO> {
    return httpClient.post<AuthStatusDTO>("/auth/setup-pin", body)
  },

  unlockPin(body: UnlockPinBody): Promise<AuthStatusDTO> {
    return httpClient.post<AuthStatusDTO>("/auth/unlock", body)
  },

  changePin(body: ChangePinBody): Promise<AuthStatusDTO> {
    return httpClient.post<AuthStatusDTO>("/auth/change-pin", body)
  },

  lockApp(): Promise<AuthStatusDTO> {
    return httpClient.post<AuthStatusDTO>("/auth/lock", {})
  },

  patchAuthSettings(body: PatchAuthSettingsBody): Promise<AuthStatusDTO> {
    return httpClient.patch<AuthStatusDTO>("/auth/settings", body)
  },

  listTrustedAuthSessions(): Promise<AuthSessionsDTO> {
    return httpClient.get<AuthSessionsDTO>("/auth/sessions")
  },

  revokeTrustedAuthSession(publicId: string): Promise<AuthSessionsDTO> {
    return httpClient.delete<AuthSessionsDTO>(`/auth/sessions/${encodeURIComponent(publicId)}`)
  },

  revokeOtherTrustedAuthSessions(): Promise<AuthSessionsDTO> {
    return httpClient.post<AuthSessionsDTO>("/auth/sessions/revoke-others", {})
  },

  getAppUpdateStatus(): Promise<AppUpdateStatusDTO> {
    return httpClient.get<AppUpdateStatusDTO>("/app-update/status")
  },

  checkAppUpdateNow(): Promise<AppUpdateStatusDTO> {
    return httpClient.post<AppUpdateStatusDTO>("/app-update/check")
  },

  downloadAppUpdateInstaller(): Promise<AppUpdateStatusDTO> {
    return httpClient.post<AppUpdateStatusDTO>("/app-update/download")
  },

  installAppUpdate(body: AppUpdateInstallBody = {}): Promise<AppUpdateStatusDTO> {
    return httpClient.post<AppUpdateStatusDTO>("/app-update/install", body)
  },

  clearDownloadedAppUpdateInstaller(): Promise<AppUpdateStatusDTO> {
    return httpClient.delete<AppUpdateStatusDTO>("/app-update/downloaded-installer")
  },

  createBackup(body: BackupCreateBody): Promise<BackupManifestDTO> {
    return httpClient
      .post<unknown>("/maintenance/backups", body)
      .then((value) => assertApiResponse("POST /maintenance/backups", value, isBackupManifestDTO))
  },

  verifyBackup(body: BackupPathBody): Promise<BackupVerificationDTO> {
    return httpClient
      .post<unknown>("/maintenance/backups/verify", body)
      .then((value) => assertApiResponse("POST /maintenance/backups/verify", value, isBackupVerificationDTO))
  },

  preflightBackupRestore(body: BackupPathBody): Promise<BackupRestorePreflightDTO> {
    return httpClient
      .post<unknown>("/maintenance/backups/preflight", body)
      .then((value) => assertApiResponse("POST /maintenance/backups/preflight", value, isBackupRestorePreflightDTO))
  },

  scanLibraryHealth(findingLimit = 500): Promise<LibraryHealthReportDTO> {
    const q = new URLSearchParams({ findingLimit: String(findingLimit) })
    return httpClient
      .post<unknown>(`/library/health/scan?${q.toString()}`)
      .then((value) => assertApiResponse("POST /library/health/scan", value, isLibraryHealthReportDTO))
  },

  startLibraryHealthRepair(body: StartLibraryHealthRepairBody): Promise<LibraryHealthRepairDTO> {
    return httpClient
      .post<unknown>("/library/health/repairs", body)
      .then((value) => assertApiResponse("POST /library/health/repairs", value, isLibraryHealthRepairDTO))
  },

  getLibraryHealthRepair(repairId: string): Promise<LibraryHealthRepairDTO> {
    return httpClient
      .get<unknown>(`/library/health/repairs/${encodeURIComponent(repairId)}`)
      .then((value) => assertApiResponse("GET /library/health/repairs/{id}", value, isLibraryHealthRepairDTO))
  },

  startLibraryHealthAction(body: StartLibraryHealthActionBody): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>("/library/health/actions", body)
  },

  listPlayedMovies(): Promise<PlayedMoviesListDTO> {
    return httpClient.get<PlayedMoviesListDTO>("/library/played-movies")
  },

  getHomepageDailyRecommendations(): Promise<HomepageDailyRecommendationsDTO> {
    return httpClient
      .get<unknown>("/homepage/recommendations")
      .then((value) => assertApiResponse("GET /homepage/recommendations", value, isHomepageDailyRecommendationsDTO))
  },

  refreshHomepageDailyRecommendations(
    body?: RefreshHomepageDailyRecommendationsBody,
  ): Promise<HomepageDailyRecommendationsDTO> {
    return httpClient
      .post<unknown>("/homepage/recommendations/refresh", body)
      .then((value) => assertApiResponse("POST /homepage/recommendations/refresh", value, isHomepageDailyRecommendationsDTO))
  },

  listHomepageRecommendationFeedback(): Promise<RecommendationFeedbackListDTO> {
    return httpClient
      .get<unknown>("/homepage/recommendations/feedback")
      .then((value) => assertApiResponse("GET /homepage/recommendations/feedback", value, isRecommendationFeedbackListDTO))
  },

  createHomepageRecommendationFeedback(body: CreateRecommendationFeedbackBody): Promise<RecommendationFeedbackDTO> {
    return httpClient
      .post<unknown>("/homepage/recommendations/feedback", body)
      .then((value) => assertApiResponse("POST /homepage/recommendations/feedback", value, isRecommendationFeedbackDTO))
  },

  deleteHomepageRecommendationFeedback(id: string): Promise<void> {
    return httpClient.delete(`/homepage/recommendations/feedback/${encodeURIComponent(id)}`)
  },

  recordPlayedMovie(movieId: string): Promise<void> {
    return httpClient.post<void>(`/library/played-movies/${encodeURIComponent(movieId)}`)
  },

  listMovies(params?: ListMoviesParams): Promise<MoviesPageDTO> {
    return httpClient
      .get<unknown>("/library/movies", params as Record<string, string | number | undefined>)
      .then((value) => assertApiResponse("GET /library/movies", value, isMoviesPageDTO))
  },

  listSavedViews(): Promise<SavedViewsDTO> {
    return httpClient
      .get<unknown>("/library/saved-views")
      .then((value) => assertApiResponse("GET /library/saved-views", value, isSavedViewsDTO))
  },

  createSavedView(body: CreateSavedViewBody): Promise<SavedViewDTO> {
    return httpClient
      .post<unknown>("/library/saved-views", body)
      .then((value) => assertApiResponse("POST /library/saved-views", value, isSavedViewDTO))
  },

  patchSavedView(id: string, body: PatchSavedViewBody): Promise<SavedViewDTO> {
    return httpClient
      .patch<unknown>(`/library/saved-views/${encodeURIComponent(id)}`, body)
      .then((value) => assertApiResponse("PATCH /library/saved-views/:id", value, isSavedViewDTO))
  },

  deleteSavedView(id: string): Promise<void> {
    return httpClient.delete(`/library/saved-views/${encodeURIComponent(id)}`)
  },

  reorderSavedViews(body: ReorderSavedViewsBody): Promise<SavedViewsDTO> {
    return httpClient
      .put<unknown>("/library/saved-views/order", body)
      .then((value) => assertApiResponse("PUT /library/saved-views/order", value, isSavedViewsDTO))
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

  previewActorMerge(body: ActorMergePreviewRequest): Promise<ActorMergePreviewDTO> {
    return httpClient
      .post<unknown>("/library/actors/merge-preview", body)
      .then((value) => assertApiResponse("POST /library/actors/merge-preview", value, isActorMergePreviewDTO))
  },

  applyActorMerge(body: ApplyActorMergeRequest): Promise<ActorMergeAuditDTO> {
    return httpClient
      .post<unknown>("/library/actors/merge", body)
      .then((value) => assertApiResponse("POST /library/actors/merge", value, isActorMergeAuditDTO))
  },

  listActorMergeAudits(params?: { limit?: number; offset?: number }): Promise<ActorMergeAuditListDTO> {
    return httpClient
      .get<unknown>("/library/actors/merge-audits", params)
      .then((value) => assertApiResponse("GET /library/actors/merge-audits", value, isActorMergeAuditListDTO))
  },

  getPersonalInsightsOverview(params: {
    range: PersonalInsightsRange
    timezone: string
  }): Promise<PersonalInsightsOverviewDTO> {
    return httpClient
      .get<unknown>("/insights/overview", params)
      .then((value) => assertApiResponse("GET /insights/overview", value, isPersonalInsightsOverviewDTO))
  },

  getPersonalInsightsBreakdown(params: {
    range: PersonalInsightsRange
    timezone: string
    dimension: PersonalInsightsDimension
    limit?: number
  }): Promise<PersonalInsightsBreakdownDTO> {
    return httpClient
      .get<unknown>("/insights/breakdown", params)
      .then((value) => assertApiResponse("GET /insights/breakdown", value, isPersonalInsightsBreakdownDTO))
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

  listLibraryPathStorageStatus(): Promise<LibraryPathStorageStatusListDTO> {
    return httpClient.get<LibraryPathStorageStatusListDTO>("/library/paths/storage-status")
  },

  checkLibraryPathStorageStatus(
    body?: CheckLibraryPathStorageStatusBody,
  ): Promise<LibraryPathStorageStatusListDTO> {
    return httpClient.post<LibraryPathStorageStatusListDTO>(
      "/library/paths/storage-status/check",
      body ?? {},
    )
  },

  rebindLibraryPathStorage(id: string): Promise<LibraryPathStorageStatusDTO> {
    return httpClient.post<LibraryPathStorageStatusDTO>(
      `/library/paths/${encodeURIComponent(id)}/storage-binding/rebind`,
    )
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

  createMovieClip(movieId: string, body: CreateMovieClipBody): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>(`/library/movies/${encodeURIComponent(movieId)}/clips`, body)
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
