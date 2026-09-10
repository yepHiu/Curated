import { mockPhotoLibraryService } from "@/services/adapters/mock/mock-photo-library-service"
import { webPhotoLibraryService } from "@/services/adapters/web/web-photo-library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

export const usePhotoLibraryService = () =>
  USE_WEB ? webPhotoLibraryService : mockPhotoLibraryService
