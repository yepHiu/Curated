export type AgentProcessTool = {
  toolCallId: string
  name: string
  pending: boolean
  ok?: boolean
  evidence?: import("@/api/types").AIEvidenceDTO
  providerRows?: import("@/api/types").AIAgentProviderTitleDTO[]
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
      kind: "resolution"
      resolution: import("@/api/types").AIEntityResolutionDTO
      selected?: boolean
    }
  | { id: string; kind: "outcome"; outcome: import("@/api/types").AIChatOutcomeDTO }
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
