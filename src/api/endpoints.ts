import { httpClient } from "./http-client"

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
  CreateCuratedFrameBody,
  CreatePlaybackSessionBody,
  CuratedFramesListDTO,
  DevPerformanceSummaryDTO,
  HealthDTO,
  LibraryPathDTO,
  UpdateLibraryPathBody,
  ListActorsParams,
  ListMoviesParams,
  MetadataRefreshQueuedDTO,
  NativePlaybackLaunchDTO,
  MetadataScrapeByPathsBody,
  MovieCommentDTO,
  MovieDetailDTO,
  MoviesPageDTO,
  PlaybackDescriptorDTO,
  PatchCuratedFrameTagsBody,
  PostCuratedFramesExportBody,
  PatchMovieBody,
  PatchSettingsBody,
  PlayedMoviesListDTO,
  PlaybackProgressListDTO,
  ProxyJavBusPingRequestBody,
  ProxyJavBusPingResponse,
  PutMovieCommentBody,
  PutPlaybackProgressBody,
  SettingsDTO,
  StartScanBody,
  RecentTasksDTO,
  TaskDTO,
} from "./types"

export const api = {
  health(): Promise<HealthDTO> {
    return httpClient.get<HealthDTO>("/health")
  },

  getDevPerformanceSummary(): Promise<DevPerformanceSummaryDTO> {
    return httpClient.get<DevPerformanceSummaryDTO>("/dev/performance")
  },

  listPlayedMovies(): Promise<PlayedMoviesListDTO> {
    return httpClient.get<PlayedMoviesListDTO>("/library/played-movies")
  },

  recordPlayedMovie(movieId: string): Promise<void> {
    return httpClient.post<void>(`/library/played-movies/${encodeURIComponent(movieId)}`)
  },

  listMovies(params?: ListMoviesParams): Promise<MoviesPageDTO> {
    return httpClient.get<MoviesPageDTO>("/library/movies", params as Record<string, string | number | undefined>)
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

  scrapeActorProfile(name: string): Promise<TaskDTO> {
    const q = new URLSearchParams({ name })
    return httpClient.post<TaskDTO>(`/library/actors/scrape?${q.toString()}`)
  },

  getMovie(movieId: string): Promise<MovieDetailDTO> {
    return httpClient.get<MovieDetailDTO>(`/library/movies/${encodeURIComponent(movieId)}`)
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
    return httpClient.patch<MovieDetailDTO>(`/library/movies/${encodeURIComponent(movieId)}`, body)
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

  listCuratedFrames(): Promise<CuratedFramesListDTO> {
    return httpClient.get<CuratedFramesListDTO>("/curated-frames")
  },

  createCuratedFrame(body: CreateCuratedFrameBody): Promise<void> {
    return httpClient.post<void>("/curated-frames", body)
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
