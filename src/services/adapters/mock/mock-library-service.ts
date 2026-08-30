import { computed, ref, watch } from "vue"
import type {
  ActorListItemDTO,
  ActorMergeAuditDTO,
  ActorMergeAuditListDTO,
  ActorMergePreviewDTO,
  ActorMergePreviewRequest,
  ActorMergeProfileFieldDTO,
  ApplyActorMergeRequest,
  ActorProfileDTO,
  ActorsListDTO,
  BackendLogSettingsDTO,
  ConnectedClientsDTO,
  CuratedFrameExportFormat,
  CuratedFrameExportMode,
  HealthDTO,
  HomepageDailyRecommendationsDTO,
  HomepageRecommendationFeedbackEffectDTO,
  HomepageRecommendationItemDTO,
  CreateRecommendationFeedbackBody,
  RecommendationFeedbackDTO,
  RecommendationFeedbackListDTO,
  RefreshHomepageDailyRecommendationsBody,
  LibraryPathStorageStatusDTO,
  NativePlayerPreset,
  ListActorsParams,
  MetadataMovieScrapeMode,
  MetadataRefreshQueuedDTO,
  MovieCommentDTO,
  ImportMovieCodeCheckDTO,
  ImportMovieCodeMatchDTO,
  PersonalInsightsBreakdownDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
  PatchPlayerSettingsBody,
  PlayerSettingsDTO,
  PatchBackendLogBody,
  PatchMovieBody,
  PingAllProvidersResponse,
  ProviderHealthDTO,
  ProxyJavBusPingResponse,
  PutMovieCommentBody,
  SavedViewDTO,
  SavedViewFiltersV1,
  TaskDTO,
} from "@/api/types"
import type { LibrarySetting } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { i18n } from "@/i18n"
import { countCuratedFrames } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { normalizeActorIdentity } from "@/lib/actor-identity"
import { getCurrentUtcDayKey } from "@/lib/current-utc-day-key"
import { buildHomepagePortalModel } from "@/lib/homepage-portal"
import { buildSettingsDashboardStats } from "@/lib/library-stats"
import { buildPersonalInsightsBreakdown, buildPersonalInsightsOverview } from "@/lib/personal-insights"
import { listSortedByUpdatedDesc } from "@/lib/playback-progress-storage"
import { listPlaybackWatchTimeMovieEntries } from "@/lib/playback-watch-time-storage"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import { getLocalMovieComment, putLocalMovieComment } from "@/lib/movie-comment-local-storage"
import {
  classifyMovieCodes,
  extractMovieNumber,
  strongerMovieCodeMatch,
} from "@/lib/movie-number"
import { HttpClientError } from "@/api/http-client"
import {
  normalizeSavedViewFiltersV1,
  normalizeSavedViewName,
  normalizedSavedViewNameKey,
} from "@/lib/saved-view-model"
import { loadLocalSavedViews, saveLocalSavedViews } from "@/lib/saved-views-local-storage"
import {
  loadLocalRecommendationFeedback,
  saveLocalRecommendationFeedback,
} from "@/lib/recommendation-feedback-local-storage"
import {
  loadLocalActorMergeState,
  saveLocalActorMergeState,
  type LocalActorMergeState,
} from "@/lib/actor-merge-local-storage"

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
import type { LibraryService, ResumableMovieImportSession } from "@/services/contracts/library-service"
import {
  getCuratedFrameExportMode,
  setCuratedFrameExportMode as persistCuratedFrameExportMode,
} from "@/lib/curated-frames/settings-storage"

const organizeLibraryMock = ref(false)
const backupDirectoryMock = ref("")
const autoLibraryWatchMock = ref(true)
const autoActorProfileScrapeMock = ref(false)
const autoDownloadUpdatesMock = ref(false)
const launchAtLoginMock = ref(false)
const launchAtLoginSupportedMock = ref(false)
const curatedFrameExportFormatMock = ref<CuratedFrameExportFormat>("jpg")
const curatedFrameExportModeMock = ref<CuratedFrameExportMode>(getCuratedFrameExportMode())
const metadataMovieProviderMock = ref("")
/** Mock 无引擎枚举，列表为空＝仅自动模式 */
const metadataMovieProvidersMock = ref<string[]>([])
/** Mock：有序的 Provider 列表 */
const metadataMovieProviderChainMock = ref<string[]>([])
const metadataMovieScrapeModeMock = ref<MetadataMovieScrapeMode>("auto")
/** Mock：HTTP 代理配置 */
const proxyMock = ref<import("@/api/types").ProxySettingsDTO>({ enabled: false })
/** Mock：实验性 Agent provider 配置（localStorage 持久化，便于刷新后保留演示配置） */
const AI_PROVIDER_MOCK_STORAGE_KEY = "curated-ai-provider-mock-v1"

function readAIProviderMock(): import("@/api/types").AIProviderSettingsDTO {
  try {
    const raw = localStorage.getItem(AI_PROVIDER_MOCK_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as import("@/api/types").AIProviderSettingsDTO
      return {
        kind: parsed.kind || "openai-compatible",
        baseUrl: typeof parsed.baseUrl === "string" ? parsed.baseUrl : "",
        apiKey: typeof parsed.apiKey === "string" ? parsed.apiKey : "",
        model: typeof parsed.model === "string" ? parsed.model : "",
      }
    }
  } catch {
    // ignore malformed local storage
  }
  return { kind: "openai-compatible", baseUrl: "", model: "" }
}

function persistAIProviderMock(value: import("@/api/types").AIProviderSettingsDTO) {
  try {
    localStorage.setItem(AI_PROVIDER_MOCK_STORAGE_KEY, JSON.stringify(value))
  } catch {
    // ignore storage failures
  }
}

const aiProviderMock = ref<import("@/api/types").AIProviderSettingsDTO>(readAIProviderMock())
const playerSettingsMock = ref<PlayerSettingsDTO>({
  hardwareDecode: true,
  hardwareEncoder: "auto",
  nativePlayerPreset: "custom",
  nativePlayerEnabled: false,
  nativePlayerCommand: "",
  streamPushEnabled: true,
  forceStreamPush: false,
  ffmpegCommand: "ffmpeg",
  preferNativePlayer: false,
  seekForwardStepSec: 10,
  seekBackwardStepSec: 10,
})
const backendLogMock = ref<BackendLogSettingsDTO>({ logDir: "", logLevel: "info" })
const defaultImportLibraryPathIdMock = ref("library-a")
const savedViewsMock = ref<SavedViewDTO[]>(
  loadLocalSavedViews(typeof localStorage === "undefined" ? undefined : localStorage),
)
const recommendationFeedbackMock = ref<RecommendationFeedbackDTO[]>(
  loadLocalRecommendationFeedback(typeof localStorage === "undefined" ? undefined : localStorage),
)
const actorMergeStateMock = ref<LocalActorMergeState>(
  loadLocalActorMergeState(typeof localStorage === "undefined" ? undefined : localStorage),
)
const libraryPathStorageStatusesMock = ref<LibraryPathStorageStatusDTO[]>([
  {
    libraryPathId: "library-a",
    path: "D:/Media/JAV/Main",
    title: "Primary archive",
    status: "online",
    message: "Storage path is online.",
    checkedAt: new Date().toISOString(),
    rootPath: "D:/",
    driveType: "fixed",
    volumeLabel: "Media",
    identityConfidence: "mock",
    canRescan: true,
    canImport: true,
  },
  {
    libraryPathId: "library-b",
    path: "E:/Vault/JAV/New",
    title: "Recently imported",
    status: "online",
    message: "Storage path is online.",
    checkedAt: new Date().toISOString(),
    rootPath: "E:/",
    driveType: "removable",
    volumeLabel: "Vault",
    identityConfidence: "mock",
    canRescan: true,
    canImport: true,
  },
  {
    libraryPathId: "library-c",
    path: "F:/Offline/Collections",
    title: "Cold storage",
    status: "offline",
    message: "Mock: this external archive is offline.",
    checkedAt: new Date().toISOString(),
    rootPath: "F:/",
    driveType: "removable",
    volumeLabel: "Cold",
    identityConfidence: "mock",
    canRescan: false,
    canImport: false,
  },
])

