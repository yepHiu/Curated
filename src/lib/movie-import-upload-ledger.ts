import type { MovieImportUploadFileManifest } from "@/api/types"

/**
 * 本机断点续传会话账本：记录"添加影片"创建的可续传上传会话元数据（仅指纹，
 * 不含文件内容），用于在页面刷新 / 中断后把用户重新选择的文件匹配回原会话，
 * 跳过服务端已持久化的分片。跨浏览器 / 设备不可恢复；与后端 24 小时滑动
 * TTL 对齐，加载时按 lastActiveAt 修剪过期条目。
 */
export const MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY = "curated-movie-import-uploads-v1"
export const MOVIE_IMPORT_UPLOAD_LEDGER_TTL_MS = 24 * 60 * 60 * 1000

export type MovieImportUploadLedgerFile = MovieImportUploadFileManifest

export interface MovieImportUploadLedgerEntry {
  uploadId: string
  targetLibraryPathId: string
  chunkSize: number
  files: MovieImportUploadLedgerFile[]
  createdAt: string
  lastActiveAt: string
}

function isLedgerFile(value: unknown): value is MovieImportUploadLedgerFile {
  if (!value || typeof value !== "object") return false
  const candidate = value as Partial<MovieImportUploadLedgerFile>
  return (
    typeof candidate.relativePath === "string" &&
    candidate.relativePath.trim() !== "" &&
    typeof candidate.size === "number" &&
    Number.isFinite(candidate.size) &&
    candidate.size > 0
  )
}

function coerceEntry(value: unknown): MovieImportUploadLedgerEntry | null {
  if (!value || typeof value !== "object") return null
  const candidate = value as Partial<MovieImportUploadLedgerEntry>
  if (
    typeof candidate.uploadId !== "string" ||
    candidate.uploadId.trim() === "" ||
    typeof candidate.targetLibraryPathId !== "string" ||
    typeof candidate.chunkSize !== "number" ||
    !Array.isArray(candidate.files) ||
    !candidate.files.every(isLedgerFile)
  ) {
    return null
  }
  const now = new Date().toISOString()
  return {
    uploadId: candidate.uploadId,
    targetLibraryPathId: candidate.targetLibraryPathId,
    chunkSize: candidate.chunkSize,
    files: candidate.files,
    createdAt: typeof candidate.createdAt === "string" ? candidate.createdAt : now,
    lastActiveAt: typeof candidate.lastActiveAt === "string" ? candidate.lastActiveAt : now,
  }
}

function persistLedger(entries: MovieImportUploadLedgerEntry[]): void {
  try {
    window.localStorage.setItem(MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY, JSON.stringify(entries))
  } catch {
    // localStorage 不可用（隐私模式 / 配额）时静默降级为仅内存行为
  }
}

/** 读取账本并修剪超过 TTL 的条目（写回存储，读取本身失败返回空列表）。 */
export function loadMovieImportUploadLedger(nowMs = Date.now()): MovieImportUploadLedgerEntry[] {
  let raw: string | null = null
  try {
    raw = window.localStorage.getItem(MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY)
  } catch {
    return []
  }
  if (!raw) return []
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    persistLedger([])
    return []
  }
  if (!Array.isArray(parsed)) {
    persistLedger([])
    return []
  }
  const entries: MovieImportUploadLedgerEntry[] = []
  for (const item of parsed) {
    const entry = coerceEntry(item)
    if (!entry) continue
    const lastActiveMs = Date.parse(entry.lastActiveAt)
    if (Number.isFinite(lastActiveMs) && nowMs - lastActiveMs > MOVIE_IMPORT_UPLOAD_LEDGER_TTL_MS) {
      continue
    }
    if (entries.some((existing) => existing.uploadId === entry.uploadId)) continue
    entries.push(entry)
  }
  if (entries.length !== parsed.length) {
    persistLedger(entries)
  }
  return entries
}

export function addMovieImportUploadLedgerEntry(entry: MovieImportUploadLedgerEntry): void {
  const entries = loadMovieImportUploadLedger().filter(
    (existing) => existing.uploadId !== entry.uploadId,
  )
  entries.push(entry)
  persistLedger(entries)
}

export function removeMovieImportUploadLedgerEntry(uploadId: string): void {
  const entries = loadMovieImportUploadLedger().filter(
    (existing) => existing.uploadId !== uploadId,
  )
  persistLedger(entries)
}

export function touchMovieImportUploadLedgerEntry(uploadId: string, nowMs = Date.now()): void {
  const entries = loadMovieImportUploadLedger()
  const entry = entries.find((existing) => existing.uploadId === uploadId)
  if (!entry) return
  entry.lastActiveAt = new Date(nowMs).toISOString()
  persistLedger(entries)
}

function ledgerFileKey(file: MovieImportUploadLedgerFile): string {
  return `${file.relativePath}\u0000${file.size}\u0000${file.lastModified ?? 0}`
}

/**
 * 判断重新选择的文件集合是否与账本条目完全一致（顺序无关）。
 * 完全一致才允许续传，避免把不同文件写入既有会话。
 */
export function matchesMovieImportUploadFiles(
  entryFiles: MovieImportUploadLedgerFile[],
  files: MovieImportUploadLedgerFile[],
): boolean {
  if (entryFiles.length === 0 || entryFiles.length !== files.length) return false
  const expected = new Set(entryFiles.map(ledgerFileKey))
  for (const file of files) {
    if (!expected.delete(ledgerFileKey(file))) return false
  }
  return expected.size === 0
}
