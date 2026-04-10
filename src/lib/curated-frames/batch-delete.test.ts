import { describe, expect, it, vi } from "vitest"
import { deleteCuratedFramesBatch } from "@/lib/curated-frames/batch-delete"

describe("deleteCuratedFramesBatch", () => {
  it("deletes each selected frame once in selection order", async () => {
    const remove = vi.fn().mockResolvedValue(undefined)

    const result = await deleteCuratedFramesBatch(["a", "b", "a", "c"], remove)

    expect(result).toEqual({ ok: true, deletedIds: ["a", "b", "c"] })
    expect(remove).toHaveBeenNthCalledWith(1, "a")
    expect(remove).toHaveBeenNthCalledWith(2, "b")
    expect(remove).toHaveBeenNthCalledWith(3, "c")
  })

  it("stops at the first failed delete and reports already deleted ids", async () => {
    const error = new Error("boom")
    const remove = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(error)

    const result = await deleteCuratedFramesBatch(["a", "b", "c"], remove)

    expect(result).toEqual({ ok: false, deletedIds: ["a"], failedId: "b", error })
    expect(remove).toHaveBeenCalledTimes(2)
  })
})