function buildMockConnectedClients(): ConnectedClientsDTO {
  const sampledAt = new Date().toISOString()
  const clients = [
    {
      key: "mock-local-chrome",
      ip: "127.0.0.1",
      hostname: "this-device",
      browser: "Chrome",
      browserVersion: "132",
      os: "Windows",
      osVersion: "11",
      deviceType: "desktop" as const,
      accessKind: "local" as const,
      isLocalMachine: true,
      firstSeen: "2026-05-15T09:40:00Z",
      lastSeen: sampledAt,
      requestCount: 128,
    },
    {
      key: "mock-iphone-safari",
      ip: "192.168.1.8",
      hostname: "iphone.lan",
      browser: "Safari",
      browserVersion: "18",
      os: "iOS",
      osVersion: "18.2",
      deviceType: "mobile" as const,
      accessKind: "remote" as const,
      isLocalMachine: false,
      firstSeen: "2026-05-15T09:55:00Z",
      lastSeen: "2026-05-15T10:05:00Z",
      requestCount: 34,
    },
    {
      key: "mock-mac-firefox",
      ip: "192.168.1.22",
      hostname: "macbook-pro.lan",
      browser: "Firefox",
      browserVersion: "135",
      os: "macOS",
      osVersion: "15.2",
      deviceType: "laptop" as const,
      accessKind: "remote" as const,
      isLocalMachine: false,
      firstSeen: "2026-05-15T09:10:00Z",
      lastSeen: "2026-05-15T09:58:00Z",
      requestCount: 57,
    },
  ]
  return {
    clients,
    total: clients.length,
    localCount: clients.filter((client) => client.accessKind === "local").length,
    remoteCount: clients.filter((client) => client.accessKind === "remote").length,
    sampledAt,
  }
}

function mockHttpError(status: number, code: string, message = code): HttpClientError {
  return new HttpClientError(status, {
    code,
    message,
    retryable: false,
  })
}

function persistMockSavedViews(items: SavedViewDTO[]) {
  savedViewsMock.value = items.map((item, sortOrder) => ({ ...item, sortOrder }))
  saveLocalSavedViews(
    savedViewsMock.value,
    typeof localStorage === "undefined" ? undefined : localStorage,
  )
}

function newMockSavedViewID(): string {
  const uuid = globalThis.crypto?.randomUUID?.()
  return `view_${uuid ?? `${Date.now()}_${Math.random().toString(16).slice(2)}`}`
}

function persistMockRecommendationFeedback(items: RecommendationFeedbackDTO[]) {
  recommendationFeedbackMock.value = [...items].sort(
    (left, right) => right.createdAt.localeCompare(left.createdAt) || left.id.localeCompare(right.id),
  )
  saveLocalRecommendationFeedback(
    recommendationFeedbackMock.value,
    typeof localStorage === "undefined" ? undefined : localStorage,
  )
}

function refreshActiveMockRecommendationFeedback() {
  const now = Date.now()
  const active = recommendationFeedbackMock.value.filter(
    (item) => !item.expiresAt || Date.parse(item.expiresAt) > now,
  )
  if (active.length !== recommendationFeedbackMock.value.length) {
    persistMockRecommendationFeedback(active)
  }
}

function mockRecommendationFeedbackID(): string {
  const uuid = globalThis.crypto?.randomUUID?.()
  return `feedback_${uuid ?? `${Date.now()}_${Math.random().toString(16).slice(2)}`}`
}

function mockRecommendationFeedbackTargetKey(
  targetType: RecommendationFeedbackDTO["targetType"],
  targetValue: string,
): string {
  return targetType === "actor"
    ? normalizeActorIdentity(targetValue)
    : targetValue.trim().toLocaleLowerCase()
}

function canonicalMockFeedbackTarget(
  movie: Movie,
  targetType: CreateRecommendationFeedbackBody["targetType"],
  targetValue: string,
): string | undefined {
  const wanted = targetValue.trim()
  if (targetType === "movie") return movie.id.toLocaleLowerCase() === wanted.toLocaleLowerCase() ? movie.id : undefined
  if (targetType === "studio") return movie.studio.trim().toLocaleLowerCase() === wanted.toLocaleLowerCase() ? movie.studio.trim() : undefined
  const values = targetType === "actor" ? movie.actors : [...movie.tags, ...movie.userTags]
  const wantedKey = targetType === "actor"
    ? normalizeActorIdentity(wanted)
    : wanted.toLocaleLowerCase()
  return values.find((value) => {
    const valueKey = targetType === "actor"
      ? normalizeActorIdentity(value)
      : value.trim().toLocaleLowerCase()
    return valueKey === wantedKey
  })?.trim()
}

function isValidMockFeedbackShape(body: CreateRecommendationFeedbackBody): boolean {
  if (body.action === "not_interested") return body.targetType === "movie" && body.durationDays === undefined
  if (body.action === "snooze") {
    return body.targetType === "movie" && Number.isInteger(body.durationDays) && (body.durationDays ?? 0) >= 1 && (body.durationDays ?? 0) <= 365
  }
  return body.action === "less" && ["actor", "studio", "tag"].includes(body.targetType) && body.durationDays === undefined
}

function mockFeedbackEffects(movie: Movie): HomepageRecommendationFeedbackEffectDTO[] {
  const values = {
    actor: new Set(movie.actors.map(normalizeActorIdentity)),
    studio: new Set([movie.studio.trim().toLocaleLowerCase()]),
    tag: new Set([...movie.tags, ...movie.userTags].map((value) => value.trim().toLocaleLowerCase())),
  }
  return recommendationFeedbackMock.value
    .filter((item) => {
      if (item.action !== "less" || item.targetType === "movie") return false
      const normalized = item.targetType === "actor"
        ? normalizeActorIdentity(item.targetValue)
        : item.targetValue.trim().toLocaleLowerCase()
      return values[item.targetType].has(normalized)
    })
    .map((item) => ({
      feedbackId: item.id,
      targetType: item.targetType as "actor" | "studio" | "tag",
      targetValue: item.targetValue,
      effect: "weight_reduced" as const,
    }))
    .sort((left, right) => left.targetType.localeCompare(right.targetType) || left.targetValue.localeCompare(right.targetValue))
}

function isMockRecommendationMovieBlocked(movieId: string): boolean {
  return recommendationFeedbackMock.value.some(
    (item) =>
      item.targetType === "movie" &&
      (item.action === "not_interested" || item.action === "snooze") &&
      item.targetValue.toLocaleLowerCase() === movieId.toLocaleLowerCase(),
  )
}

function mockRecommendationItem(
  entry: ReturnType<typeof buildHomepagePortalModel>["recommendations"][number],
): HomepageRecommendationItemDTO {
  return {
    movieId: entry.movie.id,
    reasons: entry.reasons.length > 0 ? entry.reasons.map((reason) => ({ ...reason })) : [{ code: "catalog_discovery" }],
    feedbackEffects: mockFeedbackEffects(entry.movie),
  }
}

function normalizeNativePlayerPreset(
  preset: PlayerSettingsDTO["nativePlayerPreset"],
  command?: string,
): NativePlayerPreset {
  switch (preset) {
    case "mpv":
    case "potplayer":
    case "custom":
      return preset
  }
  const cmd = (command ?? "").trim().toLowerCase()
  if (cmd.includes("potplayer")) return "potplayer"
  if (cmd.includes("mpv")) return "mpv"
  return "custom"
}

