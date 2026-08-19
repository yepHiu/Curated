import { mockAIService } from "@/services/adapters/mock/mock-ai-service"
import { webAIService } from "@/services/adapters/web/web-ai-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

export const useAIService = () => (USE_WEB ? webAIService : mockAIService)
