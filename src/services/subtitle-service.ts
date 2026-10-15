import { webSubtitleService } from "./adapters/web/web-subtitle-service"
import { mockSubtitleService } from "./adapters/mock/mock-subtitle-service"
export const useSubtitleService = () => import.meta.env.VITE_USE_WEB_API === "true" ? webSubtitleService : mockSubtitleService
