import { api } from "@/api/endpoints"
import { resolveApiBaseUrl } from "@/api/http-client"
import type {
  AIChatStreamHandlers,
  AIChatStreamRequest,
  AIService,
} from "@/services/contracts/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

const AI_CHAT_ENDPOINT = "/ai/chat"

function requestDeadline(parent: AbortSignal | undefined, milliseconds: number) {
  const controller = new AbortController()
  let timedOut = false
  let timer: ReturnType<typeof setTimeout>
  const reset = () => {
    clearTimeout(timer)
    timer = setTimeout(() => { timedOut = true; controller.abort() }, milliseconds)
  }
  const abort = () => controller.abort()
  if (parent?.aborted) abort()
  else parent?.addEventListener("abort", abort, { once: true })
  reset()
  return {
    signal: controller.signal, reset,
    check(error: unknown): never {
      if (timedOut) throw new AIServiceError("AI request timed out. Please try again.", "AI_TIMEOUT")
      throw error
    },
    dispose() { clearTimeout(timer); parent?.removeEventListener("abort", abort) },
  }
}

interface SSEEventPayload {
  answerEvidence?: import("@/api/types").AIAnswerEvidenceDTO
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
  books?: import("@/api/types").AIAgentBookCardDTO[]
  providerRows?: import("@/api/types").AIAgentProviderTitleDTO[]
  confirmToken?: string
  expiresAt?: string
  changes?: import("@/api/types").AIConfirmChangeDTO[]
  arguments?: Record<string, unknown>
  evidence?: import("@/api/types").AIEvidenceDTO
  resolution?: import("@/api/types").AIEntityResolutionDTO
  outcome?: import("@/api/types").AIChatOutcomeDTO
}

/**
 * Web 适配：消费 POST /api/ai/chat 的 text/event-stream。
 * 不走 30s 超时的 http-client，用独立 fetch + ReadableStream 逐事件解析。
 */
async function streamChat(input: AIChatStreamRequest, handlers: AIChatStreamHandlers) {
  const deadline = requestDeadline(handlers.signal, 90_000)
  try {
    await consumeChat(input, { ...handlers, signal: deadline.signal }, deadline.reset)
  } catch (err) {
    return deadline.check(err)
  } finally {
    deadline.dispose()
  }
}

async function consumeChat(input: AIChatStreamRequest, handlers: AIChatStreamHandlers, received: () => void) {
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
    if (handlers.signal?.aborted) throw err
    throw new AIServiceError((err as Error).message ?? "network error")
  }

  if (!resp.ok || !resp.body) {
    const error = await resp.json().catch(() => ({})) as { code?: string; message?: string }
    throw new AIServiceError(error.message || `HTTP ${resp.status}`, error.code)
  }

  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""
  let completed = false

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
      case "answer_progress":
        handlers.onAnswerProgress?.()
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
          books: payload.books,
          providerRows: payload.providerRows,
          evidence: payload.evidence,
          resolution: payload.resolution,
        })
        if (payload.movies?.length) {
          handlers.onMovieCards?.(payload.movies)
        }
        if (payload.books?.length) {
          handlers.onBookCards?.(payload.books)
        }
        break
      case "message_done":
        completed = true
        if (payload.answerEvidence) handlers.onAnswerEvidence?.(payload.answerEvidence)
        if (payload.outcome) handlers.onOutcome?.(payload.outcome)
        break
      case "movie_cards":
        if (payload.movies?.length) {
          handlers.onMovieCards?.(payload.movies)
        }
        break
      case "book_cards":
        if (payload.books?.length) {
          handlers.onBookCards?.(payload.books)
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

  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      received()
      buffer += decoder.decode(value, { stream: true })
      buffer = buffer.replace(/\r\n/g, "\n")
      for (;;) {
        const sep = buffer.indexOf("\n\n")
        if (sep < 0) break
        const block = buffer.slice(0, sep)
        buffer = buffer.slice(sep + 2)
        handleEventBlock(block)
      }
    }
    if (buffer.trim()) handleEventBlock(buffer)
    if (!completed) throw new AIServiceError("The AI response was interrupted. You can send the request again.", "AI_STREAM_INTERRUPTED")
  } finally {
    await reader.cancel().catch(() => undefined)
    reader.releaseLock()
  }
}

async function postJSON<T>(path: string, body: unknown, signal?: AbortSignal): Promise<T> {
  const deadline = requestDeadline(signal, 120_000)
  try {
    return await fetchJSON<T>(path, body, deadline.signal)
  } catch (err) {
    return deadline.check(err)
  } finally {
    deadline.dispose()
  }
}

async function fetchJSON<T>(path: string, body: unknown, signal: AbortSignal): Promise<T> {
  const base = resolveApiBaseUrl(import.meta.env)
  let resp: Response
  try {
    resp = await fetch(`${base}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      credentials: "include",
      body: JSON.stringify(body),
      signal,
    })
  } catch (err) {
    throw new AIServiceError((err as Error).message ?? "network error")
  }
  const text = await resp.text()
  let parsed: unknown
  if (text.trim()) {
    try {
      parsed = JSON.parse(text) as unknown
    } catch {
      parsed = undefined
    }
  }
  if (!resp.ok) {
    const apiErr = parsed as { code?: string; message?: string } | undefined
    throw new AIServiceError(apiErr?.message || `HTTP ${resp.status}`, apiErr?.code)
  }
  return parsed as T
}

export const webAIService: AIService = {
  streamChat,
  listSessions: async () => (await api.listAIChatSessions()).items ?? [],
  createSession: (title) => api.createAIChatSession(title),
  getSession: (id, cursor) => api.getAIChatSession(id, cursor),
  deleteSession: (id) => api.deleteAIChatSession(id),
  runAction: (name, body, signal) => postJSON(`/ai/actions/${encodeURIComponent(name)}`, body, signal),
  confirmTool: (body) => postJSON("/ai/confirm", body),
}
