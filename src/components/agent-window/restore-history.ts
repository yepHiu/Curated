import type { AIAgentMovieCardDTO, AIChatStoredMessageDTO } from "@/api/types"
import { parsePresentMoviesContent } from "@/lib/agent-movie-cards"
import { isAgentProcessTool } from "@/lib/agent-tool-labels"
import type { AgentChatEntry, AgentProcessTool } from "./types"

export function restoreChatHistory(messages: AIChatStoredMessageDTO[]): AgentChatEntry[] {
  const entries: AgentChatEntry[] = []
  let movies: AIAgentMovieCardDTO[] = []
  let tools: AgentProcessTool[] = []
  const flush = (id: string) => {
    if (tools.length) entries.push({ id: `${id}-process`, kind: "process", tools, thinking: "", thinkingActive: false, open: false })
    tools = []
  }
  for (const message of messages) {
    if (message.role === "user") {
      flush(message.id)
      if (movies.length) entries.push({ id: `${message.id}-previous-cards`, kind: "assistant", content: "", movies })
      movies = []
      entries.push({ id: message.id, kind: "user", content: message.content })
    } else if (message.role === "tool") {
      if (message.toolName === "present_movies") movies = parsePresentMoviesContent(message.content)
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
        if (event.type === "tool_call_result") {
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
      if (message.content || movies.length) entries.push({ id: message.id, kind: "assistant", content: message.content, movies, answerEvidence })
      movies = []
      entries.push(...trailing)
    }
  }
  flush("history-end")
  if (movies.length) entries.push({ id: "history-end-cards", kind: "assistant", content: "", movies })
  return entries
}
