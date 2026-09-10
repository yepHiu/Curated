import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { mockAIService } from "./mock-ai-service"

describe("mockAIService.streamChat", () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("streams fake deltas containing the user message", async () => {
    const deltas: string[] = []
    const promise = mockAIService.streamChat({ messages: [{ role: "user", content: "你好" }] }, {
      onDelta: (delta) => deltas.push(delta),
    })
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(deltas.length).toBeGreaterThan(0)
    expect(deltas.join("")).toContain("你好")
  })

  it("rejects when no user message is present", async () => {
    await expect(
      mockAIService.streamChat({ messages: [{ role: "system", content: "s" }] }, { onDelta: () => {} }),
    ).rejects.toThrow(/user/)
  })

  it.each(["推荐漫画库里的内容", "统计写真库", "search photo books", "recommend manga"])("does not fabricate movie conclusions for %s", async (content) => {
    const tools = vi.fn()
    const cards = vi.fn()
    const deltas: string[] = []
    const promise = mockAIService.streamChat({ messages: [{ role: "user", content }] }, {
      onDelta: (delta) => deltas.push(delta), onToolStart: tools, onMovieCards: cards,
    })
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(tools).not.toHaveBeenCalled()
    expect(cards).not.toHaveBeenCalled()
    expect(deltas.join("")).toContain("仅支持影片相关数据")
  })

  it("emits a fake tool card for a library question", async () => {
    const tools: string[] = []
    const promise = mockAIService.streamChat(
      { messages: [{ role: "user", content: "这个月看了多久" }] },
      {
        onDelta: () => {},
        onToolStart: (event) => tools.push(event.name),
        onToolResult: (event) => tools.push(`${event.name}:${event.summary}`),
      },
    )
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(tools[0]).toBe("get_insights_overview")
    expect(tools[1]).toContain("watchedSeconds")
  })

  it("emits a fake provider title lookup for related works", async () => {
    const tools: string[] = []
    const promise = mockAIService.streamChat(
      { messages: [{ role: "user", content: "她还拍过什么" }] },
      {
        onDelta: () => {},
        onToolStart: (event) => tools.push(event.name),
        onToolResult: (event) => tools.push(`${event.name}:${event.summary}`),
      },
    )
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(tools[0]).toBe("search_provider_titles")
    expect(tools[1]).toContain("inLibrary")
  })

  it("emits a fake source page read", async () => {
    const tools: string[] = []
    const promise = mockAIService.streamChat(
      { messages: [{ role: "user", content: "读一下源站页长评" }] },
      {
        onDelta: () => {},
        onToolStart: (event) => tools.push(event.name),
      },
    )
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(tools[0]).toBe("get_source_page")
  })

  it("emits movie cards for a picker question", async () => {
    const movies: { movieId: string }[] = []
    const promise = mockAIService.streamChat(
      { messages: [{ role: "user", content: "今晚看什么" }] },
      {
        onDelta: () => {},
        onMovieCards: (items) => movies.push(...items),
      },
    )
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(movies[0]?.movieId).toBe("mock-movie-1")
  })

  it("returns a fake comment polish preview", async () => {
    const preview = await mockAIService.runAction("polish_comment", {
      movieId: "m1",
      body: "slow ending",
    })
    expect(preview.confirmToken).toBeTruthy()
    expect(preview.proposedText).toContain("slow ending")
    const applied = await mockAIService.confirmTool({
      sessionId: preview.sessionId,
      name: preview.name,
      arguments: preview.arguments ?? {},
      confirmToken: preview.confirmToken ?? "",
    })
    expect(applied.ok).toBe(true)
  })

  it("returns a fake summary translation preview", async () => {
    const preview = await mockAIService.runAction("translate_summary", { movieId: "m1", body: "Old plot." })
    expect(preview.name).toBe("update_movie_display_overrides")
    expect(preview.proposedText).toContain("Localized")
    expect(preview.arguments).toMatchObject({ userSummary: expect.any(String) })
  })

  it("returns a fake insights narrative without a confirm token", async () => {
    const preview = await mockAIService.runAction("insights_narrative", { range: "30d", timezone: "UTC" })
    expect(preview.noop).toBe(true)
    expect(preview.proposedText).toContain("full-per-entity")
  })

  it("resolves without deltas when aborted mid-stream", async () => {
    const controller = new AbortController()
    const deltas: string[] = []
    const promise = mockAIService.streamChat({ messages: [{ role: "user", content: "hi" }] }, {
      onDelta: (delta) => {
        deltas.push(delta)
        controller.abort()
      },
      signal: controller.signal,
    })
    await vi.advanceTimersByTimeAsync(10_000)
    await expect(promise).resolves.toBeUndefined()
  })
})
