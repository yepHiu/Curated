import { computed, ref } from "vue"
import type { PatchMovieBody } from "@/api/types"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import {
  loadMockMoviePrefs,
  mergeMockPrefsIntoMovie,
  upsertMockMoviePrefs,
} from "@/lib/mock-movie-prefs-storage"
import type { LibraryService } from "@/services/contracts/library-service"

const libraryStats: readonly LibraryStat[] = [
  {
    label: "Movies Indexed",
    value: "2,184",
    detail: "Across local and removable libraries",
  },
  {
    label: "Favorite Picks",
    value: "246",
    detail: "Curated for quick rewatch sessions",
  },
  {
    label: "Metadata Health",
    value: "98%",
    detail: "Poster, cast, and tags fully matched",
  },
]

const organizeLibraryMock = ref(false)

const libraryPathsState = ref<LibrarySetting[]>([
  {
    id: "library-a",
    path: "D:/Media/JAV/Main",
    title: "Primary archive",
  },
  {
    id: "library-b",
    path: "E:/Vault/JAV/New",
    title: "Recently imported",
  },
  {
    id: "library-c",
    path: "F:/Offline/Collections",
    title: "Cold storage",
  },
])

const movieSeeds: Omit<Movie, "id" | "code" | "location" | "addedAt">[] = [
  {
    title: "Midnight Kiss Broadcast",
    studio: "Velvet North",
    actors: ["Mina Kaze", "Rin Asuka"],
    tags: ["Romance", "4K", "Late Night"],
    runtimeMinutes: 134,
    rating: 4.8,
    summary:
      "A polished late-night feature with a slow-burn mood, crisp lighting, and strong cast chemistry.",
    isFavorite: true,
    resolution: "2160p",
    year: 2025,
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
  },
  {
    title: "Silk Line Directive",
    studio: "Studio Garnet",
    actors: ["Airi Sena"],
    tags: ["Drama", "Office", "High Rating"],
    runtimeMinutes: 126,
    rating: 4.7,
    summary:
      "An elegant office-set release with high production values and detailed metadata coverage.",
    isFavorite: true,
    resolution: "1080p",
    year: 2025,
    tone: "from-secondary via-accent/60 to-card",
    coverClass: "aspect-[4/4.8]",
  },
  {
    title: "Neon Velvet Archive",
    studio: "Moonlight Works",
    actors: ["Yua Mori", "Nao Shin"],
    tags: ["Sci-Fi", "Stylized", "New"],
    runtimeMinutes: 118,
    rating: 4.5,
    summary:
      "A stylized catalog favorite that mixes strong visual direction with a fast pace.",
    isFavorite: false,
    resolution: "2160p",
    year: 2026,
    tone: "from-accent via-primary/15 to-card",
    coverClass: "aspect-[4/5.2]",
  },
  {
    title: "Horizon Zero Kisses",
    studio: "North Pier",
    actors: ["Emi Kisaragi"],
    tags: ["Travel", "Outdoor", "Recently Added"],
    runtimeMinutes: 142,
    rating: 4.2,
    summary:
      "A travel-heavy feature with standout scenery and a well-tagged scene structure.",
    isFavorite: false,
    resolution: "1080p",
    year: 2026,
    tone: "from-muted via-primary/10 to-card",
    coverClass: "aspect-[4/5.8]",
  },
  {
    title: "Private Room Memoir",
    studio: "Golden Frame",
    actors: ["Sora Minami", "Miu Arata"],
    tags: ["Character", "Favorites", "Longform"],
    runtimeMinutes: 151,
    rating: 4.9,
    summary:
      "One of the strongest longform entries in the library, with rich cast notes and clean artwork.",
    isFavorite: true,
    resolution: "2160p",
    year: 2024,
    tone: "from-primary/25 via-accent/50 to-card",
    coverClass: "aspect-[4/5]",
  },
  {
    title: "Lovers in Static",
    studio: "Afterglow",
    actors: ["Kanna Rei"],
    tags: ["Moody", "Slow Burn", "Archive"],
    runtimeMinutes: 129,
    rating: 4.1,
    summary:
      "A moody catalog entry used as a reference for poster-heavy browsing and tag grouping.",
    isFavorite: false,
    resolution: "1080p",
    year: 2023,
    tone: "from-secondary/80 via-muted to-card",
    coverClass: "aspect-[4/4.6]",
  },
]

const codePrefixes = ["MKB", "SLD", "NVA", "HZK", "PRM", "LVS", "KTR", "AMR", "VLT", "NOA"]
const storagePools = ["D:/Media/JAV/Main", "E:/Vault/JAV/New", "F:/Offline/Collections"]

