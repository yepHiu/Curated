import { httpClient } from "./http-client"
import type {
  AddLibraryPathBody,
  HealthDTO,
  LibraryPathDTO,
  UpdateLibraryPathBody,
  ListMoviesParams,
  MetadataRefreshQueuedDTO,
  MetadataScrapeByPathsBody,
  MovieDetailDTO,
  MoviesPageDTO,
  PatchMovieBody,
  PatchSettingsBody,
  SettingsDTO,
  StartScanBody,
  TaskDTO,
} from "./types"

export const api = {
  health(): Promise<HealthDTO> {
    return httpClient.get<HealthDTO>("/health")
  },

  listMovies(params?: ListMoviesParams): Promise<MoviesPageDTO> {
    return httpClient.get<MoviesPageDTO>("/library/movies", params as Record<string, string | number | undefined>)
  },

  getMovie(movieId: string): Promise<MovieDetailDTO> {
    return httpClient.get<MovieDetailDTO>(`/library/movies/${encodeURIComponent(movieId)}`)
  },

  patchMovie(movieId: string, body: PatchMovieBody): Promise<MovieDetailDTO> {
    return httpClient.patch<MovieDetailDTO>(`/library/movies/${encodeURIComponent(movieId)}`, body)
  },

  deleteMovie(movieId: string): Promise<void> {
    return httpClient.delete(`/library/movies/${encodeURIComponent(movieId)}`)
  },

  getSettings(): Promise<SettingsDTO> {
    return httpClient.get<SettingsDTO>("/settings")
  },

  patchSettings(body: PatchSettingsBody): Promise<SettingsDTO> {
    return httpClient.patch<SettingsDTO>("/settings", body)
  },

  addLibraryPath(body: AddLibraryPathBody): Promise<LibraryPathDTO> {
    return httpClient.post<LibraryPathDTO>("/library/paths", body)
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

  startMetadataRefreshByPaths(body: MetadataScrapeByPathsBody): Promise<MetadataRefreshQueuedDTO> {
    return httpClient.post<MetadataRefreshQueuedDTO>("/library/metadata-scrape", body)
  },

  getTaskStatus(taskId: string): Promise<TaskDTO> {
    return httpClient.get<TaskDTO>(`/tasks/${encodeURIComponent(taskId)}`)
  },
}
