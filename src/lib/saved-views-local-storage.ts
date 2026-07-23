import type { SavedViewDTO } from "@/api/types"
import { isSavedViewDTO } from "@/api/guards"

export const SAVED_VIEWS_STORAGE_KEY = "curated-library-saved-views-v1"

interface SavedViewsStorageShape {
  schemaVersion: 1
  items: SavedViewDTO[]
}

export function loadLocalSavedViews(storage: Pick<Storage, "getItem"> | undefined = globalThis.localStorage): SavedViewDTO[] {
  if (!storage) {
    return []
  }
  try {
    const raw = storage.getItem(SAVED_VIEWS_STORAGE_KEY)
    if (!raw) {
      return []
    }
    const parsed = JSON.parse(raw) as unknown
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      Array.isArray(parsed) ||
      (parsed as { schemaVersion?: unknown }).schemaVersion !== 1 ||
      !Array.isArray((parsed as { items?: unknown }).items)
    ) {
      return []
    }
    const items = (parsed as SavedViewsStorageShape).items
    if (!items.every(isSavedViewDTO)) {
      return []
    }
    const ids = new Set<string>()
    const names = new Set<string>()
    for (const item of items) {
      const normalizedName = item.name.trim().toLocaleLowerCase()
      if (!item.id.trim() || !normalizedName || ids.has(item.id) || names.has(normalizedName)) {
        return []
      }
      ids.add(item.id)
      names.add(normalizedName)
    }
    return items
      .map((item) => ({ ...item, filters: { ...item.filters } }))
      .sort((left, right) => left.sortOrder - right.sortOrder || left.createdAt.localeCompare(right.createdAt))
      .map((item, sortOrder) => ({ ...item, sortOrder }))
  } catch {
    return []
  }
}

export function saveLocalSavedViews(
  items: readonly SavedViewDTO[],
  storage: Pick<Storage, "setItem"> | undefined = globalThis.localStorage,
): void {
  if (!storage) {
    return
  }
  const payload: SavedViewsStorageShape = {
    schemaVersion: 1,
    items: items.map((item, sortOrder) => ({
      ...item,
      filters: { ...item.filters },
      sortOrder,
    })),
  }
  try {
    storage.setItem(SAVED_VIEWS_STORAGE_KEY, JSON.stringify(payload))
  } catch {
    // Private mode / quota failures must not break the current in-memory view list.
  }
}
