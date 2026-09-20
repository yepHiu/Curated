import type { AIAgentBookCardDTO, AIAgentMovieCardDTO, AIChatStoredMessageDTO } from "@/api/types"
import { parsePresentBooksContent } from "@/lib/agent-book-cards"
import { parsePresentMoviesContent } from "@/lib/agent-movie-cards"
import { isAgentProcessTool } from "@/lib/agent-tool-labels"
import type { AgentChatEntry, AgentProcessTool } from "./types"

/** 把已保存的会话还原为窗口条目；影片卡与书卡分开投影，不授予新的写权限或引用。 */
export function restoreChatHistory(messages: AIChatStoredMessageDTO[]): AgentChatEntry[] {
  const entries: AgentChatEntry[] = []
  let movies: AIAgentMovieCardDTO[] = []
  let books: AIAgentBookCardDTO[] = []
  let tools: AgentProcessTool[] = []
  let contextPhase: import("@/api/types").AIContextStatusDTO["phase"] | undefined
  /** 把尚未挂到助手气泡上的过程工具冲刷成一条 process 条目。 */
  const flush = (id: string) => {
    if (tools.length || contextPhase) entries.push({ id: `${id}-process`, kind: "process", tools, contextPhase: contextPhase === "compacting" ? "limited" : contextPhase, thinking: "", thinkingActive: false, open: false })
    tools = []
    contextPhase = undefined
  }
  /** 把尚未挂到助手气泡上的影片卡或书卡冲刷成一条 assistant 条目。 */
  const flushCards = (id: string) => {
    if (movies.length || books.length) {
      entries.push({ id, kind: "assistant", content: "", movies: movies.length ? movies : undefined, books: books.length ? books : undefined })
    }
    movies = []
    books = []
  }
  for (const message of messages) {
    if (message.role === "user") {
      flush(message.id)
      flushCards(`${message.id}-previous-cards`)
      entries.push({ id: message.id, kind: "user", content: message.content })
    } else if (message.role === "tool") {
      if (message.toolName === "present_movies") movies = parsePresentMoviesContent(message.content)
      if (message.toolName === "present_comics" || message.toolName === "present_photos") {
        books = parsePresentBooksContent(message.content)
      }
      if (isAgentProcessTool(message.toolName || "")) {
        // Legacy summaries do not contain a reliable success flag.
        tools.push({ toolCallId: message.toolCallId || message.id, name: message.toolName || "tool", pending: false })
      }
    } else if (message.role === "assistant") {
      const trailing: AgentChatEntry[] = []
      let answerEvidence: import("@/api/types").AIAnswerEvidenceDTO | undefined
      for (const [index, event] of (message.events ?? []).entries()) {
        const id = `${message.id}-event-${index}`
        if (event.movies?.length) movies = event.movies
        if (event.books?.length) books = event.books
        if (event.type === "context_status" && event.context) {
          contextPhase = event.context.phase
        } else if (event.type === "tool_call_result") {
          if (isAgentProcessTool(event.name || "")) tools.push({
            toolCallId: event.toolCallId || id, name: event.name || "tool", pending: false,
            ok: event.ok, evidence: event.evidence, providerRows: event.providerRows,
          })
          if (event.resolution && event.resolution.status !== "matched") trailing.push({ id, kind: "resolution", resolution: event.resolution })
        } else if (event.type === "message_done" && event.outcome) {
          answerEvidence = event.answerEvidence
          trailing.push({ id, kind: "outcome", outcome: event.outcome })
        } else if (event.type === "confirm_required") {
          trailing.push({ id, kind: "confirm", name: event.name || "", changes: event.changes ?? [],
            confirmToken: "", arguments: {}, sessionId: message.sessionId, status: event.applied ? "applied" : "archived" })
        }
      }
      flush(message.id)
      if (message.content || movies.length || books.length) {
        entries.push({
          id: message.id,
          kind: "assistant",
          content: message.content,
          movies: movies.length ? movies : undefined,
          books: books.length ? books : undefined,
          answerEvidence,
        })
      }
      movies = []
      books = []
      entries.push(...trailing)
    }
  }
  flush("history-end")
  flushCards("history-end-cards")
  return entries
}
