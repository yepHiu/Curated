import { afterEach, describe, expect, it, vi } from "vitest"
import { webAIService } from "./web-ai-service"

vi.mock("@/api/endpoints", () => ({ api: {} }))
vi.mock("@/api/http-client", () => ({ resolveApiBaseUrl: () => "/api" }))

afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })

const input = { messages: [{ role: "user" as const, content: "question" }] }
function response(events: unknown[]) {
  return new Response(events.map(event => `data: ${JSON.stringify(event)}\n\n`).join(""))
}

describe("AI transport termination", () => {
  it("delivers compaction lifecycle without treating it as completion", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response([
      { type: "context_status", context: { phase: "compacting" } },
      { type: "context_status", context: { phase: "unknown" } },
      { type: "context_status", context: { phase: "ready" } },
      { type: "text_delta", delta: "continued answer" },
      { type: "message_done", outcome: { status: "completed" } },
    ])))
    const onContextStatus = vi.fn(), onDelta = vi.fn(), onOutcome = vi.fn()
    await webAIService.streamChat(input, { onContextStatus, onDelta, onOutcome })
    expect(onContextStatus.mock.calls).toEqual([[{ phase: "compacting" }], [{ phase: "ready" }]])
    expect(onDelta).toHaveBeenCalledWith("continued answer")
    expect(onOutcome).toHaveBeenCalledTimes(1)
  })
  it("delivers server progress and published evidence while suppressing raw thinking", async () => {
    const answerEvidence = { version: 1, items: [{ refId: "r1", source: "local", tool: "search_movies", retrievedAt: "2026-09-09T00:00:00Z", fields: { code: "TEST-101" } }] }
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response([
      { type: "answer_progress" }, { type: "thinking_delta", delta: "FAKE-999" },
      { type: "text_delta", delta: "Published record" },
      { type: "message_done", outcome: { status: "completed" }, answerEvidence },
    ])))
    const onThinking = vi.fn(), onAnswerProgress = vi.fn(), onAnswerEvidence = vi.fn(), onDelta = vi.fn()
    await webAIService.streamChat(input, { onThinking, onAnswerProgress, onAnswerEvidence, onDelta })
    expect(onThinking).not.toHaveBeenCalled()
    expect(onAnswerProgress).toHaveBeenCalledOnce()
    expect(onAnswerEvidence).toHaveBeenCalledWith(answerEvidence)
    expect(onDelta).toHaveBeenCalledExactlyOnceWith("Published record")
  })

  it("accepts SSE heartbeats across a long buffered answer without a false idle timeout", async () => {
    vi.useFakeTimers()
    let controller!: ReadableStreamDefaultController<Uint8Array>
    const stream = new ReadableStream<Uint8Array>({ start(c) { controller = c } })
    const encoder = new TextEncoder()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(stream)))
    const onDelta = vi.fn()
    const pending = webAIService.streamChat(input, { onDelta })
    for (let i = 0; i < 8; i++) {
      await vi.advanceTimersByTimeAsync(15_000)
      controller.enqueue(encoder.encode(": keep-alive\n\n"))
      await vi.advanceTimersByTimeAsync(0)
    }
    controller.enqueue(encoder.encode('data: {"type":"text_delta","delta":"checked"}\n\ndata: {"type":"message_done"}\n\n'))
    controller.close()
    await pending
    expect(onDelta).toHaveBeenCalledExactlyOnceWith("checked")
    expect(vi.getTimerCount()).toBe(0)
  })
  it("preserves partial text and reports an EOF without a terminal event", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response([{ type: "text_delta", delta: "partial" }])))
    const onDelta = vi.fn()
    await expect(webAIService.streamChat(input, { onDelta })).rejects.toMatchObject({ code: "AI_STREAM_INTERRUPTED" })
    expect(onDelta).toHaveBeenCalledWith("partial")
  })
  it("accepts completion and preserves structured HTTP errors", async () => {
    const fetch = vi.fn().mockResolvedValueOnce(response([{ type: "message_done", outcome: { status: "completed" } }]))
      .mockResolvedValueOnce(new Response(JSON.stringify({ code: "AUTH_LOCKED", message: "Unlock first" }), { status: 401 }))
    vi.stubGlobal("fetch", fetch)
    const onOutcome = vi.fn()
    await webAIService.streamChat(input, { onDelta: vi.fn(), onOutcome })
    expect(onOutcome).toHaveBeenCalledWith({ status: "completed" })
    await expect(webAIService.streamChat(input, { onDelta: vi.fn() })).rejects.toMatchObject({ code: "AUTH_LOCKED", message: "Unlock first" })
  })
  it("bounds a provider that never sends response headers", async () => {
    vi.useFakeTimers()
    vi.stubGlobal("fetch", vi.fn((_url, options: RequestInit) => new Promise((_resolve, reject) => {
      options.signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")))
    })))
    const pending = expect(webAIService.streamChat(input, { onDelta: vi.fn() })).rejects.toMatchObject({ code: "AI_TIMEOUT" })
    await vi.advanceTimersByTimeAsync(90_000)
    await pending
    expect(vi.getTimerCount()).toBe(0)
  })
  it("aborts action generation without retrying", async () => {
    const controller = new AbortController()
    const fetch = vi.fn((_url, options: RequestInit) => new Promise((_resolve, reject) => {
      options.signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")))
    }))
    vi.stubGlobal("fetch", fetch)
    const pending = expect(webAIService.runAction("translate_title", { movieId: "m1" }, controller.signal)).rejects.toThrow()
    controller.abort()
    await pending
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})