const buildMovie = (index: number): Movie => {
  const seed = movieSeeds[index % movieSeeds.length]
  const prefix = codePrefixes[index % codePrefixes.length]
  const serial = String(100 + index).padStart(3, "0")
  const month = String((index % 12) + 1).padStart(2, "0")
  const day = String((index % 27) + 1).padStart(2, "0")
  const runtimeOffset = index % 17
  const ratingOffset = (index % 5) * 0.1
  const yearOffset = index % 3
  const storage = storagePools[index % storagePools.length]

  const rating = Math.max(3.9, Number((seed.rating - ratingOffset).toFixed(1)))
  return {
    ...seed,
    id: `${prefix.toLowerCase()}-${serial}`,
    title: `${seed.title} ${index + 1}`,
    code: `${prefix}-${serial}`,
    runtimeMinutes: seed.runtimeMinutes + runtimeOffset,
    rating,
    metadataRating: rating,
    userRating: undefined,
    isFavorite: index % 4 === 0 ? true : seed.isFavorite,
    addedAt: `2026-${month}-${day}`,
    location: `${storage}/${prefix}-${serial}.${index % 2 === 0 ? "mkv" : "mp4"}`,
    year: seed.year + yearOffset,
    tags: [...seed.tags, index % 6 === 0 ? "Trending" : "Catalog"],
    thumbUrl: `https://picsum.photos/seed/jav-thumb-${prefix}-${serial}/280/400`,
    coverUrl: `https://picsum.photos/seed/jav-cover-${prefix}-${serial}/560/840`,
    previewImages: [
      `https://picsum.photos/seed/jav-p1-${prefix}-${serial}/640/360`,
      `https://picsum.photos/seed/jav-p2-${prefix}-${serial}/640/360`,
      `https://picsum.photos/seed/jav-p3-${prefix}-${serial}/640/360`,
    ],
  }
}

loadMockMoviePrefs()

const moviesState = ref<Movie[]>(
  Array.from({ length: 180 }, (_, index) => mergeMockPrefsIntoMovie(buildMovie(index))),
)

function applyMockPatchMovie(movieId: string, body: PatchMovieBody): Movie | undefined {
  const id = movieId.trim()
  const idx = moviesState.value.findIndex((m) => m.id === id)
  if (idx < 0) {
    return undefined
  }
  const cur = moviesState.value[idx]
  let next: Movie = { ...cur }
  if (body.isFavorite !== undefined) {
    next.isFavorite = body.isFavorite
  }
  if (body.rating !== undefined) {
    if (body.rating === null) {
      next.userRating = undefined
      next.rating = next.metadataRating ?? cur.rating
    } else {
      next.userRating = body.rating
      next.rating = body.rating
      if (next.metadataRating === undefined) {
        next.metadataRating = cur.rating
      }
    }
  }
  moviesState.value = moviesState.value.map((m, i) => (i === idx ? next : m))

  const prefsPatch: { isFavorite?: boolean; userRating?: number | null } = {}
  if (body.isFavorite !== undefined) {
    prefsPatch.isFavorite = next.isFavorite
  }
  if (body.rating !== undefined) {
    prefsPatch.userRating = body.rating === null ? null : body.rating
  }
  if (Object.keys(prefsPatch).length > 0) {
    upsertMockMoviePrefs(id, prefsPatch)
  }

  return next
}

export const mockLibraryService: LibraryService = {
  movies: computed(() => moviesState.value),
  libraryStats,
  libraryPaths: computed(() => libraryPathsState.value),
  organizeLibrary: computed(() => organizeLibraryMock.value),

  async refreshSettings() {
    // Mock: paths are in-memory only; no remote settings.
  },

  async setOrganizeLibrary(value: boolean) {
    organizeLibraryMock.value = value
  },

  async addLibraryPath(path: string, title?: string) {
    const trimmed = path.trim()
    if (!trimmed) return
    if (!isAbsoluteLibraryPath(trimmed)) {
      throw new Error("library path must be an absolute path")
    }
    const id = `mock-${Date.now()}`
    libraryPathsState.value = [
      ...libraryPathsState.value,
      { id, path: trimmed, title: (title?.trim() || trimmed) },
    ]
  },

  async updateLibraryPathTitle(id: string, title: string) {
    const t = title.trim()
    libraryPathsState.value = libraryPathsState.value.map((p) =>
      p.id === id ? { ...p, title: t || p.title } : p,
    )
  },

  async removeLibraryPath(id: string) {
    libraryPathsState.value = libraryPathsState.value.filter((p) => p.id !== id)
  },

  async scanLibraryPaths() {
    // Mock: no backend scan.
    return null
  },

  async refreshMovieMetadata() {
    return null
  },

  getMoviePlaybackUrl() {
    return null
  },

  async ensureMovieCached() {
    // Mock 数据全在内存，无需远程补全。
  },

  getMovieById(movieId) {
    return moviesState.value.find((movie) => movie.id === movieId)
  },
  getRelatedMovies(movieId, limit = 6) {
    return moviesState.value.filter((movie) => movie.id !== movieId).slice(0, limit)
  },

  async patchMovie(movieId, body) {
    return applyMockPatchMovie(movieId, body)
  },

  async toggleFavorite(movieId, nextValue) {
    const id = movieId.trim()
    const currentMovie = moviesState.value.find((movie) => movie.id === id)
    if (!currentMovie) {
      return undefined
    }
    const targetValue = typeof nextValue === "boolean" ? nextValue : !currentMovie.isFavorite
    return applyMockPatchMovie(id, { isFavorite: targetValue })
  },

  async deleteMovie(movieId: string) {
    moviesState.value = moviesState.value.filter((movie) => movie.id !== movieId)
  },

  mergeMovieIntoCache() {
    // Mock：无中央 HTTP 缓存；持久化由 localStorage 在 applyMockPatchMovie 中处理。
  },
}
