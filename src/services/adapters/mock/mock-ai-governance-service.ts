import { defaultAIGovernance, type AIGovernanceService, type AIGovernanceSettings } from "@/services/contracts/ai-governance-service"
const key = "curated-ai-governance-mock-v1"
function settings(): AIGovernanceSettings {
  try { return { ...defaultAIGovernance(), ...JSON.parse(localStorage.getItem(key) ?? "{}") } }
  catch { return defaultAIGovernance() }
}

/** Mock has no measured provider calls. Never invent consumption or failure data. */
export const mockAIGovernanceService: AIGovernanceService = {
  async getSettings() { return settings() },
  async saveSettings(value) { localStorage.setItem(key, JSON.stringify(value)); return { ...value } },
  async getUsage(query) {
    return { items: [], total: 0, offset: query.offset ?? 0, limit: query.limit ?? 25,
      summary: { runs: 0, failed: 0, partial: 0, cancelled: 0, modelCalls: 0, usageCalls: 0,
        toolCalls: 0, promptTokens: 0, completionTokens: 0, totalTokens: 0, avgDurationMs: null, avgFirstTextMs: null } }
  },
  async getAudit(query) { return { items: [], total: 0, offset: query.offset ?? 0, limit: query.limit ?? 25 } },
  async cleanup() { return { runs: 0, audit: 0, receipts: 0 } },
}
