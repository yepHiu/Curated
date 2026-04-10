import { describe, expect, it, vi } from "vitest"
import {
  commitCuratedFrameTags,
  shouldCommitCuratedFrameTagDraft,
  shouldShowCuratedFrameTagRetry,
} from "@/lib/curated-frames/p2-state"

describe("curated frame P2 state helpers", () => {
  it("skips tag updates when draft equals last saved tags", async () => {
    const update = vi.fn()
    const result = await commitCuratedFrameTags({
      frameId: "frame-1",
      tags: ["a"],
      lastSavedTags: ["a"],
      update,
    })
    expect(update).not.toHaveBeenCalled()
    expect(result).toEqual({ ok: true, status: "idle", lastSavedTags: ["a"] })
  })

  it("keeps last saved tags when tag update fails", async () => {
    const update = vi.fn().mockRejectedValue(new Error("boom"))
    const result = await commitCuratedFrameTags({
      frameId: "frame-1",
      tags: ["a", "b"],
      lastSavedTags: ["a"],
      update,
    })
    expect(update).toHaveBeenCalledWith("frame-1", ["a", "b"])
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.status).toBe("error")
      expect(result.lastSavedTags).toEqual(["a"])
      expect(result.error).toBeInstanceOf(Error)
    }
  })

  it("queues automatic tag save only when the draft changed and no save is in flight", () => {
    expect(shouldCommitCuratedFrameTagDraft({
      tags: ["a", "b"],
      lastSavedTags: ["a"],
      saveInFlight: false,
    })).toBe(true)

    expect(shouldCommitCuratedFrameTagDraft({
      tags: ["a"],
      lastSavedTags: ["a"],
      saveInFlight: false,
    })).toBe(false)

    expect(shouldCommitCuratedFrameTagDraft({
      tags: ["a", "b"],
      lastSavedTags: ["a"],
      saveInFlight: true,
    })).toBe(false)
  })

  it("keeps manual tag actions limited to failed autosave retries", () => {
    expect(shouldShowCuratedFrameTagRetry("dirty")).toBe(false)
    expect(shouldShowCuratedFrameTagRetry("saving")).toBe(false)
    expect(shouldShowCuratedFrameTagRetry("saved")).toBe(false)
    expect(shouldShowCuratedFrameTagRetry("error")).toBe(true)
  })
})
