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
    const promise = mockAIService.streamChat([{ role: "user", content: "你好" }], {
      onDelta: (delta) => deltas.push(delta),
    })
    await vi.advanceTimersByTimeAsync(10_000)
    await promise
    expect(deltas.length).toBeGreaterThan(0)
    expect(deltas.join("")).toContain("你好")
  })

  it("rejects when no user message is present", async () => {
    await expect(
      mockAIService.streamChat([{ role: "system", content: "s" }], { onDelta: () => {} }),
    ).rejects.toThrow(/user/)
  })

  it("resolves without deltas when aborted mid-stream", async () => {
    const controller = new AbortController()
    const deltas: string[] = []
    const promise = mockAIService.streamChat([{ role: "user", content: "hi" }], {
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
