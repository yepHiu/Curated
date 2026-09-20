export const DEFAULT_AI_CONTEXT_WINDOW = 65_536
export const MIN_AI_CONTEXT_WINDOW = 32_768
export const MAX_AI_CONTEXT_WINDOW = 2_097_152

export function validAIContextWindow(value: number): boolean {
  return Number.isSafeInteger(value) && value >= MIN_AI_CONTEXT_WINDOW && value <= MAX_AI_CONTEXT_WINDOW
}

// Official capacity references checked on 2026-09-21. These fill capacity only;
// a proxy/local deployment may expose a smaller window than the model supports.
// Nominal 1M/200K documentation uses conservative decimal values here.
export const AI_CONTEXT_PRESETS = [
  { id: "deepseek-flash", label: "DeepSeek Flash", tokens: 1_000_000, source: "https://api-docs.deepseek.com/quick_start/pricing" },
  { id: "deepseek-v4-pro", label: "DeepSeek V4 Pro", tokens: 1_000_000, source: "https://api-docs.deepseek.com/quick_start/pricing" },
  { id: "minimax-m3", label: "MiniMax M3", tokens: 1_000_000, source: "https://platform.minimax.io/docs/guides/models-intro" },
  { id: "minimax-m2.7", label: "MiniMax M2.7", tokens: 204_800, source: "https://platform.minimax.io/docs/guides/text-generation" },
  { id: "minimax-m2.5", label: "MiniMax M2.5", tokens: 204_800, source: "https://platform.minimax.io/docs/guides/text-generation" },
  { id: "glm-5", label: "GLM-5", tokens: 200_000, source: "https://docs.z.ai/guides/llm/glm-5" },
  { id: "gpt-5.4", label: "GPT-5.4", tokens: 1_050_000, source: "https://developers.openai.com/api/docs/models/gpt-5.4" },
  { id: "claude-sonnet-5", label: "Claude Sonnet 5", tokens: 1_000_000, source: "https://platform.claude.com/docs/en/about-claude/models/overview" },
  { id: "claude-opus-5", label: "Claude Opus 5", tokens: 1_000_000, source: "https://platform.claude.com/docs/en/about-claude/models/overview" },
] as const
