import type { AIActionPreviewDTO, AIChatSessionDTO, AIChatStoredMessageDTO, AIToolApplyDTO } from "@/api/types"
import type {
  AIChatStreamHandlers,
  AIChatStreamRequest,
  AIService,
} from "@/services/contracts/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

const MOCK_CHUNK_DELAY_MS = 60

type MockSession = AIChatSessionDTO & { messages: AIChatStoredMessageDTO[] }

const sessions: MockSession[] = []
let seq = 0

function nowIso() {
  return new Date().toISOString()
}

function nextId(prefix: string) {
  seq += 1
  return `${prefix}${seq}`
}

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

function ensureSession(id?: string, title?: string): MockSession {
  if (id) {
    const found = sessions.find((item) => item.id === id)
    if (found) return found
  }
  const created: MockSession = {
    id: nextId("ses_mock_"),
    title: title ?? "",
    createdAt: nowIso(),
    updatedAt: nowIso(),
    messages: [],
  }
  sessions.unshift(created)
  return created
}

function fakeToolFor(content: string): { name: string; summary: string } | null {
  if (/看了|多久|统计|insight/i.test(content)) {
    return { name: "get_insights_overview", summary: "watchedSeconds: 3600, startedMovies: 4" }
  }
  if (/演员|actor/i.test(content)) {
    return { name: "list_actors", summary: "total: 3, items truncated" }
  }
  if (/推荐|今晚|找片|search|未看/i.test(content)) {
    return { name: "search_movies", summary: "total: 5, playState=unplayed" }
  }
  return null
}

const MOCK_MOVIE_CARDS = [
  {
    movieId: "mock-movie-1",
    title: "Sample Title",
    code: "ABC-123",
    actors: ["Mock Actor"],
    reason: "轻松、时长合适",
  },
]

/** Mock 适配：假流式回复，并可按关键词给出假工具卡。 */
async function streamChat(input: AIChatStreamRequest, handlers: AIChatStreamHandlers) {
  const lastUser = [...input.messages].reverse().find((m) => m.role === "user")
  if (!lastUser) {
    throw new AIServiceError("messages must include a user message")
  }
  const session = ensureSession(input.sessionId, lastUser.content.slice(0, 40))
  handlers.onSession?.(session.id)
  session.messages.push({
    id: nextId("msg_"),
    sessionId: session.id,
    role: "user",
    content: lastUser.content,
    seq: session.messages.length + 1,
    createdAt: nowIso(),
  })

  const tool = fakeToolFor(lastUser.content)
  if (tool) {
    handlers.onThinking?.("先从资料库核对一下。")
    await sleep(MOCK_CHUNK_DELAY_MS, handlers.signal)
    const toolCallId = nextId("call_")
    handlers.onToolStart?.({ toolCallId, name: tool.name })
    await sleep(MOCK_CHUNK_DELAY_MS, handlers.signal)
    handlers.onToolResult?.({ toolCallId, name: tool.name, ok: true, summary: tool.summary })
    session.messages.push({
      id: nextId("msg_"),
      sessionId: session.id,
      role: "tool",
      content: tool.summary,
      toolName: tool.name,
      toolCallId,
      seq: session.messages.length + 1,
      createdAt: nowIso(),
    })
    if (tool.name === "search_movies") {
      const presentId = nextId("call_")
      handlers.onToolStart?.({ toolCallId: presentId, name: "present_movies" })
      await sleep(MOCK_CHUNK_DELAY_MS, handlers.signal)
      handlers.onToolResult?.({
        toolCallId: presentId,
        name: "present_movies",
        ok: true,
        summary: "1 movies",
        movies: MOCK_MOVIE_CARDS,
      })
      handlers.onMovieCards?.(MOCK_MOVIE_CARDS)
      session.messages.push({
        id: nextId("msg_"),
        sessionId: session.id,
        role: "tool",
        content: JSON.stringify({ movies: MOCK_MOVIE_CARDS }),
        toolName: "present_movies",
        toolCallId: presentId,
        seq: session.messages.length + 1,
        createdAt: nowIso(),
      })
    }
  }

  const reply = tool
    ? `[Mock Agent] 已用 ${tool.name} 查库（假数据）：${tool.summary}。连接真实后端后会返回资料库数字。`
    : `[Mock Agent] 收到：「${lastUser.content.slice(0, 120)}」。当前为 Mock 模式假流式回复。`
  const chunks = reply.match(/[\s\S]{1,3}/g) ?? [reply]
  try {
    let full = ""
    for (const chunk of chunks) {
      await sleep(MOCK_CHUNK_DELAY_MS, handlers.signal)
      handlers.onDelta(chunk)
      full += chunk
    }
    session.messages.push({
      id: nextId("msg_"),
      sessionId: session.id,
      role: "assistant",
      content: full,
      seq: session.messages.length + 1,
      createdAt: nowIso(),
    })
    session.updatedAt = nowIso()
  } catch (err) {
    if ((err as DOMException)?.name === "AbortError") return
    throw err
  }
}

export const mockAIService: AIService = {
  streamChat,
  async listSessions() {
    return sessions.map(({ messages: _messages, ...dto }) => dto)
  },
  async createSession(title) {
    return ensureSession(undefined, title)
  },
  async getSession(id) {
    const found = sessions.find((item) => item.id === id)
    if (!found) {
      throw new AIServiceError("session not found", "COMMON_NOT_FOUND")
    }
    return found
  },
  async deleteSession(id) {
    const index = sessions.findIndex((item) => item.id === id)
    if (index >= 0) sessions.splice(index, 1)
  },
  async runAction(name, body): Promise<AIActionPreviewDTO> {
    if (name === "insights_narrative") {
      return {
        action: name,
        name: "insights_narrative",
        sessionId: nextId("act_mock_"),
        proposedText: "Watch time is up and completion is mixed. Actor ranking uses full-per-entity shares.",
        noop: true,
      }
    }
    if (name === "clean_summary" || name === "translate_title") {
      const original = name === "clean_summary" ? "Visit ads.example for the plot." : (body.body ?? "Sample Title")
      const proposed = name === "clean_summary" ? "Clean plot only." : `Localized: ${original}`
      const field = name === "clean_summary" ? "userSummary" : "userTitle"
      return {
        action: name,
        name: "update_movie_display_overrides",
        sessionId: nextId("act_mock_"),
        originalText: original,
        proposedText: proposed,
        confirmToken: nextId("cfm_mock_"),
        arguments: { movieId: body.movieId, [field]: proposed },
        changes: [{ path: `display.${field}`, before: original, after: proposed }],
      }
    }
    const original = (body.body ?? "").trim() || "saved note"
    const proposed = `Polished: ${original}`
    return {
      action: name,
      name: "save_movie_comment",
      sessionId: nextId("act_mock_"),
      originalText: original,
      proposedText: proposed,
      confirmToken: nextId("cfm_mock_"),
      arguments: { movieId: body.movieId, body: proposed },
      changes: [{ path: "comment.body", before: original, after: proposed }],
    }
  },
  async confirmTool(body): Promise<AIToolApplyDTO> {
    const proposed =
      typeof body.arguments.body === "string" ? body.arguments.body : "saved"
    return { ok: true, name: body.name, data: { body: proposed, updatedAt: nowIso() } }
  },
}
