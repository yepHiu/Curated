import type { MovieCommentDTO } from "@/api/types"

const STORAGE_KEY = "jav-library-movie-comment-v1"

type StoreShape = Record<string, { body: string; updatedAt: string }>

function readAll(): StoreShape {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw?.trim()) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== "object") return {}
    return parsed as StoreShape
  } catch {
    return {}
  }
}

function writeAll(data: StoreShape) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
}

export function getLocalMovieComment(movieId: string): MovieCommentDTO {
  const id = movieId.trim()
  if (!id) return { body: "", updatedAt: "" }
  const row = readAll()[id]
  if (!row) return { body: "", updatedAt: "" }
  return { body: row.body ?? "", updatedAt: row.updatedAt ?? "" }
}

export function putLocalMovieComment(movieId: string, body: string): MovieCommentDTO {
  const id = movieId.trim()
  const updatedAt = new Date().toISOString()
  const all = readAll()
  all[id] = { body, updatedAt }
  writeAll(all)
  return { body, updatedAt }
}
