import type { AIChatMessageDTO } from "@/api/types"
import type { AIChatStreamHandlers, AIService } from "@/services/contracts/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

const MOCK_CHUNK_DELAY_MS = 60

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException("aborted", "AbortError"))
      return
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort)
      resolve()
    }, ms)
    function onAbort() {
      clearTimeout(timer)
      reject(new DOMException("aborted", "AbortError"))
    }
    signal?.addEventListener("abort", onAbort, { once: true })
  })
}

/** Mock 适配：假流式回复，保证 Agent Window 前端在无后端时可独立开发。 */
async function streamChat(messages: AIChatMessageDTO[], handlers: AIChatStreamHandlers) {
  const lastUser = [...messages].reverse().find((m) => m.role === "user")
  if (!lastUser) {
    throw new AIServiceError("messages must include a user message")
  }
  const reply = `[Mock Agent] 收到：「${lastUser.content.slice(0, 120)}」。当前为 Mock 模式假流式回复，连接真实后端并配置 provider 后可对话。`
  const chunks = reply.match(/[\s\S]{1,3}/g) ?? [reply]
  try {
    for (const chunk of chunks) {
      await sleep(MOCK_CHUNK_DELAY_MS, handlers.signal)
      handlers.onDelta(chunk)
    }
  } catch (err) {
    if ((err as DOMException)?.name === "AbortError") return
    throw err
  }
}

export const mockAIService: AIService = { streamChat }
