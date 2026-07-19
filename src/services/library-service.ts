import { mockLibraryService } from "@/services/adapters/mock/mock-library-service"
import {
  startWebLibraryService,
  webLibraryService,
} from "@/services/adapters/web/web-library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

if (USE_WEB) {
  void startWebLibraryService()
}

export const useLibraryService = () => (USE_WEB ? webLibraryService : mockLibraryService)
