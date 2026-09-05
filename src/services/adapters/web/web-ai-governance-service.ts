import { httpClient } from "@/api/http-client"
import type { AIGovernanceService } from "@/services/contracts/ai-governance-service"

export const webAIGovernanceService: AIGovernanceService = {
  getSettings: () => httpClient.get("/ai/settings"),
  saveSettings: (value) => httpClient.patch("/ai/settings", value),
  getUsage: (query) => httpClient.get("/ai/usage", { ...query }),
  getAudit: (query) => httpClient.get("/ai/audit", { ...query }),
  cleanup: () => httpClient.post("/ai/cleanup", {}),
}
