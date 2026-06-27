import { mockComicLibraryService } from "@/services/adapters/mock/mock-comic-library-service"
import { webComicLibraryService } from "@/services/adapters/web/web-comic-library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

export const useComicLibraryService = () =>
  USE_WEB ? webComicLibraryService : mockComicLibraryService
