import { api } from "@/api/endpoints"
import { resolveApiBaseUrl } from "@/api/http-client"
import type {
  AIChatStreamHandlers,
  AIChatStreamRequest,
  AIService,
} from "@/services/contracts/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

const AI_CHAT_ENDPOINT = "/ai/chat"

interface SSEEventPayload {
  type?: string
  delta?: string
  code?: string
  message?: string
  sessionId?: string
  toolCallId?: string
  name?: string
  ok?: boolean
  summary?: string
  truncated?: boolean
  movies?: import("@/api/types").AIAgentMovieCardDTO[]
  confirmToken?: string
  expiresAt?: string
  changes?: import("@/api/types").AIConfirmChangeDTO[]
  arguments?: Record<string, unknown>
}

/**
 * Web 适配：消费 POST /api/ai/chat 的 text/event-stream。
 * 不走 30s 超时的 http-client，用独立 fetch + ReadableStream 逐事件解析。
 */
async function streamChat(input: AIChatStreamRequest, handlers: AIChatStreamHandlers) {
  const base = resolveApiBaseUrl(import.meta.env)
  let resp: Response
  try {
    resp = await fetch(`${base}${AI_CHAT_ENDPOINT}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        messages: input.messages,
        sessionId: input.sessionId,
        context: input.context,
        locale: input.locale,
      }),
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
    if (payload.sessionId) {
      handlers.onSession?.(payload.sessionId)
    }
    switch (payload.type) {
      case "text_delta":
        if (payload.delta) handlers.onDelta(payload.delta)
        break
      case "thinking_delta":
        if (payload.delta) handlers.onThinking?.(payload.delta)
        break
      case "tool_call_started":
        handlers.onToolStart?.({
          toolCallId: payload.toolCallId ?? payload.name ?? "tool",
          name: payload.name ?? "",
        })
        break
      case "tool_call_result":
        handlers.onToolResult?.({
          toolCallId: payload.toolCallId ?? payload.name ?? "tool",
          name: payload.name ?? "",
          ok: payload.ok,
          summary: payload.summary,
          truncated: payload.truncated,
          movies: payload.movies,
        })
        if (payload.movies?.length) {
          handlers.onMovieCards?.(payload.movies)
        }
        break
      case "movie_cards":
        if (payload.movies?.length) {
          handlers.onMovieCards?.(payload.movies)
        }
        break
      case "confirm_required":
        if (payload.confirmToken && payload.name) {
          handlers.onConfirmRequired?.({
            toolCallId: payload.toolCallId ?? payload.name,
            name: payload.name,
            confirmToken: payload.confirmToken,
            expiresAt: payload.expiresAt,
            changes: payload.changes ?? [],
            arguments: payload.arguments ?? {},
            sessionId: payload.sessionId,
          })
        }
        break
      case "error":
        throw new AIServiceError(payload.message ?? "chat failed", payload.code)
      default:
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

export const webAIService: AIService = {
  streamChat,
  listSessions: async () => (await api.listAIChatSessions()).items ?? [],
  createSession: (title) => api.createAIChatSession(title),
  getSession: (id) => api.getAIChatSession(id),
  deleteSession: (id) => api.deleteAIChatSession(id),
  runAction: (name, body) => api.runAIAction(name, body),
  confirmTool: (body) => api.confirmAITool(body),
}
