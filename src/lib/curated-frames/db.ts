import type { CuratedFrameRecord } from "@/domain/curated-frame/types"
import { api } from "@/api/endpoints"
import { bumpCuratedFramesRevision } from "@/lib/curated-frames/revision"
import { filterCuratedFramesByQuery } from "@/lib/curated-frames/search"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

const DB_NAME = "jav-curated-frames"
const DB_VERSION = 1
const STORE_FRAMES = "frames"
const STORE_KV = "kv"
const KV_DIRECTORY_HANDLE = "export-directory-handle"

/** Web API 模式下无本地 Blob，展示用 curatedFrameImageUrl(id) */
export interface CuratedFrameDbRow extends CuratedFrameRecord {
  imageBlob?: Blob
}

export interface CuratedFrameListQuery {
  q?: string
  actor?: string
  movieId?: string
  tag?: string
  limit?: number
  offset?: number
}

export interface CuratedFramePageResult {
  items: CuratedFrameDbRow[]
  total: number
  limit: number
  offset: number
}

function mapCuratedFrameItem(it: CuratedFrameRecord): CuratedFrameDbRow {
  return {
    id: it.id,
    movieId: it.movieId,
    title: it.title,
    code: it.code,
    actors: [...it.actors],
    positionSec: it.positionSec,
    capturedAt: it.capturedAt,
    tags: [...it.tags],
  }
}

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION)
    req.onerror = () => reject(req.error ?? new Error("indexedDB open failed"))
    req.onsuccess = () => resolve(req.result)
    req.onupgradeneeded = () => {
      const db = req.result
      if (!db.objectStoreNames.contains(STORE_FRAMES)) {
        db.createObjectStore(STORE_FRAMES, { keyPath: "id" })
      }
      if (!db.objectStoreNames.contains(STORE_KV)) {
        db.createObjectStore(STORE_KV, { keyPath: "key" })
      }
    }
  })
}

function reqToPromise<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error ?? new Error("IDBRequest failed"))
  })
}

export async function putCuratedFrame(row: CuratedFrameDbRow): Promise<void> {
  if (USE_WEB) {
    throw new Error("putCuratedFrame: Web API 模式请使用 saveCuratedCaptureFromVideo 内的后端写入")
  }
  if (!row.imageBlob) {
    throw new Error("putCuratedFrame: 本地模式需要 imageBlob")
  }
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_FRAMES, "readwrite")
    await reqToPromise(tx.objectStore(STORE_FRAMES).put(row))
    await new Promise<void>((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error ?? new Error("tx failed"))
      tx.onabort = () => reject(tx.error ?? new Error("tx aborted"))
    })
    bumpCuratedFramesRevision()
  } finally {
    db.close()
  }
}

export async function listCuratedFramesByCapturedAtDesc(): Promise<CuratedFrameDbRow[]> {
  const page = await listCuratedFramesPage()
  return page.items
}

export async function listCuratedFramesPage(
  query: CuratedFrameListQuery = {},
): Promise<CuratedFramePageResult> {
  if (USE_WEB) {
    const page = await api.listCuratedFrames(query)
    return {
      items: page.items.map(mapCuratedFrameItem),
      total: page.total,
      limit: page.limit,
      offset: page.offset,
    }
  }
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_FRAMES, "readonly")
    const store = tx.objectStore(STORE_FRAMES)
    const rows = await reqToPromise(store.getAll() as IDBRequest<CuratedFrameDbRow[]>)
    const ordered = rows.sort((a, b) => b.capturedAt.localeCompare(a.capturedAt))
    let filtered = ordered
    if (query.q?.trim()) {
      filtered = filterCuratedFramesByQuery(filtered, query.q)
    }
    if (query.actor?.trim()) {
      const actor = query.actor.trim()
      filtered = filtered.filter((row) => row.actors.some((name) => name.trim() === actor))
    }
    if (query.movieId?.trim()) {
      const movieId = query.movieId.trim()
      filtered = filtered.filter((row) => row.movieId.trim() === movieId)
    }
    if (query.tag?.trim()) {
      const tag = query.tag.trim()
      filtered = filtered.filter((row) => row.tags.some((name) => name.trim() === tag))
    }
    const total = filtered.length
    const limit = query.limit && query.limit > 0 ? query.limit : total
    const offset = query.offset && query.offset > 0 ? query.offset : 0
    return {
      items: filtered.slice(offset, offset + limit),
      total,
      limit,
      offset,
    }
  } finally {
    db.close()
  }
}

