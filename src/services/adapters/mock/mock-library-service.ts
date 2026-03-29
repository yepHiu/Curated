import { computed, ref, watch } from "vue"
import type {
  ActorListItemDTO,
  ActorsListDTO,
  ListActorsParams,
  MetadataMovieScrapeMode,
  MetadataRefreshQueuedDTO,
  PatchMovieBody,
  TaskDTO,
} from "@/api/types"
import type { LibrarySetting } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { i18n } from "@/i18n"
import { countCuratedFrames } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { buildSettingsDashboardStats } from "@/lib/library-stats"
import { sampleRandomMovies } from "@/lib/random-sample"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"

function normalizeMockLibraryPath(p: string): string {
  return p.trim().replace(/\\/g, "/")
}

/** Mirrors backend pathHasLibraryRoot for mock path strings (case-insensitive prefix). */
function mockPathHasLibraryRoot(loc: string, root: string): boolean {
  const l = normalizeMockLibraryPath(loc)
  const r = normalizeMockLibraryPath(root)
  if (!l || !r || r === ".") return false
  if (l.toLowerCase() === r.toLowerCase()) return true
  const prefix = r.endsWith("/") ? r.toLowerCase() : `${r.toLowerCase()}/`
  return l.length >= prefix.length && l.slice(0, prefix.length).toLowerCase() === prefix
}
import {
  loadMockMoviePrefs,
  mergeMockPrefsIntoMovie,
  upsertMockMoviePrefs,
} from "@/lib/mock-movie-prefs-storage"
import type { LibraryService } from "@/services/contracts/library-service"

const organizeLibraryMock = ref(false)
const extendedLibraryImportMock = ref(false)
const autoLibraryWatchMock = ref(true)
const metadataMovieProviderMock = ref("")
/** Mock 无引擎枚举，列表为空＝仅自动模式 */
const metadataMovieProvidersMock = ref<string[]>([])
/** Mock：有序的 Provider 列表 */
const metadataMovieProviderChainMock = ref<string[]>([])
const metadataMovieScrapeModeMock = ref<MetadataMovieScrapeMode>("auto")
/** Mock：HTTP 代理配置 */
const proxyMock = ref<import("@/api/types").ProxySettingsDTO>({ enabled: false })

/** 设置页概览第三卡：萃取帧条数（IndexedDB） */
const curatedFramesCountState = ref(0)

async function refreshCuratedFramesCountMock() {
  try {
    curatedFramesCountState.value = await countCuratedFrames()
  } catch {
    curatedFramesCountState.value = 0
  }
}

watch(curatedFramesRevision, () => {
  void refreshCuratedFramesCountMock()
})
void refreshCuratedFramesCountMock()

/** Mock：演员用户标签（与影片 userTags 隔离） */
const mockActorUserTags = ref<Map<string, string[]>>(new Map())

function mockActorsFromMovies(): ActorListItemDTO[] {
  const counts = new Map<string, number>()
  for (const m of moviesState.value) {
    if (m.trashedAt?.trim()) continue
    for (const raw of m.actors) {
      const a = raw.trim()
      if (!a) continue
      counts.set(a, (counts.get(a) ?? 0) + 1)
    }
  }
  const names = [...counts.keys()].sort((x, y) => x.localeCompare(y))
  return names.map((name) => ({
    name,
    avatarUrl: "",
    movieCount: counts.get(name) ?? 0,
    userTags: [...(mockActorUserTags.value.get(name) ?? [])].sort((x, y) => x.localeCompare(y)),
  }))
}

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
    userTags: [],
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
    userTags: [],
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
    userTags: [],
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
    userTags: [],
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
    userTags: [],
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
    userTags: [],
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
    releaseDate: `${seed.year + yearOffset}-${month}-${day}`,
    userTags: [],
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
  if (body.userTags !== undefined) {
    next.userTags = [...body.userTags]
  }
  if (body.metadataTags !== undefined) {
    next.tags = [...body.metadataTags]
  }

  const touchDisplayFallback = () => {
    if (!next.displayScrapeFallback) {
      next.displayScrapeFallback = {
        title: cur.title,
        studio: cur.studio,
        summary: cur.summary,
        releaseDate: cur.releaseDate,
        runtimeMinutes: cur.runtimeMinutes,
        year: cur.year,
      }
    }
  }

  if (body.userTitle !== undefined) {
    if (body.userTitle === null || body.userTitle === "") {
      const fb = next.displayScrapeFallback
      next.title = fb?.title ?? cur.title
    } else {
      touchDisplayFallback()
      next.title = body.userTitle
    }
  }
  if (body.userStudio !== undefined) {
    if (body.userStudio === null || body.userStudio === "") {
      const fb = next.displayScrapeFallback
      next.studio = fb?.studio ?? cur.studio
    } else {
      touchDisplayFallback()
      next.studio = body.userStudio
    }
  }
  if (body.userSummary !== undefined) {
    if (body.userSummary === null || body.userSummary === "") {
      const fb = next.displayScrapeFallback
      next.summary = fb?.summary ?? cur.summary
    } else {
      touchDisplayFallback()
      next.summary = body.userSummary
    }
  }
  if (body.userReleaseDate !== undefined) {
    if (body.userReleaseDate === null || body.userReleaseDate === "") {
      const fb = next.displayScrapeFallback
      next.releaseDate = fb?.releaseDate
      next.year = fb?.year ?? cur.year
    } else {
      touchDisplayFallback()
      next.releaseDate = body.userReleaseDate
      const y = parseInt(body.userReleaseDate.slice(0, 4), 10)
      if (!Number.isNaN(y) && y >= 1800 && y <= 3000) {
        next.year = y
      }
    }
  }
  if (body.userRuntimeMinutes !== undefined) {
    if (body.userRuntimeMinutes === null) {
      const fb = next.displayScrapeFallback
      next.runtimeMinutes = fb?.runtimeMinutes ?? cur.runtimeMinutes
    } else {
      touchDisplayFallback()
      next.runtimeMinutes = body.userRuntimeMinutes
    }
  }

  moviesState.value = moviesState.value.map((m, i) => (i === idx ? next : m))

  const prefsPatch: {
    isFavorite?: boolean
    userRating?: number | null
    userTags?: string[]
    metadataTags?: string[]
  } = {}
  if (body.isFavorite !== undefined) {
    prefsPatch.isFavorite = next.isFavorite
  }
  if (body.rating !== undefined) {
    prefsPatch.userRating = body.rating === null ? null : body.rating
  }
  if (body.userTags !== undefined) {
    prefsPatch.userTags = next.userTags
  }
  if (body.metadataTags !== undefined) {
    prefsPatch.metadataTags = next.tags
  }
  if (Object.keys(prefsPatch).length > 0) {
    upsertMockMoviePrefs(id, prefsPatch)
  }

  return next
}

