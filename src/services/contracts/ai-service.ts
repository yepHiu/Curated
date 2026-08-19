import type { AIChatMessageDTO } from "@/api/types"

/** Agent 聊天流式回调；onDelta 在流式期间被同步多次调用 */
export interface AIChatStreamHandlers {
  onDelta(delta: string): void
  signal?: AbortSignal
}

/**
 * 实验性 Agent 服务契约（E1：纯流式对话，无工具调用）。
 * Web 走 POST /api/ai/chat 的 SSE；Mock 返回假流式以便前端独立开发。
 */
export interface AIService {
  /**
   * 发送一轮对话并流式接收回复；流正常结束或出错后 resolve/reject。
   * 服务端未配置 provider 时以 AI_PROVIDER_UNAVAILABLE 错误码 reject。
   */
  streamChat(messages: AIChatMessageDTO[], handlers: AIChatStreamHandlers): Promise<void>
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
