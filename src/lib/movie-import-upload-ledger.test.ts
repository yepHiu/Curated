import { beforeEach, describe, expect, it } from "vitest"
import {
  MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY,
  MOVIE_IMPORT_UPLOAD_LEDGER_TTL_MS,
  addMovieImportUploadLedgerEntry,
  loadMovieImportUploadLedger,
  matchesMovieImportUploadFiles,
  removeMovieImportUploadLedgerEntry,
  touchMovieImportUploadLedgerEntry,
} from "./movie-import-upload-ledger"

function makeEntry(overrides: Partial<Parameters<typeof addMovieImportUploadLedgerEntry>[0]> = {}) {
  return {
    uploadId: "upload_abc123def4567890",
    targetLibraryPathId: "library-a",
    chunkSize: 32 * 1024 * 1024,
    files: [
      { relativePath: "ABC-001.mp4", size: 1024, lastModified: 1755400000000 },
      { relativePath: "Folder/ABC-002.mkv", size: 2048, lastModified: 1755400000001 },
    ],
    createdAt: "2026-08-17T00:00:00.000Z",
    lastActiveAt: "2026-08-17T00:00:00.000Z",
    ...overrides,
  }
}

beforeEach(() => {
  window.localStorage.clear()
})

describe("movie import upload ledger", () => {
  it("round-trips entries through localStorage", () => {
    addMovieImportUploadLedgerEntry(makeEntry())

    const entries = loadMovieImportUploadLedger()
    expect(entries).toHaveLength(1)
    expect(entries[0].uploadId).toBe("upload_abc123def4567890")
    expect(entries[0].files).toHaveLength(2)
    expect(entries[0].files[0]).toEqual({
      relativePath: "ABC-001.mp4",
      size: 1024,
      lastModified: 1755400000000,
    })
  })

  it("replaces entries with the same uploadId instead of duplicating", () => {
    addMovieImportUploadLedgerEntry(makeEntry())
    addMovieImportUploadLedgerEntry(makeEntry({ chunkSize: 16 * 1024 * 1024 }))

    const entries = loadMovieImportUploadLedger()
    expect(entries).toHaveLength(1)
    expect(entries[0].chunkSize).toBe(16 * 1024 * 1024)
  })

  it("removes entries by uploadId", () => {
    addMovieImportUploadLedgerEntry(makeEntry())
    removeMovieImportUploadLedgerEntry("upload_abc123def4567890")

    expect(loadMovieImportUploadLedger()).toHaveLength(0)
    expect(window.localStorage.getItem(MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY)).toBe("[]")
  })

  it("prunes entries inactive beyond the TTL and writes the pruned list back", () => {
    const recent = makeEntry()
    const stale = makeEntry({
      uploadId: "upload_old000000000001",
      lastActiveAt: new Date(Date.now() - MOVIE_IMPORT_UPLOAD_LEDGER_TTL_MS - 60_000).toISOString(),
    })
    window.localStorage.setItem(
      MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY,
      JSON.stringify([stale, recent]),
    )

    const entries = loadMovieImportUploadLedger()
    expect(entries).toHaveLength(1)
    expect(entries[0].uploadId).toBe(recent.uploadId)
    const stored = JSON.parse(
      window.localStorage.getItem(MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY) ?? "[]",
    )
    expect(stored).toHaveLength(1)
  })

  it("drops malformed stored payloads without throwing", () => {
    window.localStorage.setItem(MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY, "not-json")
    expect(loadMovieImportUploadLedger()).toEqual([])

    window.localStorage.setItem(MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY, JSON.stringify({ nope: 1 }))
    expect(loadMovieImportUploadLedger()).toEqual([])

    window.localStorage.setItem(
      MOVIE_IMPORT_UPLOAD_LEDGER_STORAGE_KEY,
      JSON.stringify([{ uploadId: 42 }]),
    )
    expect(loadMovieImportUploadLedger()).toEqual([])
  })

  it("touches lastActiveAt for an existing entry only", () => {
    const before = makeEntry()
    addMovieImportUploadLedgerEntry(before)
    touchMovieImportUploadLedgerEntry(
      before.uploadId,
      Date.parse("2026-08-18T00:00:00.000Z"),
    )

    const entries = loadMovieImportUploadLedger(Date.parse("2026-08-18T00:00:00.000Z"))
    expect(entries[0].lastActiveAt).toBe("2026-08-18T00:00:00.000Z")

    touchMovieImportUploadLedgerEntry("upload_missing00000001")
    expect(loadMovieImportUploadLedger()).toHaveLength(1)
  })
})

describe("matchesMovieImportUploadFiles", () => {
  const entryFiles = makeEntry().files

  it("matches the same file set regardless of order", () => {
    expect(
      matchesMovieImportUploadFiles(entryFiles, [entryFiles[1], entryFiles[0]]),
    ).toBe(true)
  })

  it("rejects mismatched size, path, or lastModified", () => {
    expect(
      matchesMovieImportUploadFiles(entryFiles, [
        { ...entryFiles[0], size: 999 },
        entryFiles[1],
      ]),
    ).toBe(false)
    expect(
      matchesMovieImportUploadFiles(entryFiles, [
        { ...entryFiles[0], relativePath: "OTHER-001.mp4" },
        entryFiles[1],
      ]),
    ).toBe(false)
    expect(
      matchesMovieImportUploadFiles(entryFiles, [
        { ...entryFiles[0], lastModified: 1 },
        entryFiles[1],
      ]),
    ).toBe(false)
  })

  it("rejects different set sizes and empty manifests", () => {
    expect(matchesMovieImportUploadFiles(entryFiles, [entryFiles[0]])).toBe(false)
    expect(matchesMovieImportUploadFiles(entryFiles, [])).toBe(false)
    expect(matchesMovieImportUploadFiles([], [])).toBe(false)
  })

  it("treats missing lastModified as 0 consistently", () => {
    const files = [{ relativePath: "ABC-003.mp4", size: 10 }]
    expect(matchesMovieImportUploadFiles(files, [{ ...files[0], lastModified: 0 }])).toBe(true)
    expect(matchesMovieImportUploadFiles(files, [...files])).toBe(true)
    expect(matchesMovieImportUploadFiles(files, [{ ...files[0], lastModified: 7 }])).toBe(false)
  })
})
