import { httpClient } from "./http-client"
import type {
  AddPhotoLibraryPathBody,
  AddPhotoLibraryPathResultDTO,
  BookCommentDTO,
  ListPhotoBooksParams,
  PatchPhotoSettingsBody,
  PhotoBookDetailDTO,
  PhotoBooksPageDTO,
  PhotoLibraryPathDTO,
  PhotoImportUploadProgress,
  PatchPhotoBookBody,
  PutBookCommentBody,
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
  /** Upload archives to the independent default photo root. */
  importPhotos(files: File[], options?: { onUploadProgress?: (progress: PhotoImportUploadProgress) => void }): Promise<TaskDTO> {
    const form = new FormData()
    form.set("totalBytes", String(files.reduce((sum, file) => sum + file.size, 0)))
    for (const file of files) {
      form.append("files", file, file.name)
    }
    return httpClient.postFormWithProgress<TaskDTO>("/import/photos", form, { onUploadProgress: options?.onUploadProgress })
  },
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

  /** 更新一本写真的可写字段，目前用于本地评分。 */
  patchPhoto(id: string, body: PatchPhotoBookBody): Promise<PhotoBookDetailDTO> {
    return httpClient.patch<PhotoBookDetailDTO>(`/library/photos/${encodeURIComponent(id)}`, body)
  },

  replacePhotoTags(id: string, tags: string[]): Promise<PhotoBookDetailDTO> {
    return httpClient.patch<PhotoBookDetailDTO>(`/library/photos/books/${encodeURIComponent(id)}/tags`, { tags })
  },

  /** 读取一本写真的个人备注；尚未保存时返回空正文。 */
  getPhotoComment(id: string): Promise<BookCommentDTO> {
    return httpClient.get<BookCommentDTO>(
      `/library/photos/books/${encodeURIComponent(id)}/comment`,
    )
  },

  /** 覆盖保存一本写真的个人备注。 */
  putPhotoComment(id: string, body: PutBookCommentBody): Promise<BookCommentDTO> {
    return httpClient.put<BookCommentDTO>(
      `/library/photos/books/${encodeURIComponent(id)}/comment`,
      body,
    )
  },
}