function defaultNativePlayerCommand(preset: PlayerSettingsDTO["nativePlayerPreset"]): string {
  const normalized = normalizeNativePlayerPreset(preset)
  if (normalized === "potplayer") return "PotPlayerMini64.exe"
  if (normalized === "mpv") return "mpv"
  return ""
}

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
const mockActorExternalLinks = ref<Map<string, string[]>>(new Map())

function normalizeMockActorIdentity(value: string): string {
  return normalizeActorIdentity(value)
}

function resolveMockCanonicalActorName(value: string): string {
  let current = value.trim()
  const visited = new Set<string>()
  for (let index = 0; index < 32; index += 1) {
    const normalized = normalizeMockActorIdentity(current)
    if (!normalized || visited.has(normalized)) return current
    visited.add(normalized)
    const next = actorMergeStateMock.value.aliases[normalized]?.canonicalName.trim()
    if (!next) return current
    current = next
  }
  return current
}

function mockActorAliasesFor(canonicalName: string): string[] {
  const canonical = normalizeMockActorIdentity(canonicalName)
  return Object.values(actorMergeStateMock.value.aliases)
    .filter(
      (entry) =>
        normalizeMockActorIdentity(resolveMockCanonicalActorName(entry.canonicalName)) === canonical,
    )
    .map((entry) => entry.alias)
    .sort((left, right) => left.localeCompare(right))
}

function applyPersistedMockActorAliases(movie: Movie): Movie {
  const actors = [
    ...new Map(
      movie.actors.map((actor) => {
        const canonical = resolveMockCanonicalActorName(actor)
        return [normalizeMockActorIdentity(canonical), canonical] as const
      }),
    ).values(),
  ]
  return { ...movie, actors }
}

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

function buildMockCompletedTask(taskId: string, type: string, message = ""): TaskDTO {
  const now = new Date().toISOString()
  return {
    taskId,
    type,
    status: "completed",
    createdAt: now,
    startedAt: now,
    finishedAt: now,
    progress: 1,
    message,
  }
}

function findMockActor(name: string): ActorListItemDTO | undefined {
	const normalized = resolveMockCanonicalActorName(name)
  if (!normalized) {
    return undefined
  }
  return mockActorsFromMovies().find((actor) => actor.name === normalized)
}

function buildMockActorProfile(name: string): ActorProfileDTO {
  const actor = findMockActor(name)
  if (!actor) {
    throw mockHttpError(404, "COMMON_NOT_FOUND", "actor not found")
  }
  return {
    name: actor.name,
    avatarUrl: actor.avatarUrl,
    userTags: actor.userTags ?? [],
    externalLinks: [...(mockActorExternalLinks.value.get(actor.name) ?? [])],
    aliases: mockActorAliasesFor(actor.name),
    summary: "",
  }
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
    metadataProvider: ["JavBus", "FANZA", "JavLibrary"][index % 3],
    previewImages: [
      `https://picsum.photos/seed/jav-p1-${prefix}-${serial}/640/360`,
      `https://picsum.photos/seed/jav-p2-${prefix}-${serial}/640/360`,
      `https://picsum.photos/seed/jav-p3-${prefix}-${serial}/640/360`,
    ],
  }
}

loadMockMoviePrefs()

const moviesState = ref<Movie[]>(
  Array.from({ length: 180 }, (_, index) =>
    applyPersistedMockActorAliases(mergeMockPrefsIntoMovie(buildMovie(index))),
  ),
)

const mockActorMergeProfileFieldNames = [
  "avatarRemoteUrl",
  "avatarLocalPath",
  "summary",
  "homepage",
  "provider",
  "providerActorId",
  "height",
  "birthday",
] as const

function mockActorMergeProfileFields(): ActorMergeProfileFieldDTO[] {
  return mockActorMergeProfileFieldNames.map((field) => ({
    field,
    sourceValue: "",
    targetValue: "",
    defaultSelection: "target",
    conflict: false,
  }))
}

function mockActorMergeError(code: string, message = code): HttpClientError {
  const status = code === "ACTOR_MERGE_NOT_FOUND" ? 404 : code === "ACTOR_MERGE_INVALID" ? 400 : 409
  return mockHttpError(status, code, message)
}

function mockActorMergeStableUnique(values: string[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const raw of values) {
    const value = raw.trim()
    const key = value
    if (!key || seen.has(key)) continue
    seen.add(key)
    result.push(value)
  }
  return result
}

function mockActorMergeStableUniqueActors(values: string[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const raw of values) {
    const value = raw.trim()
    const key = normalizeMockActorIdentity(value)
    if (!key || seen.has(key)) continue
    seen.add(key)
    result.push(value)
  }
  return result
}

function mockActorMergeSummary(source: string[], target: string[]) {
  const targetSet = new Set(target)
  const duplicateCount = source.filter((value) => targetSet.has(value)).length
  return {
    sourceCount: source.length,
    targetCount: target.length,
    duplicateCount,
    resultCount: source.length + target.length - duplicateCount,
  }
}

function mockActorMergeFeedbackSummary(sourceNames: Set<string>, targetNames: Set<string>) {
  let sourceCount = 0
  let targetCount = 0
  for (const item of recommendationFeedbackMock.value) {
    if (item.action !== "less" || item.targetType !== "actor") continue
    const normalized = normalizeMockActorIdentity(item.targetValue)
    if (sourceNames.has(normalized)) sourceCount += 1
    else if (targetNames.has(normalized)) targetCount += 1
  }
  const total = sourceCount + targetCount
  return {
    sourceCount,
    targetCount,
    duplicateCount: total > 0 ? total - 1 : 0,
    resultCount: total > 0 ? 1 : 0,
  }
}

function mockActorMergeHash(value: unknown): string {
  const text = JSON.stringify(value)
  let hash = 0xcbf29ce484222325n
  const prime = 0x100000001b3n
  for (let index = 0; index < text.length; index += 1) {
    hash ^= BigInt(text.charCodeAt(index))
    hash = BigInt.asUintN(64, hash * prime)
  }
  return hash.toString(16).padStart(16, "0")
}