export const mockLibraryService: LibraryService = {
  movies: computed(() => moviesState.value.filter((m) => !m.trashedAt?.trim())),
  trashedMovies: computed(() =>
    moviesState.value
      .filter((m) => Boolean(m.trashedAt?.trim()))
      .slice()
      .sort((a, b) => (b.trashedAt ?? "").localeCompare(a.trashedAt ?? "")),
  ),
  libraryStats: computed(() =>
    buildSettingsDashboardStats(
      moviesState.value.filter((m) => !m.trashedAt?.trim()),
      curatedFramesCountState.value,
      i18n.global.locale.value as string,
    ),
  ),
  libraryPaths: computed(() => libraryPathsState.value),
  organizeLibrary: computed(() => organizeLibraryMock.value),
  extendedLibraryImport: computed(() => extendedLibraryImportMock.value),
  autoLibraryWatch: computed(() => autoLibraryWatchMock.value),
  metadataMovieProvider: computed(() => metadataMovieProviderMock.value),
  metadataMovieProviders: computed(() => metadataMovieProvidersMock.value),
  metadataMovieProviderChain: computed(() => metadataMovieProviderChainMock.value),
  metadataMovieScrapeMode: computed(() => metadataMovieScrapeModeMock.value),
  proxy: computed(() => proxyMock.value),

  async setProxy(config: import("@/api/types").ProxySettingsDTO) {
    proxyMock.value = { ...config }
  },

  async refreshSettings() {
    // Mock: paths are in-memory only; no remote settings.
  },

  async reloadMoviesFromApi() {
    // Mock: 列表为本地种子，无远端同步。
  },

  async setOrganizeLibrary(value: boolean) {
    organizeLibraryMock.value = value
  },

  async setExtendedLibraryImport(value: boolean) {
    extendedLibraryImportMock.value = value
  },

  async setAutoLibraryWatch(value: boolean) {
    autoLibraryWatchMock.value = value
  },

  async setMetadataMovieProvider(name: string) {
    const trimmed = name.trim()
    if (trimmed !== "" && metadataMovieProvidersMock.value.length === 0) {
      throw new Error("Mock mode has no provider list; use Web API to pick a source.")
    }
    if (
      trimmed !== "" &&
      !metadataMovieProvidersMock.value.some((p) => p.toLowerCase() === trimmed.toLowerCase())
    ) {
      throw new Error("Unknown metadata provider in mock.")
    }
    metadataMovieProviderMock.value = trimmed
    metadataMovieScrapeModeMock.value = trimmed === "" ? "auto" : "specified"
  },

  async setMetadataMovieProviderChain(chain: string[]) {
    const filtered = chain.map((p) => p.trim()).filter(Boolean)
    // In mock mode, we accept any non-empty strings since there's no real provider list
    metadataMovieProviderChainMock.value = filtered
    if (filtered.length === 0) {
      metadataMovieScrapeModeMock.value = "auto"
      metadataMovieProviderMock.value = ""
    } else {
      metadataMovieScrapeModeMock.value = "chain"
      metadataMovieProviderMock.value = ""
    }
  },

  async setMetadataMovieScrapeMode(mode: MetadataMovieScrapeMode) {
    metadataMovieScrapeModeMock.value = mode
  },

  async addLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
    const trimmed = path.trim()
    if (!trimmed) return null
    if (!isAbsoluteLibraryPath(trimmed)) {
      throw new Error("library path must be an absolute path")
    }
    const id = `mock-${Date.now()}`
    libraryPathsState.value = [
      ...libraryPathsState.value,
      { id, path: trimmed, title: (title?.trim() || trimmed) },
    ]
    return null
  },

  async updateLibraryPathTitle(id: string, title: string) {
    const t = title.trim()
    libraryPathsState.value = libraryPathsState.value.map((p) =>
      p.id === id ? { ...p, title: t || p.title } : p,
    )
  },

  async removeLibraryPath(id: string) {
    const removed = libraryPathsState.value.find((p) => p.id === id)
    libraryPathsState.value = libraryPathsState.value.filter((p) => p.id !== id)
    if (!removed) return
    const removedRoot = normalizeMockLibraryPath(removed.path)
    const remainingRoots = libraryPathsState.value.map((p) => normalizeMockLibraryPath(p.path))
    moviesState.value = moviesState.value.filter((m) => {
      const loc = normalizeMockLibraryPath(m.location)
      if (!loc) return true
      if (!mockPathHasLibraryRoot(loc, removedRoot)) return true
      return remainingRoots.some((r) => mockPathHasLibraryRoot(loc, r))
    })
  },

  async scanLibraryPaths() {
    // Mock: no backend scan.
    return null
  },

  async refreshMovieMetadata() {
    return null
  },

  async revealMovieInFileManager() {
    throw new Error("MOCK_REVEAL_NOT_SUPPORTED")
  },

  async refreshMetadataForLibraryPaths(): Promise<MetadataRefreshQueuedDTO> {
    return { queued: 0, skipped: 0, invalidPaths: [] }
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
    const id = movieId.trim()
    const pool = moviesState.value.filter(
      (movie) => movie.id !== id && !movie.trashedAt?.trim(),
    )
    return sampleRandomMovies(pool, limit, id || "_")
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
    const id = movieId.trim()
    const idx = moviesState.value.findIndex((m) => m.id === id)
    if (idx < 0) return
    const ts = new Date().toISOString()
    moviesState.value = moviesState.value.map((m, i) =>
      i === idx ? { ...m, trashedAt: ts } : m,
    )
  },

  async restoreMovie(movieId: string) {
    const id = movieId.trim()
    moviesState.value = moviesState.value.map((m) =>
      m.id === id ? { ...m, trashedAt: undefined } : m,
    )
  },

  async deleteMoviePermanently(movieId: string) {
    const id = movieId.trim()
    moviesState.value = moviesState.value.filter((m) => m.id !== id)
  },

  mergeMovieIntoCache() {
    // Mock：无中央 HTTP 缓存；持久化由 localStorage 在 applyMockPatchMovie 中处理。
  },

  async listActors(params?: ListActorsParams): Promise<ActorsListDTO> {
    let rows = mockActorsFromMovies().filter((r) => r.movieCount > 0)
    const q = params?.q?.trim().toLowerCase() ?? ""
    if (q) {
      rows = rows.filter((r) => {
        if (r.name.toLowerCase().includes(q)) {
          return true
        }
        return (r.userTags ?? []).some((t) => t.toLowerCase().includes(q))
      })
    }
    const actorTag = params?.actorTag?.trim() ?? ""
    if (actorTag) {
      rows = rows.filter((r) => (r.userTags ?? []).some((t) => t === actorTag))
    }
    if (params?.sort === "movieCount") {
      rows = [...rows].sort(
        (a, b) => b.movieCount - a.movieCount || a.name.localeCompare(b.name),
      )
    }
    const total = rows.length
    const limit = params?.limit && params.limit > 0 ? params.limit : 50
    const offset = params?.offset && params.offset > 0 ? params.offset : 0
    const slice = rows.slice(offset, offset + limit)
    return { total, actors: slice }
  },

  async patchActorUserTags(name: string, userTags: string[]): Promise<ActorListItemDTO> {
    const n = name.trim()
    if (!n) {
      throw new Error("actor name is required")
    }
    if (!mockActorsFromMovies().some((r) => r.name === n)) {
      throw new Error("actor not found")
    }
    const normalized = [
      ...new Set(
        userTags
          .map((t) => t.trim())
          .filter((t) => t !== ""),
      ),
    ].sort((x, y) => x.localeCompare(y))
    const next = new Map(mockActorUserTags.value)
    next.set(n, normalized)
    mockActorUserTags.value = next
    const row = mockActorsFromMovies().find((r) => r.name === n)
    if (!row) {
      throw new Error("actor not found")
    }
    return row
  },
}
