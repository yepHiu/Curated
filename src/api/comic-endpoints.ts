import { httpClient } from "./http-client"
import type {
  AddComicLibraryPathBody,
  AddComicLibraryPathResultDTO,
  ComicLibraryPathDTO,
  PatchComicSettingsBody,
  SettingsDTO,
  TaskDTO,
  UpdateComicLibraryPathBody,
} from "./types"

export const comicApi = {
  getSettings(): Promise<SettingsDTO> {
    return httpClient.get<SettingsDTO>("/settings")
  },

  patchComicSettings(body: PatchComicSettingsBody): Promise<SettingsDTO> {
    return httpClient.patch<SettingsDTO>("/settings", body)
  },

  listComicLibraryPaths(): Promise<ComicLibraryPathDTO[]> {
    return httpClient.get<ComicLibraryPathDTO[]>("/library/comics/paths")
  },

  addComicLibraryPath(body: AddComicLibraryPathBody): Promise<AddComicLibraryPathResultDTO> {
    return httpClient.post<AddComicLibraryPathResultDTO>("/library/comics/paths", body)
  },

  updateComicLibraryPathTitle(
    id: string,
    body: UpdateComicLibraryPathBody,
  ): Promise<ComicLibraryPathDTO> {
    return httpClient.patch<ComicLibraryPathDTO>(
      `/library/comics/paths/${encodeURIComponent(id)}`,
      body,
    )
  },

  deleteComicLibraryPath(id: string): Promise<void> {
    return httpClient.delete(`/library/comics/paths/${encodeURIComponent(id)}`)
  },

  startComicScan(): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>("/library/comics/scans", {})
  },
}
