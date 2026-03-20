import { mockLibraryService } from "@/services/adapters/mock/mock-library-service"
import { webLibraryService } from "@/services/adapters/web/web-library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

export const useLibraryService = () => (USE_WEB ? webLibraryService : mockLibraryService)