function previewMockActorMerge(body: ActorMergePreviewRequest): ActorMergePreviewDTO {
  const sourceInput = body.sourceName.trim()
  const targetInput = body.targetName.trim()
  if (!sourceInput || !targetInput) {
    throw mockActorMergeError("ACTOR_MERGE_INVALID", "sourceName and targetName are required")
  }
  const actors = mockActorsFromMovies()
  const source = actors.find((actor) => actor.name === sourceInput)
  if (!source) {
    const resolved = resolveMockCanonicalActorName(sourceInput)
    if (normalizeMockActorIdentity(resolved) !== normalizeMockActorIdentity(sourceInput) && findMockActor(resolved)) {
      throw mockActorMergeError("ACTOR_MERGE_SOURCE_IS_ALIAS")
    }
    throw mockActorMergeError("ACTOR_MERGE_NOT_FOUND")
  }
  const targetName = resolveMockCanonicalActorName(targetInput)
  const target = actors.find(
    (actor) => normalizeMockActorIdentity(actor.name) === normalizeMockActorIdentity(targetName),
  )
  if (!target) throw mockActorMergeError("ACTOR_MERGE_NOT_FOUND")
  if (source.name === target.name) throw mockActorMergeError("ACTOR_MERGE_SELF")

  const sourceAliases = mockActorAliasesFor(source.name)
  const targetAliases = mockActorAliasesFor(target.name)
  const sourceNames = new Set(
    [source.name, ...sourceAliases].map(normalizeMockActorIdentity),
  )
  const targetNames = new Set(
    [target.name, ...targetAliases].map(normalizeMockActorIdentity),
  )
  const sourceMovieIds = moviesState.value
    .filter((movie) => movie.actors.some((actor) => sourceNames.has(normalizeMockActorIdentity(actor))))
    .map((movie) => movie.id)
    .sort()
  const targetMovieIds = moviesState.value
    .filter((movie) => movie.actors.some((actor) => targetNames.has(normalizeMockActorIdentity(actor))))
    .map((movie) => movie.id)
    .sort()
  const sourceTags = [...(mockActorUserTags.value.get(source.name) ?? [])].sort()
  const targetTags = [...(mockActorUserTags.value.get(target.name) ?? [])].sort()
  const sourceLinks = [...(mockActorExternalLinks.value.get(source.name) ?? [])]
  const targetLinks = [...(mockActorExternalLinks.value.get(target.name) ?? [])]
  const aliasesToMove = mockActorMergeStableUniqueActors([source.name, ...sourceAliases])
  const blockingReasons: { code: string; message: string }[] = []
  const externalLinks = {
    source: sourceLinks,
    target: targetLinks,
    result: mockActorMergeStableUnique([...targetLinks, ...sourceLinks]),
  }
  if (externalLinks.result.length > 16) {
    blockingReasons.push({
      code: "ACTOR_MERGE_LINK_LIMIT",
      message: `merged external links would contain ${externalLinks.result.length} entries; maximum is 16`,
    })
  }
  for (const alias of aliasesToMove) {
    const normalized = normalizeMockActorIdentity(alias)
    const collision = actors.find(
      (actor) =>
        actor.name !== source.name &&
        actor.name !== target.name &&
        normalizeMockActorIdentity(actor.name) === normalized,
    )
    const aliasOwner = actorMergeStateMock.value.aliases[normalized]?.canonicalName
    if (collision || (aliasOwner && normalizeMockActorIdentity(resolveMockCanonicalActorName(aliasOwner)) !== normalizeMockActorIdentity(target.name))) {
      blockingReasons.push({
        code: "ACTOR_MERGE_CONFLICT",
        message: `alias ${alias} conflicts with another actor identity`,
      })
    }
  }
  const preview: ActorMergePreviewDTO = {
    previewToken: "",
    source: {
      id: actors.findIndex((actor) => actor.name === source.name) + 1,
      name: source.name,
      aliases: sourceAliases,
    },
    target: {
      id: actors.findIndex((actor) => actor.name === target.name) + 1,
      name: target.name,
      aliases: targetAliases,
    },
    movies: mockActorMergeSummary(sourceMovieIds, targetMovieIds),
    userTags: {
      source: sourceTags,
      target: targetTags,
      result: mockActorMergeStableUnique([...targetTags, ...sourceTags]),
    },
    externalLinks,
    recommendationFeedback: mockActorMergeFeedbackSummary(sourceNames, targetNames),
    curatedFramesAffected: 0,
    aliasesToMove,
    profileFields: mockActorMergeProfileFields(),
    canApply: blockingReasons.length === 0,
    blockingReasons,
    requiredDecisions: [],
  }
  preview.previewToken = mockActorMergeHash({
    preview,
    sourceMovieIds,
    targetMovieIds,
    feedback: recommendationFeedbackMock.value,
    aliases: actorMergeStateMock.value.aliases,
  })
  return preview
}

