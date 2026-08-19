import { resolveApiBaseUrl } from "@/api/http-client"
import type { AIChatMessageDTO } from "@/api/types"
import type { AIChatStreamHandlers, AIService } from "@/services/contracts/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

const AI_CHAT_ENDPOINT = "/ai/chat"

interface SSEEventPayload {
  type?: string
  delta?: string
  code?: string
  message?: string
}

/**
 * Web 适配：消费 POST /api/ai/chat 的 text/event-stream。
 * 不走 30s 超时的 http-client，用独立 fetch + ReadableStream 逐事件解析。
 */
async function streamChat(messages: AIChatMessageDTO[], handlers: AIChatStreamHandlers) {
  const base = resolveApiBaseUrl(import.meta.env)
  let resp: Response
  try {
    resp = await fetch(`${base}${AI_CHAT_ENDPOINT}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ messages }),
      signal: handlers.signal,
    })
  } catch (err) {
    if (handlers.signal?.aborted) return
    throw new AIServiceError((err as Error).message ?? "network error")
  }

  if (!resp.ok || !resp.body) {
    throw new AIServiceError(`HTTP ${resp.status}`)
  }

  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  const handleEventBlock = (block: string) => {
    const lines = block.split("\n")
    let data = ""
    for (const line of lines) {
      if (line.startsWith("data:")) {
        data += (data ? "\n" : "") + line.slice(5).trim()
      }
    }
    if (!data) return
    let payload: SSEEventPayload
    try {
      payload = JSON.parse(data) as SSEEventPayload
    } catch {
      return
    }
    switch (payload.type) {
      case "text_delta":
        if (payload.delta) handlers.onDelta(payload.delta)
        break
      case "error":
        throw new AIServiceError(payload.message ?? "chat failed", payload.code)
      default:
        // message_start / message_done：无需前端动作
        break
    }
  }

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    for (;;) {
      const sep = buffer.indexOf("\n\n")
      if (sep < 0) break
      const block = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      handleEventBlock(block)
    }
  }
  if (buffer.trim()) handleEventBlock(buffer)
}

export const webAIService: AIService = { streamChat }
