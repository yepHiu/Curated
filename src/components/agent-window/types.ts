export type AgentProcessTool = {
  toolCallId: string
  name: string
  pending: boolean
  ok?: boolean
}

export type AgentChatEntry =
  | { id: string; kind: "user"; content: string }
  | { id: string; kind: "assistant"; content: string; movies?: import("@/api/types").AIAgentMovieCardDTO[] }
  | {
      id: string
      kind: "process"
      thinking: string
      thinkingActive: boolean
      tools: AgentProcessTool[]
      open: boolean
    }
  | {
      id: string
      kind: "confirm"
      name: string
      confirmToken: string
      expiresAt?: string
      changes: import("@/api/types").AIConfirmChangeDTO[]
      arguments: Record<string, unknown>
      sessionId: string
      status: "pending" | "applying" | "applied" | "discarded"
      error?: string
    }
