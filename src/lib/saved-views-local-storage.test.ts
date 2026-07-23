import { describe, expect, it } from "vitest"
import type { SavedViewDTO } from "@/api/types"
import {
  loadLocalSavedViews,
  saveLocalSavedViews,
  SAVED_VIEWS_STORAGE_KEY,
} from "@/lib/saved-views-local-storage"

function view(id: string, name: string, sortOrder: number): SavedViewDTO {
  return {
    id,
    name,
    filters: { schemaVersion: 1, mode: "library", playState: "all", tab: "all" },
    sortOrder,
    createdAt: `2026-07-20T00:00:0${sortOrder}Z`,
    updatedAt: `2026-07-20T00:00:0${sortOrder}Z`,
  }
}

describe("saved views localStorage", () => {
  it("round-trips ordered versioned views", () => {
    let raw: string | null = null
    const storage = {
      getItem: () => raw,
      setItem: (_key: string, value: string) => {
        raw = value
      },
    }
    saveLocalSavedViews([view("b", "Second", 1), view("a", "First", 0)], storage)
    expect(raw).toContain('"schemaVersion":1')
    expect(loadLocalSavedViews(storage).map((item) => item.id)).toEqual(["b", "a"])
    expect(loadLocalSavedViews(storage).map((item) => item.sortOrder)).toEqual([0, 1])
  })

  it("rejects corrupt schema duplicate ids and duplicate normalized names", () => {
    const read = (value: unknown) =>
      loadLocalSavedViews({ getItem: (key: string) => key === SAVED_VIEWS_STORAGE_KEY ? JSON.stringify(value) : null })
    expect(read({ schemaVersion: 2, items: [] })).toEqual([])
    expect(read({ schemaVersion: 1, items: [view("a", "One", 0), view("a", "Two", 1)] })).toEqual([])
    expect(read({ schemaVersion: 1, items: [view("a", "One", 0), view("b", " one ", 1)] })).toEqual([])
  })

  it("does not throw when storage quota rejects a write", () => {
    expect(() =>
      saveLocalSavedViews([view("a", "One", 0)], {
        setItem: () => {
          throw new DOMException("quota", "QuotaExceededError")
        },
      }),
    ).not.toThrow()
  })
})
