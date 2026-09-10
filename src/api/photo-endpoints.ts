import { httpClient } from "./http-client"
import type {
  AddPhotoLibraryPathBody,
  AddPhotoLibraryPathResultDTO,
  ListPhotoBooksParams,
  PatchPhotoSettingsBody,
  PhotoBookDetailDTO,
  PhotoBooksPageDTO,
  PhotoLibraryPathDTO,
  SettingsDTO,
  StartScanBody,
  TaskDTO,
  UpdatePhotoLibraryPathBody,
} from "./types"

function photoListParamsToQuery(
  params?: ListPhotoBooksParams,
): Record<string, string | number | undefined> | undefined {
  if (!params) return undefined
  return {
    q: params.q,
    tag: params.tag,
    favorite: params.favorite === undefined ? undefined : String(params.favorite),
    limit: params.limit,
    offset: params.offset,
  }
}

export const photoApi = {
  getSettings(): Promise<SettingsDTO> {
    return httpClient.get<SettingsDTO>("/settings")
  },

  patchPhotoSettings(body: PatchPhotoSettingsBody): Promise<SettingsDTO> {
    return httpClient.patch<SettingsDTO>("/settings", body)
  },

  listPhotoLibraryPaths(): Promise<PhotoLibraryPathDTO[]> {
    return httpClient.get<PhotoLibraryPathDTO[]>("/library/photos/paths")
  },

  addPhotoLibraryPath(body: AddPhotoLibraryPathBody): Promise<AddPhotoLibraryPathResultDTO> {
    return httpClient.post<AddPhotoLibraryPathResultDTO>("/library/photos/paths", body)
  },

  updatePhotoLibraryPathTitle(
    id: string,
    body: UpdatePhotoLibraryPathBody,
  ): Promise<PhotoLibraryPathDTO> {
    return httpClient.patch<PhotoLibraryPathDTO>(
      `/library/photos/paths/${encodeURIComponent(id)}`,
      body,
    )
  },

  deletePhotoLibraryPath(id: string): Promise<void> {
    return httpClient.delete(`/library/photos/paths/${encodeURIComponent(id)}`)
  },

  startPhotoScan(body?: StartScanBody): Promise<TaskDTO> {
    return httpClient.post<TaskDTO>("/library/photos/scans", body ?? {})
  },

  listPhotos(params?: ListPhotoBooksParams): Promise<PhotoBooksPageDTO> {
    return httpClient.get<PhotoBooksPageDTO>("/library/photos", photoListParamsToQuery(params))
  },

  getPhoto(id: string): Promise<PhotoBookDetailDTO> {
    return httpClient.get<PhotoBookDetailDTO>(`/library/photos/${encodeURIComponent(id)}`)
  },
}
