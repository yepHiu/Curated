import type {
  AIActionPreviewDTO,
  AIActionRequestBody,
  AIAgentMovieCardDTO,
  AIChatContextDTO,
  AIChatMessageDTO,
  AIChatSessionDetailDTO,
  AIChatSessionDTO,
  AIConfirmChangeDTO,
  AIToolApplyDTO,
  AIToolApplyRequestBody,
} from "@/api/types"

/** Agent 聊天流式回调；onDelta 在流式期间被同步多次调用 */
export interface AIChatStreamRequest {
  messages: AIChatMessageDTO[]
  sessionId?: string
  context?: AIChatContextDTO
  locale?: string
}

export interface AIChatToolEvent {
  toolCallId: string
  name: string
  ok?: boolean
  summary?: string
  truncated?: boolean
  movies?: AIAgentMovieCardDTO[]
}

export interface AIChatConfirmEvent {
  toolCallId: string
  name: string
  confirmToken: string
  expiresAt?: string
  changes: AIConfirmChangeDTO[]
  arguments: Record<string, unknown>
  sessionId?: string
}

export interface AIChatStreamHandlers {
  onDelta(delta: string): void
  onToolStart?(event: AIChatToolEvent): void
  onToolResult?(event: AIChatToolEvent): void
  onMovieCards?(movies: AIAgentMovieCardDTO[]): void
  onConfirmRequired?(event: AIChatConfirmEvent): void
  onSession?(sessionId: string): void
  onThinking?(delta: string): void
  signal?: AbortSignal
}

/**
 * 实验性 Agent 服务契约（E3：只读工具 + 会话 + 笔记 Action 确认写）。
 * Web 走 POST /api/ai/chat 的 SSE；Mock 返回假流式与假工具结果。
 */
export interface AIService {
  streamChat(input: AIChatStreamRequest, handlers: AIChatStreamHandlers): Promise<void>
  listSessions(): Promise<AIChatSessionDTO[]>
  createSession(title?: string): Promise<AIChatSessionDTO>
  getSession(id: string): Promise<AIChatSessionDetailDTO>
  deleteSession(id: string): Promise<void>
  runAction(name: string, body: AIActionRequestBody): Promise<AIActionPreviewDTO>
  confirmTool(body: AIToolApplyRequestBody): Promise<AIToolApplyDTO>
}

/** AI 服务错误；code 对齐后端错误码（如 AI_PROVIDER_UNAVAILABLE / AI_CHAT_FAILED） */
export class AIServiceError extends Error {
  readonly code?: string

  constructor(message: string, code?: string) {
    super(message)
    this.name = "AIServiceError"
    this.code = code
  }
}