function applyMockActorMerge(body: ApplyActorMergeRequest): ActorMergeAuditDTO {
  if (!body.confirm || !body.previewToken.trim()) {
    throw mockActorMergeError("ACTOR_MERGE_INVALID", "confirm and previewToken are required")
  }
  const preview = previewMockActorMerge(body)
  if (preview.previewToken !== body.previewToken.trim()) {
    throw mockActorMergeError("ACTOR_MERGE_STALE_PREVIEW")
  }
  if (!preview.canApply) {
    const blocker = preview.blockingReasons[0]
    throw mockActorMergeError(blocker?.code ?? "ACTOR_MERGE_CONFLICT", blocker?.message)
  }
  const knownFields = new Set(mockActorMergeProfileFieldNames)
  for (const [field, selection] of Object.entries(body.profileDecisions ?? {})) {
    if (!knownFields.has(field as (typeof mockActorMergeProfileFieldNames)[number]) || !["source", "target"].includes(selection)) {
      throw mockActorMergeError("ACTOR_MERGE_INVALID", `invalid profile decision for ${field}`)
    }
  }

  const sourceNames = new Set(
    [preview.source.name, ...preview.source.aliases].map(normalizeMockActorIdentity),
  )
  moviesState.value = moviesState.value.map((movie) => ({
    ...movie,
    actors: mockActorMergeStableUniqueActors(
      movie.actors.map((actor) =>
        sourceNames.has(normalizeMockActorIdentity(actor)) ? preview.target.name : actor,
      ),
    ),
  }))

  const nextTags = new Map(mockActorUserTags.value)
  nextTags.set(preview.target.name, [...preview.userTags.result])
  nextTags.delete(preview.source.name)
  mockActorUserTags.value = nextTags
  const nextLinks = new Map(mockActorExternalLinks.value)
  nextLinks.set(preview.target.name, [...preview.externalLinks.result])
  nextLinks.delete(preview.source.name)
  mockActorExternalLinks.value = nextLinks

  const feedbackByKey = new Map<string, RecommendationFeedbackDTO>()
  for (const item of recommendationFeedbackMock.value) {
    const next =
      item.action === "less" &&
      item.targetType === "actor" &&
      sourceNames.has(normalizeMockActorIdentity(item.targetValue))
        ? { ...item, targetValue: preview.target.name, updatedAt: new Date().toISOString() }
        : item
    const key = `${next.action}\u0000${next.targetType}\u0000${mockRecommendationFeedbackTargetKey(next.targetType, next.targetValue)}`
    if (!feedbackByKey.has(key)) feedbackByKey.set(key, next)
  }
  persistMockRecommendationFeedback([...feedbackByKey.values()])

  const aliases = { ...actorMergeStateMock.value.aliases }
  for (const [normalizedAlias, entry] of Object.entries(aliases)) {
    if (
      sourceNames.has(
        normalizeMockActorIdentity(resolveMockCanonicalActorName(entry.canonicalName)),
      )
    ) {
      aliases[normalizedAlias] = { ...entry, canonicalName: preview.target.name }
    }
  }
  for (const alias of preview.aliasesToMove) {
    aliases[normalizeMockActorIdentity(alias)] = {
      alias,
      canonicalName: preview.target.name,
    }
  }
  const appliedAt = new Date().toISOString()
  const profileDecisions = Object.fromEntries(
    preview.profileFields.map((field) => [
      field.field,
      body.profileDecisions?.[field.field] ?? field.defaultSelection,
    ]),
  ) as Record<string, "source" | "target">
  const audit: ActorMergeAuditDTO = {
    id: `amrg_${globalThis.crypto?.randomUUID?.() ?? `${Date.now()}_${Math.random().toString(16).slice(2)}`}`,
    sourceActorId: preview.source.id,
    targetActorId: preview.target.id,
    sourceName: preview.source.name,
    targetName: preview.target.name,
    previewToken: preview.previewToken,
    appliedAt,
    summary: {
      movies: preview.movies,
      userTags: [...preview.userTags.result],
      externalLinks: [...preview.externalLinks.result],
      aliases: mockActorMergeStableUniqueActors([...preview.target.aliases, ...preview.aliasesToMove]),
      recommendationFeedback: preview.recommendationFeedback,
      curatedFramesAffected: preview.curatedFramesAffected,
      profileDecisions,
    },
  }
  actorMergeStateMock.value = {
    aliases,
    audits: [audit, ...actorMergeStateMock.value.audits].slice(0, 500),
  }
  saveLocalActorMergeState(
    actorMergeStateMock.value,
    typeof localStorage === "undefined" ? undefined : localStorage,
  )
  return audit
}

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
  moviesLoaded: computed(() => true),
  loadError: computed(() => null),
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
  libraryPathStorageStatuses: computed(() => libraryPathStorageStatusesMock.value),
  savedViews: computed(() => savedViewsMock.value),
  defaultImportLibraryPathId: computed(() => defaultImportLibraryPathIdMock.value),
  backupDirectory: computed(() => backupDirectoryMock.value),
  organizeLibrary: computed(() => organizeLibraryMock.value),
  autoLibraryWatch: computed(() => autoLibraryWatchMock.value),
  autoActorProfileScrape: computed(() => autoActorProfileScrapeMock.value),
  autoDownloadUpdates: computed(() => autoDownloadUpdatesMock.value),
  launchAtLogin: computed(() => launchAtLoginMock.value),
  launchAtLoginSupported: computed(() => launchAtLoginSupportedMock.value),
  curatedFrameExportFormat: computed(() => curatedFrameExportFormatMock.value),
  curatedFrameExportMode: computed(() => curatedFrameExportModeMock.value),
  metadataMovieProvider: computed(() => metadataMovieProviderMock.value),
  metadataMovieProviders: computed(() => metadataMovieProvidersMock.value),
  metadataMovieProviderChain: computed(() => metadataMovieProviderChainMock.value),
  metadataMovieScrapeMode: computed(() => metadataMovieScrapeModeMock.value),
  proxy: computed(() => proxyMock.value),
  aiProvider: computed(() => aiProviderMock.value),
  playerSettings: computed(() => playerSettingsMock.value),
  backendLog: computed(() => backendLogMock.value),

  async setProxy(config: import("@/api/types").ProxySettingsDTO) {
    proxyMock.value = { ...config }
  },

  async setAIProvider(patch: import("@/api/types").PatchAIProviderBody) {
    const prev = aiProviderMock.value
    const next: import("@/api/types").AIProviderSettingsDTO = {
      ...prev,
      ...(patch.baseUrl !== undefined ? { baseUrl: patch.baseUrl } : {}),
      ...(patch.apiKey !== undefined ? { apiKey: patch.apiKey } : {}),
      ...(patch.model !== undefined ? { model: patch.model } : {}),
    }
    aiProviderMock.value = next
    persistAIProviderMock(next)
  },

  async testAIProvider(
    provider?: import("@/api/types").AIProviderSettingsDTO,
  ): Promise<import("@/api/types").AIProviderTestResponse> {
    const target = provider ?? aiProviderMock.value
    if (!target.baseUrl.trim() || !target.model.trim()) {
      return { ok: false, latencyMs: 0, message: "mock: baseUrl and model are required" }
    }
    await new Promise((resolve) => setTimeout(resolve, 200))
    return { ok: true, latencyMs: 42 }
  },

  async patchPlayerSettings(patch: PatchPlayerSettingsBody) {
    const prev = playerSettingsMock.value
    playerSettingsMock.value = {
      hardwareDecode:
        patch.hardwareDecode !== undefined ? patch.hardwareDecode : prev.hardwareDecode,
      hardwareEncoder:
        patch.hardwareEncoder !== undefined ? patch.hardwareEncoder : prev.hardwareEncoder,
      nativePlayerPreset:
        patch.nativePlayerPreset !== undefined
          ? patch.nativePlayerPreset
          : prev.nativePlayerPreset,
      nativePlayerEnabled:
        patch.nativePlayerEnabled !== undefined
          ? patch.nativePlayerEnabled
          : prev.nativePlayerEnabled,
      nativePlayerCommand:
        patch.nativePlayerCommand !== undefined
          ? patch.nativePlayerCommand.trim()
          : prev.nativePlayerCommand,
      streamPushEnabled:
        patch.streamPushEnabled !== undefined
          ? patch.streamPushEnabled
          : prev.streamPushEnabled,
      forceStreamPush:
        patch.forceStreamPush !== undefined ? patch.forceStreamPush : prev.forceStreamPush,
      ffmpegCommand:
        patch.ffmpegCommand !== undefined
          ? (patch.ffmpegCommand.trim() || "ffmpeg")
          : prev.ffmpegCommand,
      preferNativePlayer:
        patch.preferNativePlayer !== undefined
          ? patch.preferNativePlayer
          : prev.preferNativePlayer,
      seekForwardStepSec:
        patch.seekForwardStepSec !== undefined
          ? Math.max(1, patch.seekForwardStepSec)
          : prev.seekForwardStepSec,
      seekBackwardStepSec:
        patch.seekBackwardStepSec !== undefined
          ? Math.max(1, patch.seekBackwardStepSec)
          : prev.seekBackwardStepSec,
    }
    if (!playerSettingsMock.value.streamPushEnabled) {
      playerSettingsMock.value = {
        ...playerSettingsMock.value,
        forceStreamPush: false,
      }
    }
    playerSettingsMock.value = {
      ...playerSettingsMock.value,
      nativePlayerPreset: normalizeNativePlayerPreset(
        playerSettingsMock.value.nativePlayerPreset,
        playerSettingsMock.value.nativePlayerCommand,
      ),
      nativePlayerCommand:
        (playerSettingsMock.value.nativePlayerCommand ??
          defaultNativePlayerCommand(playerSettingsMock.value.nativePlayerPreset)).trim() ||
        defaultNativePlayerCommand(playerSettingsMock.value.nativePlayerPreset),
    }
  },

  async patchBackendLog(patch: PatchBackendLogBody) {
    const prev = backendLogMock.value
    backendLogMock.value = {
      logDir: patch.logDir !== undefined ? patch.logDir : prev.logDir,
      logFilePrefix:
        patch.logFilePrefix !== undefined ? patch.logFilePrefix : prev.logFilePrefix,
      logMaxAgeDays:
        patch.logMaxAgeDays !== undefined ? patch.logMaxAgeDays : prev.logMaxAgeDays,
      logLevel: patch.logLevel !== undefined ? patch.logLevel : prev.logLevel,
    }
  },

  async health(): Promise<HealthDTO> {
    return {
      name: "curated-mock",
      version: "mock",
      channel: "dev",
      transport: "mock",
      databasePath: "mock",
    }
  },

  async createBackup(): Promise<never> {
    throw new Error("Backup maintenance requires Web API mode")
  },

  async createMovieClip(): Promise<never> {
    throw new Error("GIF clip export requires Web API mode")
  },

  async verifyBackup(): Promise<never> {
    throw new Error("Backup maintenance requires Web API mode")
  },

  async preflightBackupRestore(): Promise<never> {
    throw new Error("Backup maintenance requires Web API mode")
  },

  async scanLibraryHealth(): Promise<never> {
    throw new Error("Library health requires Web API mode")
  },

  async startLibraryHealthRepair(): Promise<never> {
    throw new Error("Library health repair requires Web API mode")
  },

  async getLibraryHealthRepair(): Promise<never> {
    throw new Error("Library health repair requires Web API mode")
  },

  async startLibraryHealthAction(): Promise<never> {
    throw new Error("Library health cleanup requires Web API mode")
  },

  async listConnectedClients(): Promise<ConnectedClientsDTO> {
    return buildMockConnectedClients()
  },

  async pingProxyJavbus(): Promise<ProxyJavBusPingResponse> {
    return { ok: true, latencyMs: 0, httpStatus: 200, message: "mock" }
  },

  async pingProxyGoogle(): Promise<ProxyJavBusPingResponse> {
    return { ok: true, latencyMs: 0, httpStatus: 200, message: "mock" }
  },

  async pingProvider(name: string): Promise<ProviderHealthDTO> {
    return {
      name: name.trim(),
      status: "ok",
      latencyMs: 0,
      message: "mock",
    }
  },

  async pingAllProviders(): Promise<PingAllProvidersResponse> {
    return {
      providers: [],
      total: 0,
      ok: 0,
      fail: 0,
    }
  },

  async getHomepageDailyRecommendations(): Promise<HomepageDailyRecommendationsDTO> {
    refreshActiveMockRecommendationFeedback()
    const dateUtc = getCurrentUtcDayKey()
    const model = buildHomepagePortalModel({
      movies: moviesState.value.filter((movie) => !isMockRecommendationMovieBlocked(movie.id)),
      daySeed: dateUtc,
    })
    const recommendations = model.recommendations
      .map((entry, index) => ({ entry, index, effects: mockFeedbackEffects(entry.movie) }))
      .sort((left, right) => left.effects.length - right.effects.length || left.index - right.index)
      .map(({ entry }) => mockRecommendationItem(entry))
    return {
      dateUtc,
      generatedAt: `${dateUtc}T00:00:00Z`,
      generationVersion: "mock-v2",
      heroMovieIds: model.heroMovies.map((movie) => movie.id),
      recommendationMovieIds: recommendations.map((item) => item.movieId),
      recommendations,
    }
  },

  async refreshHomepageDailyRecommendations(
    body?: RefreshHomepageDailyRecommendationsBody,
  ): Promise<HomepageDailyRecommendationsDTO> {
    const snapshot = await this.getHomepageDailyRecommendations()
    const preservedHeroMovieIds = body?.preserveHeroMovieIds
      ?.map((movieId) => movieId.trim())
      .filter(Boolean)
    const excludedRecommendationMovieIds = new Set(
      body?.excludeRecommendationMovieIds
        ?.map((movieId) => movieId.trim())
        .filter(Boolean) ?? [],
    )

    if (!preservedHeroMovieIds || preservedHeroMovieIds.length === 0) {
      return snapshot
    }

    const blockedMovieIds = new Set([...preservedHeroMovieIds, ...excludedRecommendationMovieIds])
    const recommendationMovieIds = [
      ...snapshot.recommendationMovieIds.filter((movieId) => !blockedMovieIds.has(movieId)),
      ...moviesState.value
        .map((movie) => movie.id)
        .filter((movieId) => !blockedMovieIds.has(movieId) && !isMockRecommendationMovieBlocked(movieId)),
    ].slice(0, Math.max(snapshot.recommendationMovieIds.length, 6))

    const recommendationByMovieID = new Map(snapshot.recommendations.map((item) => [item.movieId, item]))
    const recommendations = recommendationMovieIds.map((movieId) => recommendationByMovieID.get(movieId) ?? {
      movieId,
      reasons: [{ code: "catalog_discovery" as const }],
      feedbackEffects: mockFeedbackEffects(moviesState.value.find((movie) => movie.id === movieId)!),
    })

    return {
      ...snapshot,
      heroMovieIds: preservedHeroMovieIds,
      recommendationMovieIds,
      recommendations,
    }
  },

  async listHomepageRecommendationFeedback(): Promise<RecommendationFeedbackListDTO> {
    refreshActiveMockRecommendationFeedback()
    return { items: recommendationFeedbackMock.value.map((item) => ({ ...item })) }
  },

  async createHomepageRecommendationFeedback(
    body: CreateRecommendationFeedbackBody,
  ): Promise<RecommendationFeedbackDTO> {
    refreshActiveMockRecommendationFeedback()
    if (!isValidMockFeedbackShape(body)) {
      throw mockHttpError(400, "RECOMMENDATION_FEEDBACK_INVALID")
    }
    const movie = moviesState.value.find(
      (candidate) => candidate.id === body.sourceMovieId.trim() && !candidate.trashedAt,
    )
    if (!movie) throw mockHttpError(404, "RECOMMENDATION_FEEDBACK_TARGET_NOT_FOUND")
    const targetValue = canonicalMockFeedbackTarget(movie, body.targetType, body.targetValue)
    if (!targetValue) throw mockHttpError(404, "RECOMMENDATION_FEEDBACK_TARGET_NOT_FOUND")
    const key = `${body.action}\u0000${body.targetType}\u0000${mockRecommendationFeedbackTargetKey(body.targetType, targetValue)}`
    const existing = recommendationFeedbackMock.value.find(
      (item) => `${item.action}\u0000${item.targetType}\u0000${mockRecommendationFeedbackTargetKey(item.targetType, item.targetValue)}` === key,
    )
    if (existing) return { ...existing }
    if (recommendationFeedbackMock.value.length >= 500) {
      throw mockHttpError(409, "RECOMMENDATION_FEEDBACK_LIMIT_REACHED")
    }
    const now = new Date()
    const item: RecommendationFeedbackDTO = {
      id: mockRecommendationFeedbackID(),
      action: body.action,
      targetType: body.targetType,
      targetValue,
      sourceMovieId: movie.id,
      ...(body.action === "snooze"
        ? { expiresAt: new Date(now.getTime() + body.durationDays! * 86_400_000).toISOString() }
        : {}),
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    }
    persistMockRecommendationFeedback([item, ...recommendationFeedbackMock.value])
    return { ...item }
  },

  async deleteHomepageRecommendationFeedback(id: string): Promise<void> {
    const next = recommendationFeedbackMock.value.filter((item) => item.id !== id.trim())
    if (next.length === recommendationFeedbackMock.value.length) {
      throw mockHttpError(404, "RECOMMENDATION_FEEDBACK_TARGET_NOT_FOUND")
    }
    persistMockRecommendationFeedback(next)
  },

  async refreshSettings() {
    // Mock: paths are in-memory only; no remote settings.
  },

  async checkLibraryPathStorageStatus(libraryPathIds?: string[]) {
    const wanted = new Set((libraryPathIds ?? []).map((id) => id.trim()).filter(Boolean))
    const now = new Date().toISOString()
    const checked = libraryPathsState.value
      .filter((path) => wanted.size === 0 || wanted.has(path.id))
      .map((path) => {
        const existing = libraryPathStorageStatusesMock.value.find(
          (status) => status.libraryPathId === path.id,
        )
        return {
          libraryPathId: path.id,
          path: path.path,
          title: path.title,
          status: existing?.status ?? "online",
          message: existing?.message ?? "Storage path is online.",
          checkedAt: now,
          rootPath: existing?.rootPath,
          driveType: existing?.driveType,
          volumeLabel: existing?.volumeLabel,
          identityConfidence: existing?.identityConfidence ?? "mock",
          expectedVolumeId: existing?.expectedVolumeId,
          currentVolumeId: existing?.currentVolumeId,
          canRescan: existing?.status ? existing.canRescan : true,
          canImport: existing?.status ? existing.canImport : true,
        }
      })
    if (wanted.size === 0) {
      libraryPathStorageStatusesMock.value = checked
      return
    }
    const next = new Map(libraryPathStorageStatusesMock.value.map((status) => [status.libraryPathId, status]))
    for (const status of checked) {
      next.set(status.libraryPathId, status)
    }
    libraryPathStorageStatusesMock.value = [...next.values()]
  },

  async rebindLibraryPathStorage(id: string) {
    const trimmed = id.trim()
    const path = libraryPathsState.value.find((entry) => entry.id === trimmed)
    if (!path) {
      throw mockHttpError(404, "COMMON_NOT_FOUND", "library path not found")
    }
    const now = new Date().toISOString()
    const rebound = {
      libraryPathId: trimmed,
      path: path.path,
      title: path.title,
      status: "online" as const,
      message: "Storage path is online.",
      checkedAt: now,
      identityConfidence: "mock",
      canRescan: true,
      canImport: true,
    }
    const found = libraryPathStorageStatusesMock.value.some((status) => status.libraryPathId === trimmed)
    libraryPathStorageStatusesMock.value = found
      ? libraryPathStorageStatusesMock.value.map((status) =>
          status.libraryPathId === trimmed
            ? {
                ...status,
                ...rebound,
                expectedVolumeId: status.currentVolumeId ?? status.expectedVolumeId,
              }
            : status,
        )
      : [...libraryPathStorageStatusesMock.value, rebound]
  },

  async reloadMoviesFromApi() {
    // Mock: 列表为本地种子，无远端同步。
  },

  async listMoviesForExport() {
    return moviesState.value.filter((movie) => !movie.trashedAt?.trim())
  },

  async ensureTrashLoaded() {
    // Mock: trash list is already derived from in-memory state.
  },

  async refreshSavedViews() {
    savedViewsMock.value = loadLocalSavedViews(
      typeof localStorage === "undefined" ? undefined : localStorage,
    )
  },

  async createSavedView(name: string, filters: SavedViewFiltersV1): Promise<SavedViewDTO> {
    if (savedViewsMock.value.length >= 50) {
      throw mockHttpError(409, "SAVED_VIEW_LIMIT_REACHED", "saved view limit reached")
    }
    let normalizedName: string
    let normalizedFilters: SavedViewFiltersV1
    try {
      normalizedName = normalizeSavedViewName(name)
      normalizedFilters = normalizeSavedViewFiltersV1(filters)
    } catch (error) {
      throw mockHttpError(400, "SAVED_VIEW_INVALID", error instanceof Error ? error.message : "invalid saved view")
    }
    const nameKey = normalizedSavedViewNameKey(normalizedName)
    if (savedViewsMock.value.some((item) => normalizedSavedViewNameKey(item.name) === nameKey)) {
      throw mockHttpError(409, "SAVED_VIEW_NAME_CONFLICT", "saved view name already exists")
    }
    const now = new Date().toISOString()
    const created: SavedViewDTO = {
      id: newMockSavedViewID(),
      name: normalizedName,
      filters: normalizedFilters,
      sortOrder: savedViewsMock.value.length,
      createdAt: now,
      updatedAt: now,
    }
    persistMockSavedViews([...savedViewsMock.value, created])
    return created
  },

  async updateSavedView(
    id: string,
    patch: { name?: string; filters?: SavedViewFiltersV1 },
  ): Promise<SavedViewDTO> {
    const index = savedViewsMock.value.findIndex((item) => item.id === id.trim())
    if (index < 0) {
      throw mockHttpError(404, "COMMON_NOT_FOUND", "saved view not found")
    }
    const current = savedViewsMock.value[index]!
    let name = current.name
    let filters = current.filters
    try {
      if (patch.name !== undefined) name = normalizeSavedViewName(patch.name)
      if (patch.filters !== undefined) filters = normalizeSavedViewFiltersV1(patch.filters)
    } catch (error) {
      throw mockHttpError(400, "SAVED_VIEW_INVALID", error instanceof Error ? error.message : "invalid saved view")
    }
    const nameKey = normalizedSavedViewNameKey(name)
    if (
      savedViewsMock.value.some(
        (item, itemIndex) => itemIndex !== index && normalizedSavedViewNameKey(item.name) === nameKey,
      )
    ) {
      throw mockHttpError(409, "SAVED_VIEW_NAME_CONFLICT", "saved view name already exists")
    }
    const updated: SavedViewDTO = {
      ...current,
      name,
      filters,
      updatedAt: new Date().toISOString(),
    }
    persistMockSavedViews(
      savedViewsMock.value.map((item, itemIndex) => (itemIndex === index ? updated : item)),
    )
    return updated
  },

  async deleteSavedView(id: string) {
    const trimmed = id.trim()
    if (!savedViewsMock.value.some((item) => item.id === trimmed)) {
      throw mockHttpError(404, "COMMON_NOT_FOUND", "saved view not found")
    }
    persistMockSavedViews(savedViewsMock.value.filter((item) => item.id !== trimmed))
  },

  async reorderSavedViews(ids: string[]) {
    const currentIDs = new Set(savedViewsMock.value.map((item) => item.id))
    const requested = new Set(ids)
    if (
      ids.length !== savedViewsMock.value.length ||
      requested.size !== ids.length ||
      ids.some((id) => !currentIDs.has(id))
    ) {
      throw mockHttpError(400, "SAVED_VIEW_INVALID", "saved view order must contain every view exactly once")
    }
    const byID = new Map(savedViewsMock.value.map((item) => [item.id, item]))
    persistMockSavedViews(ids.map((id) => byID.get(id)!))
  },

  async setOrganizeLibrary(value: boolean) {
    organizeLibraryMock.value = value
  },

  async setBackupDirectory(directory: string) {
    backupDirectoryMock.value = directory.trim()
  },

  async setAutoLibraryWatch(value: boolean) {
    autoLibraryWatchMock.value = value
  },

  async setAutoActorProfileScrape(value: boolean) {
    autoActorProfileScrapeMock.value = value
  },

  async setAutoDownloadUpdates(value: boolean) {
    autoDownloadUpdatesMock.value = value
  },

  async setLaunchAtLogin(value: boolean) {
    launchAtLoginMock.value = value
  },

  async setCuratedFrameExportFormat(format: CuratedFrameExportFormat) {
    curatedFrameExportFormatMock.value = format
  },
  async setCuratedFrameExportMode(mode: CuratedFrameExportMode) {
    curatedFrameExportModeMock.value = mode
    persistCuratedFrameExportMode(mode)
  },

  async setMetadataMovieProvider(name: string) {
    const trimmed = name.trim()
    if (trimmed !== "" && metadataMovieProvidersMock.value.length === 0) {
      throw mockHttpError(
        400,
        "MOCK_METADATA_PROVIDER_UNAVAILABLE",
        "Mock mode has no provider list; use Web API to pick a source.",
      )
    }
    if (
      trimmed !== "" &&
      !metadataMovieProvidersMock.value.some((p) => p.toLowerCase() === trimmed.toLowerCase())
    ) {
      throw mockHttpError(400, "COMMON_BAD_REQUEST", "Unknown metadata provider in mock.")
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
      throw mockHttpError(400, "COMMON_BAD_REQUEST", "library path must be an absolute path")
    }
    const id = `mock-${Date.now()}`
    libraryPathsState.value = [
      ...libraryPathsState.value,
      { id, path: trimmed, title: (title?.trim() || trimmed) },
    ]
    libraryPathStorageStatusesMock.value = [
      ...libraryPathStorageStatusesMock.value,
      {
        libraryPathId: id,
        path: trimmed,
        title: title?.trim() || trimmed,
        status: "online",
        message: "Storage path is online.",
        checkedAt: new Date().toISOString(),
        identityConfidence: "mock",
        canRescan: true,
        canImport: true,
      },
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
    libraryPathStorageStatusesMock.value = libraryPathStorageStatusesMock.value.filter(
      (status) => status.libraryPathId !== id,
    )
    if (defaultImportLibraryPathIdMock.value === id) {
      defaultImportLibraryPathIdMock.value = libraryPathsState.value[0]?.id ?? ""
    }
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

  async revealLibraryPathInFileManager() {
    throw mockHttpError(501, "MOCK_REVEAL_NOT_SUPPORTED")
  },

  async setDefaultImportLibraryPathId(id: string) {
    const trimmed = id.trim()
    if (trimmed && !libraryPathsState.value.some((path) => path.id === trimmed)) {
      throw mockHttpError(400, "COMMON_BAD_REQUEST", "unknown defaultImportLibraryPathId")
    }
    defaultImportLibraryPathIdMock.value = trimmed
  },

  async checkImportMovieCodes(names: string[]): Promise<ImportMovieCodeCheckDTO> {
    const trimmed = names.map((name) => name.trim()).filter(Boolean)
    if (trimmed.length === 0) {
      throw mockHttpError(400, "COMMON_BAD_REQUEST", "filenames are required")
    }
    const active = moviesState.value.filter((movie) => !movie.trashedAt?.trim())
    const items = trimmed.map((name) => {
      const extractedCode = extractMovieNumber(name)
      const matches: ImportMovieCodeMatchDTO[] = []
      if (extractedCode) {
        const byId = new Map<string, ImportMovieCodeMatchDTO>()
        for (const movie of active) {
          const kind = strongerMovieCodeMatch(
            classifyMovieCodes(extractedCode, movie.code),
            classifyMovieCodes(extractedCode, movie.id),
          )
          if (!kind) continue
          const prev = byId.get(movie.id)
          byId.set(movie.id, {
            movieId: movie.id,
            code: movie.code || extractedCode,
            title: movie.title,
            matchKind: prev ? (strongerMovieCodeMatch(prev.matchKind, kind) || kind) : kind,
          })
        }
        matches.push(...byId.values())
      }
      return {
        name,
        extractedCode: extractedCode || undefined,
        matches,
      }
    })
    return {
      items,
      matchedCount: items.filter((item) => item.matches.length > 0).length,
    }
  },

  async importMovies(files: File[]): Promise<TaskDTO | null> {
    const selected = files.filter((file) => file.name.trim())
    if (selected.length === 0) {
      return null
    }
    const targetId = defaultImportLibraryPathIdMock.value.trim()
    if (!targetId) {
      throw mockHttpError(
        400,
        "IMPORT_TARGET_NOT_CONFIGURED",
        "default import library path is not configured",
      )
    }
    const target = libraryPathsState.value.find((path) => path.id === targetId)
    if (!target) {
      throw mockHttpError(404, "IMPORT_TARGET_UNAVAILABLE", "default import library path was not found")
    }
    const now = new Date().toISOString()
    return {
      taskId: `mock-import-${Date.now()}`,
      type: "import.movies",
      status: "completed",
      createdAt: now,
      startedAt: now,
      finishedAt: now,
      progress: 100,
      message: "Movie import completed",
      metadata: {
        targetLibraryPathId: target.id,
        targetPath: target.path,
        totalFiles: selected.length,
        completedFiles: selected.length,
        failedFiles: 0,
        copiedBytes: selected.reduce((sum, file) => sum + file.size, 0),
        totalBytes: selected.reduce((sum, file) => sum + file.size, 0),
      },
    }
  },

  async scanLibraryPaths() {
    // Mock: no backend scan.
    return null
  },

  async listResumableMovieImports(): Promise<ResumableMovieImportSession[]> {
    // Mock 没有真实上传会话，不存在可恢复条目。
    return []
  },

  async abandonMovieImportUpload(): Promise<void> {
    // Mock 没有可放弃的续传会话。
  },

  async getTaskStatus(taskId: string): Promise<TaskDTO> {
    return buildMockCompletedTask(taskId.trim() || "mock-task", "mock")
  },

  async refreshMovieMetadata() {
    return null
  },

  async revealMovieInFileManager() {
    throw mockHttpError(501, "MOCK_REVEAL_NOT_SUPPORTED")
  },

  async refreshMetadataForLibraryPaths(): Promise<MetadataRefreshQueuedDTO> {
    return { queued: 0, skipped: 0, invalidPaths: [] }
  },

  async getMoviePlayback() {
    return null
  },

  prefetchMoviePlayback() {
    // Mock: descriptors resolve locally; nothing to prefetch.
  },

  async createPlaybackSession() {
    return null
  },

  async getPlaybackSession() {
    return null
  },

  async launchNativePlayback() {
    return null
  },

  async deletePlaybackSession() {
    // Mock: no playback sessions to release.
  },

  async ensureMovieCached() {
    // Mock 数据全在内存，无需远程补全。
  },

  getMovieById(movieId) {
    const id = movieId?.trim()
    if (!id) return undefined
    return moviesState.value.find((movie) => movie.id === id)
  },
  async loadMovieDetail(movieId: string) {
    const id = movieId.trim()
    if (!id) return undefined
    await Promise.resolve()
    return moviesState.value.find((movie) => movie.id === id)
  },
  getRelatedMovies(movieId, limit = 6) {
    const id = movieId.trim()
    const source = moviesState.value.find((movie) => movie.id === id)
    if (!source) return []
    const actors = new Set(
      source.actors.map((actor) => actor.trim().toLocaleLowerCase()).filter(Boolean),
    )
    if (actors.size === 0) return []
    return moviesState.value
      .filter(
        (movie) =>
          movie.id !== id &&
          !movie.trashedAt?.trim() &&
          movie.actors.some((actor) => actors.has(actor.trim().toLocaleLowerCase())),
      )
      .slice(0, Math.max(0, limit))
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
        if (mockActorAliasesFor(r.name).some((alias) => alias.toLowerCase().includes(q))) {
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
		const n = resolveMockCanonicalActorName(name)
    if (!n) {
      throw mockHttpError(400, "COMMON_BAD_REQUEST", "actor name is required")
    }
    if (!mockActorsFromMovies().some((r) => r.name === n)) {
      throw mockHttpError(404, "COMMON_NOT_FOUND", "actor not found")
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
      throw mockHttpError(404, "COMMON_NOT_FOUND", "actor not found")
    }
    return row
  },

  async getActorProfile(name: string): Promise<ActorProfileDTO> {
    return buildMockActorProfile(name)
  },

  async scrapeActorProfile(name: string): Promise<TaskDTO> {
    return buildMockCompletedTask(`mock-actor-scrape-${name.trim() || "unknown"}`, "actor-scrape")
  },

  async patchActorExternalLinks(name: string, externalLinks: string[]): Promise<ActorProfileDTO> {
    const profile = buildMockActorProfile(name)
    const normalized = externalLinks.map((link) => link.trim()).filter(Boolean)
    const next = new Map(mockActorExternalLinks.value)
    next.set(profile.name, normalized)
    mockActorExternalLinks.value = next
    return {
      ...profile,
      externalLinks: normalized,
    }
  },

  async previewActorMerge(body: ActorMergePreviewRequest): Promise<ActorMergePreviewDTO> {
    return previewMockActorMerge(body)
  },

  async applyActorMerge(body: ApplyActorMergeRequest): Promise<ActorMergeAuditDTO> {
    return applyMockActorMerge(body)
  },

  async listActorMergeAudits(params?: { limit?: number; offset?: number }): Promise<ActorMergeAuditListDTO> {
    const limit = params?.limit && params.limit > 0 ? Math.min(params.limit, 100) : 50
    const offset = params?.offset && params.offset > 0 ? params.offset : 0
    return {
      items: actorMergeStateMock.value.audits.slice(offset, offset + limit),
      total: actorMergeStateMock.value.audits.length,
      limit,
      offset,
    }
  },

  async getPersonalInsightsOverview(params: {
    range: PersonalInsightsRange
    timezone: string
  }): Promise<PersonalInsightsOverviewDTO> {
    return buildPersonalInsightsOverview({
      movies: moviesState.value,
      progressEntries: listSortedByUpdatedDesc(),
      watchTimeEntries: listPlaybackWatchTimeMovieEntries(),
      range: params.range,
      timezone: params.timezone,
    })
  },

  async getPersonalInsightsBreakdown(params: {
    range: PersonalInsightsRange
    timezone: string
    dimension: PersonalInsightsDimension
    limit?: number
  }): Promise<PersonalInsightsBreakdownDTO> {
    return buildPersonalInsightsBreakdown({
      movies: moviesState.value,
      progressEntries: listSortedByUpdatedDesc(),
      watchTimeEntries: listPlaybackWatchTimeMovieEntries(),
      range: params.range,
      timezone: params.timezone,
      dimension: params.dimension,
      limit: params.limit,
    })
  },

  async getMovieComment(movieId: string): Promise<MovieCommentDTO> {
    return getLocalMovieComment(movieId.trim())
  },

  async putMovieComment(movieId: string, body: PutMovieCommentBody): Promise<MovieCommentDTO> {
    return putLocalMovieComment(movieId.trim(), body.body)
  },

  async exportCuratedFrames(): Promise<{ blob: Blob; filename: string }> {
    throw mockHttpError(501, "MOCK_CURATED_EXPORT_NOT_SUPPORTED")
  },
}
