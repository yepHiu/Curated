import { mockAIGovernanceService } from "@/services/adapters/mock/mock-ai-governance-service"
import { webAIGovernanceService } from "@/services/adapters/web/web-ai-governance-service"
export const useAIGovernanceService = () => import.meta.env.VITE_USE_WEB_API === "true" ? webAIGovernanceService : mockAIGovernanceService
