import { describe, expect, it } from "vitest"
import { restoreChatHistory } from "./restore-history"
import type { AIChatStoredMessageDTO } from "@/api/types"

const base = { id: "reply", sessionId: "s1", role: "assistant", seq: 2, createdAt: "", content: "partial answer" }

describe("persisted AI turns", () => {
  it("restores the published source snapshot without granting a new reference", () => {
    const answerEvidence = { version: 1 as const, items: [{ refId: "expired", source: "provider" as const, tool: "search_provider_titles", retrievedAt: "2026-09-09T00:00:00Z", fields: { code: "TEST-101" } }] }
    const entries = restoreChatHistory([{ ...base, events: [{ type: "message_done", outcome: { status: "completed" }, answerEvidence }] }])
    expect(entries.find(e => e.kind === "assistant")).toMatchObject({ answerEvidence })
    expect(entries.some(e => e.kind === "confirm")).toBe(false)
    expect(restoreChatHistory([base]).find(e => e.kind === "assistant")).toMatchObject({ answerEvidence: undefined })
  })
  it("restores committed confirmations without granting write authority", () => {
    const entries = restoreChatHistory([{ ...base, events: [
      { type: "confirm_required", name: "save_movie_comment", receiptId: "hash", applied: true },
    ] }])
    expect(entries.find(e => e.kind === "confirm")).toMatchObject({ status: "applied", confirmToken: "", arguments: {} })
  })
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

  it("restores comic and photo cards without treating them as movie cards", () => {
    const entries = restoreChatHistory([
      { ...base, role: "tool", toolName: "present_comics", content: JSON.stringify({ books: [{ kind: "comic", comicId: "c1", title: "Summer" }] }) },
      { ...base, id: "reply-2", content: "here", events: [{ type: "message_done", outcome: { status: "completed" } }] },
    ])
    const assistant = entries.find((entry) => entry.kind === "assistant")
    expect(assistant).toMatchObject({
      books: [{ kind: "comic", comicId: "c1", title: "Summer" }],
    })
    if (assistant?.kind === "assistant") expect(assistant.movies).toBeUndefined()
  })
})
