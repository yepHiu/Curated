import { describe, expect, it } from "vitest"
import { applyLibraryBatchToggle } from "./library-batch-selection"

const orderedIds = ["a", "b", "c", "d", "e"]

describe("applyLibraryBatchToggle", () => {
  it("toggles a single movie and updates the anchor", () => {
    const selected = applyLibraryBatchToggle({
      selectedIds: new Set(),
      orderedIds,
      movieId: "c",
      shiftKey: false,
      anchorId: null,
      maxCount: 100,
    })
    expect(selected.selectedIds).toEqual(new Set(["c"]))
    expect(selected.anchorId).toBe("c")
    expect(selected.truncated).toBe(false)

    const deselected = applyLibraryBatchToggle({
      selectedIds: selected.selectedIds,
      orderedIds,
      movieId: "c",
      shiftKey: false,
      anchorId: selected.anchorId,
      maxCount: 100,
    })
    expect(deselected.selectedIds).toEqual(new Set())
    expect(deselected.anchorId).toBe("c")
  })

  it("selects the inclusive range between the anchor and the shift-clicked movie", () => {
    const result = applyLibraryBatchToggle({
      selectedIds: new Set(["b"]),
      orderedIds,
      movieId: "d",
      shiftKey: true,
      anchorId: "b",
      maxCount: 100,
    })
    expect(result.selectedIds).toEqual(new Set(["b", "c", "d"]))
    expect(result.anchorId).toBe("b")
    expect(result.truncated).toBe(false)
  })

  it("selects a backward range without dropping the original anchor", () => {
    const result = applyLibraryBatchToggle({
      selectedIds: new Set(["e"]),
      orderedIds,
      movieId: "c",
      shiftKey: true,
      anchorId: "e",
      maxCount: 100,
    })
    expect(result.selectedIds).toEqual(new Set(["c", "d", "e"]))
    expect(result.anchorId).toBe("e")
  })

  it("unions a shift range onto movies already selected outside it", () => {
    const result = applyLibraryBatchToggle({
      selectedIds: new Set(["a", "c"]),
      orderedIds,
      movieId: "e",
      shiftKey: true,
      anchorId: "c",
      maxCount: 100,
    })
    expect(result.selectedIds).toEqual(new Set(["a", "c", "d", "e"]))
  })

  it("falls back to a single toggle when shift is held without a usable anchor", () => {
    const result = applyLibraryBatchToggle({
      selectedIds: new Set(),
      orderedIds,
      movieId: "d",
      shiftKey: true,
      anchorId: null,
      maxCount: 100,
    })
    expect(result.selectedIds).toEqual(new Set(["d"]))
    expect(result.anchorId).toBe("d")
  })

  it("caps a shift range to the visible selection limit in display order", () => {
    const result = applyLibraryBatchToggle({
      selectedIds: new Set(["a"]),
      orderedIds,
      movieId: "e",
      shiftKey: true,
      anchorId: "a",
      maxCount: 3,
    })
    expect(result.selectedIds).toEqual(new Set(["a", "b", "c"]))
    expect(result.truncated).toBe(true)
    expect(result.anchorId).toBe("a")
  })
})