export async function updateCuratedFrameTags(id: string, tags: string[]): Promise<void> {
  if (USE_WEB) {
    await api.patchCuratedFrameTags(id, { tags: [...tags] })
    bumpCuratedFramesRevision()
    return
  }
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_FRAMES, "readwrite")
    const store = tx.objectStore(STORE_FRAMES)
    const row = await reqToPromise(store.get(id) as IDBRequest<CuratedFrameDbRow | undefined>)
    if (!row) {
      await new Promise<void>((resolve, reject) => {
        tx.oncomplete = () => resolve()
        tx.onerror = () => reject(tx.error)
      })
      return
    }
    row.tags = [...tags]
    await reqToPromise(store.put(row))
    await new Promise<void>((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error ?? new Error("tx failed"))
    })
    bumpCuratedFramesRevision()
  } finally {
    db.close()
  }
}

export async function findNearbyCuratedFrame(
  movieId: string,
  positionSec: number,
  thresholdSec: number,
): Promise<CuratedFrameDbRow | null> {
  if (USE_WEB || !movieId.trim() || !(thresholdSec > 0)) {
    return null
  }
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_FRAMES, "readonly")
    const rows = await reqToPromise(tx.objectStore(STORE_FRAMES).getAll() as IDBRequest<CuratedFrameDbRow[]>)
    const matches = rows
      .filter((row) => row.movieId.trim() === movieId.trim())
      .filter((row) => Math.abs(row.positionSec - positionSec) <= thresholdSec)
      .sort((a, b) => {
        const diff = Math.abs(a.positionSec - positionSec) - Math.abs(b.positionSec - positionSec)
        if (diff !== 0) {
          return diff
        }
        return b.capturedAt.localeCompare(a.capturedAt)
      })
    return matches[0] ?? null
  } finally {
    db.close()
  }
}

export async function deleteCuratedFrame(id: string): Promise<void> {
  if (USE_WEB) {
    await api.deleteCuratedFrame(id)
    bumpCuratedFramesRevision()
    return
  }
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_FRAMES, "readwrite")
    await reqToPromise(tx.objectStore(STORE_FRAMES).delete(id))
    await new Promise<void>((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error ?? new Error("tx failed"))
    })
    bumpCuratedFramesRevision()
  } finally {
    db.close()
  }
}

export async function countCuratedFrames(): Promise<number> {
  if (USE_WEB) {
    const { total } = await api.getCuratedFrameStats()
    return total
  }
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_FRAMES, "readonly")
    return await reqToPromise(tx.objectStore(STORE_FRAMES).count())
  } finally {
    db.close()
  }
}

export async function listCuratedFrameTagSuggestions(): Promise<string[]> {
  if (USE_WEB) {
    const { items } = await api.listCuratedFrameTags()
    return items.map((item) => item.name)
  }
  const page = await listCuratedFramesPage()
  const set = new Set<string>()
  for (const row of page.items) {
    for (const tag of row.tags) {
      const value = tag.trim()
      if (value) {
        set.add(value)
      }
    }
  }
  return [...set].sort((a, b) => a.localeCompare(b, "zh-CN", { numeric: true }))
}

export async function getStoredDirectoryHandle(): Promise<FileSystemDirectoryHandle | null> {
  if (typeof indexedDB === "undefined") return null
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_KV, "readonly")
    const row = await reqToPromise(
      tx.objectStore(STORE_KV).get(KV_DIRECTORY_HANDLE) as IDBRequest<
        { key: string; handle: FileSystemDirectoryHandle } | undefined
      >,
    )
    return row?.handle ?? null
  } finally {
    db.close()
  }
}

export async function setStoredDirectoryHandle(
  handle: FileSystemDirectoryHandle | null,
): Promise<void> {
  const db = await openDb()
  try {
    const tx = db.transaction(STORE_KV, "readwrite")
    const store = tx.objectStore(STORE_KV)
    if (!handle) {
      await reqToPromise(store.delete(KV_DIRECTORY_HANDLE))
    } else {
      await reqToPromise(store.put({ key: KV_DIRECTORY_HANDLE, handle }))
    }
    await new Promise<void>((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error ?? new Error("tx failed"))
    })
  } finally {
    db.close()
  }
}

export function supportsFileSystemAccess(): boolean {
  return typeof window !== "undefined" && "showDirectoryPicker" in window
}
