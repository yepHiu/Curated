import type { BookCommentDTO } from "@/api/types"

/** Mock 漫画备注 localStorage 键，与影片备注账本隔离。 */
export const MOCK_COMIC_COMMENTS_KEY = "curated-mock-comic-comments-v1"
/** Mock 写真备注 localStorage 键，与漫画备注账本隔离。 */
export const MOCK_PHOTO_COMMENTS_KEY = "curated-mock-photo-comments-v1"

type StoreShape = Record<string, { body: string; updatedAt: string }>

/** 读取指定 Mock 备注账本，损坏载荷视为空表。 */
function readAll(storageKey: string): StoreShape {
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw?.trim()) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== "object") return {}
    return parsed as StoreShape
  } catch {
    return {}
  }
}

/** 把完整 Mock 备注账本写回 localStorage。 */
function writeAll(storageKey: string, data: StoreShape) {
  localStorage.setItem(storageKey, JSON.stringify(data))
}

/** 读取一本漫画或写真的本地备注；缺失时返回空正文。 */
export function getLocalBookComment(storageKey: string, entityId: string): BookCommentDTO {
  const id = entityId.trim()
  if (!id) return { body: "", updatedAt: "" }
  const row = readAll(storageKey)[id]
  if (!row) return { body: "", updatedAt: "" }
  return { body: row.body ?? "", updatedAt: row.updatedAt ?? "" }
}

/** 覆盖保存一本漫画或写真的本地备注。 */
export function putLocalBookComment(storageKey: string, entityId: string, body: string): BookCommentDTO {
  const id = entityId.trim()
  const updatedAt = new Date().toISOString()
  const all = readAll(storageKey)
  all[id] = { body, updatedAt }
  writeAll(storageKey, all)
  return { body, updatedAt }
}

/** 删除一本漫画或写真的本地备注，供 Mock 删除书时清理。 */
export function removeLocalBookComment(storageKey: string, entityId: string) {
  const id = entityId.trim()
  if (!id) return
  const all = readAll(storageKey)
  if (!(id in all)) return
  delete all[id]
  writeAll(storageKey, all)
}
