import { httpClient } from "./http-client"
import type {
  HealthDTO,
  ListMoviesParams,
  MovieDetailDTO,
  MoviesPageDTO,
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

  getSettings(): Promise<SettingsDTO> {
    return httpClient.get<SettingsDTO>("/settings")
  },

  startScan(body?: StartScanBody): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>("/scans", body)
  },

  getTaskStatus(taskId: string): Promise<TaskDTO> {
    return httpClient.get<TaskDTO>(`/tasks/${encodeURIComponent(taskId)}`)
  },
}
