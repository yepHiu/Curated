import { describe, expect, it } from "vitest"
import { restoreChatHistory } from "./restore-history"
import type { AIChatStoredMessageDTO } from "@/api/types"

const base = { id: "reply", sessionId: "s1", role: "assistant", seq: 2, createdAt: "", content: "partial answer" }

describe("persisted AI turns", () => {
  it("restores failed tools, candidates and terminal outcome without reviving write authority", () => {
    const message: AIChatStoredMessageDTO = { ...base, events: [
      { type: "tool_call_result", name: "search_movies", toolCallId: "t1", ok: false },
      { type: "confirm_required", name: "save_movie_comment", changes: [{ path: "comment.body", before: "old", after: "new" }] },
      { type: "message_done", outcome: { status: "cancelled", reason: "stopped" } },
    ] }
    const entries = restoreChatHistory([message])
    expect(entries.find(e => e.kind === "process")).toMatchObject({ tools: [{ ok: false, pending: false }] })
    expect(entries.find(e => e.kind === "assistant")).toMatchObject({ content: "partial answer" })
    expect(entries.find(e => e.kind === "confirm")).toMatchObject({ status: "archived", confirmToken: "", arguments: {} })
    expect(entries.at(-1)).toMatchObject({ kind: "outcome", outcome: { status: "cancelled" } })
  })

  it("does not invent success for legacy summaries", () => {
    const entries = restoreChatHistory([{ ...base, role: "tool", toolName: "search_movies" }])
    expect(entries[0]).toMatchObject({ kind: "process", tools: [{ pending: false }] })
    if (entries[0]?.kind === "process") expect(entries[0].tools[0]?.ok).toBeUndefined()
  })
})
