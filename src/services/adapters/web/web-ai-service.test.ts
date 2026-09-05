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
