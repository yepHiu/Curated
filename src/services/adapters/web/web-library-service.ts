import { computed, ref, shallowRef, watch, type Ref } from "vue"
import { applyAIGovernance } from "@/lib/experimental-agent"
import type {
  BackendLogSettingsDTO,
  ActorMergeAuditDTO,
  ActorMergeAuditListDTO,
  ActorMergePreviewDTO,
  ActorMergePreviewRequest,
  ApplyActorMergeRequest,
  BackupManifestDTO,
  BackupRestorePreflightDTO,
  BackupVerificationDTO,
  CuratedFrameExportFormat,
  CuratedFrameExportMode,
  HomepageDailyRecommendationsDTO,
  CreateRecommendationFeedbackBody,
  RecommendationFeedbackDTO,
  RecommendationFeedbackListDTO,
  RefreshHomepageDailyRecommendationsBody,
  LibraryPathStorageStatusDTO,
  LibraryHealthRepairDTO,
  LibraryHealthReportDTO,
  NativePlayerPreset,
  ListActorsParams,
  MetadataMovieScrapeMode,
  MetadataRefreshQueuedDTO,
  NativePlaybackLaunchDTO,
  PersonalInsightsBreakdownDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
  PlaybackDescriptorDTO,
  PlaybackSessionStatusDTO,
  PatchBackendLogBody,
  PatchMovieBody,
  PatchPlayerSettingsBody,
  AIProviderSettingsDTO,
  AIProviderTestResponse,
  PatchAIProviderBody,
  PlayerSettingsDTO,
  ProxySettingsDTO,
  SettingsDTO,
  TaskDTO,
  StartLibraryHealthRepairBody,
  StartLibraryHealthActionBody,
  SavedViewDTO,
  SavedViewFiltersV1,
} from "@/api/types"
import { HttpClientError } from "@/api/http-client"
import { api, movieImportUploadFileManifests } from "@/api/endpoints"
import { moviePlaybackAbsoluteUrl } from "@/api/playback-url"
import type { LibrarySetting } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { i18n } from "@/i18n"
import { clientVideoCodecsQueryParam, resolvePlaybackCapabilities } from "@/lib/playback-capabilities"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { buildSettingsDashboardStats } from "@/lib/library-stats"
import {
  addMovieImportUploadLedgerEntry,
  loadMovieImportUploadLedger,
  removeMovieImportUploadLedgerEntry,
  touchMovieImportUploadLedgerEntry,
} from "@/lib/movie-import-upload-ledger"
import type { LibraryService, ResumableMovieImportSession } from "@/services/contracts/library-service"
import { normalizeHardwareEncoderPreference } from "@/lib/playback-settings-normalize"
import { mapMovieDetail, mapMovieListItem } from "./mappers"

const moviesState: Ref<Movie[]> = shallowRef([])
const moviesLoadedState = ref(false)
const loadErrorState = ref<string | null>(null)
const trashedMoviesState: Ref<Movie[]> = shallowRef([])
const libraryPathsState: Ref<LibrarySetting[]> = ref([])
const libraryPathStorageStatusesState: Ref<LibraryPathStorageStatusDTO[]> = ref([])
const savedViewsState: Ref<SavedViewDTO[]> = ref([])
const defaultImportLibraryPathIdState = ref("")
const backupDirectoryState = ref("")
/** 与后端 config.Default() / library-config.cfg 默认一致，避免首屏在 GET 完成前误显示为关 */
const organizeLibraryState = ref(true)
/** 与后端默认一致：关，避免误触新库「首次扫描」扩展逻辑 */
const autoLibraryWatchState = ref(true)
const autoActorProfileScrapeState = ref(false)
const autoDownloadUpdatesState = ref(false)
const launchAtLoginState = ref(false)
const launchAtLoginSupportedState = ref(false)
const lanEnabledState = ref(false)
const lanListeningState = ref(false)
const lanAccessUrlsState = ref<string[]>([])
const curatedFrameExportFormatState = ref<CuratedFrameExportFormat>("jpg")
const curatedFrameExportModeState = ref<CuratedFrameExportMode>("raw")
const metadataMovieProviderState = ref("")
const metadataMovieProvidersState = ref<string[]>([])
const metadataMovieProviderChainState = ref<string[]>([])
const metadataMovieScrapeModeState = ref<MetadataMovieScrapeMode>("auto")
const proxyState = ref<ProxySettingsDTO>({ enabled: false })
const aiProviderState = ref<AIProviderSettingsDTO>({ kind: "openai-compatible", baseUrl: "", model: "" })
let aiProviderSaveSeq = 0
const playerSettingsState = ref<PlayerSettingsDTO>({
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
const backendLogState = ref<BackendLogSettingsDTO>({
  logDir: "",
  logLevel: "info",
})
/** 设置页概览第三卡：萃取帧条数（GET /curated-frames 列表长度） */
const curatedFramesCountState = ref(0)
/** 忽略过期的 PATCH 响应，避免快速连点时状态被旧请求写乱 */
let organizeLibrarySaveSeq = 0
let autoLibraryWatchSaveSeq = 0
let autoActorProfileScrapeSaveSeq = 0
let autoDownloadUpdatesSaveSeq = 0
let launchAtLoginSaveSeq = 0
let lanEnabledSaveSeq = 0
let curatedFrameExportFormatSaveSeq = 0
let curatedFrameExportModeSaveSeq = 0
let defaultImportLibraryPathSaveSeq = 0
let backupDirectorySaveSeq = 0
let metadataMovieProviderSaveSeq = 0
let metadataMovieProviderChainSaveSeq = 0
let metadataMovieScrapeModeSaveSeq = 0
let proxySaveSeq = 0
let playerSettingsSaveSeq = 0
let backendLogSaveSeq = 0
let reloadMoviesDebounce: ReturnType<typeof setTimeout> | null = null

/** 并发「全量资料库列表」拉取合并为单次 in-flight，避免 ensure / reload / 多入口重复请求 */
let activeMoviesPromise: Promise<Movie[]> | null = null
let trashedMoviesPromise: Promise<Movie[]> | null = null
const pendingMovieDetailLoads = new Map<string, Promise<Movie | undefined>>()

/** 单次请求条数；多页拉取直至 total，避免首屏只看见前 200 条 */
const LIST_BATCH_SIZE = 500

type FetchPagedMoviesOptions = {
  onFirstPage?: (movies: Movie[]) => void
}

function mapLibraryPathsFromSettings(
  paths: { id: string; path: string; title: string; firstLibraryScanPending?: boolean }[],
): LibrarySetting[] {
  return paths.map((p) => ({
    id: p.id,
    path: p.path,
    title: p.title,
    firstLibraryScanPending: p.firstLibraryScanPending,
  }))
}

function mergeLibraryPathStorageStatuses(items: readonly LibraryPathStorageStatusDTO[]) {
  const next = new Map(libraryPathStorageStatusesState.value.map((item) => [item.libraryPathId, item]))
  for (const item of items) {
    next.set(item.libraryPathId, item)
  }
  libraryPathStorageStatusesState.value = [...next.values()]
}

async function fetchPagedMovies(
  mode?: "trash",
  options: FetchPagedMoviesOptions = {},
): Promise<Movie[]> {
  const first = await api.listMovies({ limit: LIST_BATCH_SIZE, offset: 0, mode })
  const all: Movie[] = first.items.map(mapMovieListItem)
  options.onFirstPage?.([...all])
  let offset = all.length
  const { total } = first

  while (offset < total) {
    const page = await api.listMovies({ limit: LIST_BATCH_SIZE, offset, mode })
    const batch = page.items.map(mapMovieListItem)
    if (batch.length === 0) {
      break
    }
    all.push(...batch)
    offset += batch.length
  }

  return all
}

function loadActiveMovies(): Promise<Movie[]> {
  if (activeMoviesPromise) {
    return activeMoviesPromise
  }
  activeMoviesPromise = fetchPagedMovies(undefined, {
    onFirstPage(movies) {
      moviesState.value = movies
      moviesLoadedState.value = true
      loadErrorState.value = null
    },
  }).finally(() => {
    activeMoviesPromise = null
  })
  return activeMoviesPromise
}

function loadTrashedMovies(): Promise<Movie[]> {
  if (trashedMoviesPromise) {
    return trashedMoviesPromise
  }
  trashedMoviesPromise = fetchPagedMovies("trash", {
    onFirstPage(movies) {
      trashedMoviesState.value = movies
      loadErrorState.value = null
    },
  }).finally(() => {
    trashedMoviesPromise = null
  })
  return trashedMoviesPromise
}

async function refreshCuratedFramesCountFromApi() {
  try {
    const { total } = await api.getCuratedFrameStats()
    curatedFramesCountState.value = total
  } catch {
    curatedFramesCountState.value = 0
  }
}

watch(curatedFramesRevision, () => {
  void refreshCuratedFramesCountFromApi()
})

function isTrashHashRoute(): boolean {
  if (typeof window === "undefined") {
    return false
  }
  const hash = window.location.hash || ""
  return hash === "#/trash" || hash.startsWith("#/trash?")
}

function formatLoadError(err: unknown, fallback: string): string {
  if (err instanceof HttpClientError) {
    return err.apiError?.message?.trim() || err.message || fallback
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

function setLoadError(err: unknown, fallbackKey: string) {
  loadErrorState.value = formatLoadError(err, i18n.global.t(fallbackKey))
}

async function reloadMoviesFromApiImmediate(options?: { includeTrash?: boolean }) {
  if (reloadMoviesDebounce) {
    clearTimeout(reloadMoviesDebounce)
    reloadMoviesDebounce = null
  }
  try {
    moviesState.value = await loadActiveMovies()
    if (options?.includeTrash) {
      trashedMoviesState.value = await loadTrashedMovies()
    }
    moviesLoadedState.value = true
    loadErrorState.value = null
  } catch (err) {
    setLoadError(err, "library.loadFailed")
    console.error("[web-library-service] failed to reload movies", err)
  }
}

async function ensureLoaded(options?: { includeTrash?: boolean }) {
  if (moviesLoadedState.value) return
  try {
    moviesState.value = await loadActiveMovies()
    if (options?.includeTrash) {
      trashedMoviesState.value = await loadTrashedMovies()
    }
    moviesLoadedState.value = true
    loadErrorState.value = null
  } catch (err) {
    setLoadError(err, "library.loadFailed")
    console.error("[web-library-service] failed to load movies", err)
  }
}

function mergeMovieIntoListState(movie: Movie) {
  const id = movie.id.trim()
  if (!id) return
  const inTrash = Boolean(movie.trashedAt?.trim())
  if (inTrash) {
    const idx = trashedMoviesState.value.findIndex((m) => m.id === id)
    if (idx >= 0) {
      trashedMoviesState.value = trashedMoviesState.value.map((m, i) =>
        i === idx ? { ...m, ...movie } : m,
      )
    } else {
      trashedMoviesState.value = [...trashedMoviesState.value, movie]
    }
    moviesState.value = moviesState.value.filter((m) => m.id !== id)
    return
  }
  const idx = moviesState.value.findIndex((m) => m.id === id)
  if (idx >= 0) {
    moviesState.value = moviesState.value.map((m, i) => (i === idx ? { ...m, ...movie } : m))
  } else {
    moviesState.value = [...moviesState.value, movie]
  }
  trashedMoviesState.value = trashedMoviesState.value.filter((m) => m.id !== id)
}

function resolveMetadataMovieScrapeMode(
  s: Pick<SettingsDTO, "metadataMovieScrapeMode" | "metadataMovieProvider" | "metadataMovieProviderChain">,
): MetadataMovieScrapeMode {
  const raw = s.metadataMovieScrapeMode?.trim().toLowerCase()
  if (raw === "auto" || raw === "specified" || raw === "chain") {
    return raw
  }
  const chain = Array.isArray(s.metadataMovieProviderChain) ? s.metadataMovieProviderChain : []
  if (chain.length > 0) return "chain"
  return (s.metadataMovieProvider ?? "").trim() === "" ? "auto" : "specified"
}

function applyMetadataMovieSettingsFromDTO(next: SettingsDTO) {
  metadataMovieProviderState.value = next.metadataMovieProvider?.trim() ?? ""
  metadataMovieProvidersState.value = Array.isArray(next.metadataMovieProviders)
    ? [...next.metadataMovieProviders]
    : []
  metadataMovieProviderChainState.value = Array.isArray(next.metadataMovieProviderChain)
    ? [...next.metadataMovieProviderChain]
    : []
  metadataMovieScrapeModeState.value = resolveMetadataMovieScrapeMode(next)
}

function applyPlayerSettingsFromDTO(next: SettingsDTO) {
  const player = next.player
  const streamPushEnabled = player?.streamPushEnabled !== false
  const forceStreamPush =
    streamPushEnabled && Boolean(player?.forceStreamPush)
  playerSettingsState.value = {
    hardwareDecode: player?.hardwareDecode !== false,
    hardwareEncoder: normalizeHardwareEncoderPreference(player?.hardwareEncoder),
    nativePlayerPreset: normalizeNativePlayerPreset(
      player?.nativePlayerPreset,
      player?.nativePlayerCommand,
    ),
    nativePlayerEnabled: Boolean(player?.nativePlayerEnabled),
    nativePlayerCommand:
      (player?.nativePlayerCommand ?? defaultNativePlayerCommand(player?.nativePlayerPreset)).trim() ||
      defaultNativePlayerCommand(player?.nativePlayerPreset),
    streamPushEnabled,
    forceStreamPush,
    ffmpegCommand: (player?.ffmpegCommand ?? "ffmpeg").trim() || "ffmpeg",
    preferNativePlayer: Boolean(player?.preferNativePlayer),
    seekForwardStepSec: Math.max(1, Number(player?.seekForwardStepSec ?? 10)),
    seekBackwardStepSec: Math.max(1, Number(player?.seekBackwardStepSec ?? 10)),
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

async function refreshLibraryPathsFromApi() {
  try {
    const settings = await api.getSettings()
    libraryPathsState.value = mapLibraryPathsFromSettings(settings.libraryPaths)
    defaultImportLibraryPathIdState.value = settings.defaultImportLibraryPathId?.trim() ?? ""
    backupDirectoryState.value = settings.backupDirectory?.trim() ?? ""
    organizeLibraryState.value = Boolean(settings.organizeLibrary)
    autoLibraryWatchState.value = settings.autoLibraryWatch !== false
    autoActorProfileScrapeState.value = Boolean(settings.autoActorProfileScrape)
    autoDownloadUpdatesState.value = Boolean(settings.autoDownloadUpdates)
    launchAtLoginState.value = Boolean(settings.launchAtLogin)
    launchAtLoginSupportedState.value = Boolean(settings.launchAtLoginSupported)
    lanEnabledState.value = Boolean(settings.lanEnabled)
    lanListeningState.value = Boolean(settings.lanListening)
    lanAccessUrlsState.value = Array.isArray(settings.lanAccessUrls) ? [...settings.lanAccessUrls] : []
    curatedFrameExportFormatState.value = settings.curatedFrameExportFormat ?? "jpg"
    curatedFrameExportModeState.value = settings.curatedFrameExportMode ?? "raw"
    applyPlayerSettingsFromDTO(settings)
    applyMetadataMovieSettingsFromDTO(settings)
    proxyState.value = settings.proxy ?? { enabled: false }
    aiProviderState.value = settings.aiProvider ?? { kind: "openai-compatible", baseUrl: "", model: "" }
    if (settings.aiGovernance) applyAIGovernance(settings.aiGovernance)
    backendLogState.value = settings.backendLog ?? {
      logDir: "",
      logLevel: "info",
    }
  } catch (err) {
    console.error("[web-library-service] failed to load settings", err)
  }
}

async function refreshLibraryPathStorageStatusesFromApi() {
  try {
    const dto = await api.listLibraryPathStorageStatus()
    libraryPathStorageStatusesState.value = dto.items ?? []
  } catch (err) {
    console.error("[web-library-service] failed to load library path storage status", err)
  }
}

function createWebLibraryService(): LibraryService {
  const impl: LibraryService = {
    supportsSourceFrame: true,
    movies: computed(() => moviesState.value),
    moviesLoaded: computed(() => moviesLoadedState.value),
    loadError: computed(() => loadErrorState.value),
    trashedMovies: computed(() => trashedMoviesState.value),
    libraryStats: computed(() => {
      const loc = i18n.global.locale.value as string
      return buildSettingsDashboardStats(moviesState.value, curatedFramesCountState.value, loc)
    }),
    libraryPaths: computed(() => libraryPathsState.value),
    libraryPathStorageStatuses: computed(() => libraryPathStorageStatusesState.value),
    savedViews: computed(() => savedViewsState.value),
    defaultImportLibraryPathId: computed(() => defaultImportLibraryPathIdState.value),
    backupDirectory: computed(() => backupDirectoryState.value),
    organizeLibrary: computed(() => organizeLibraryState.value),
    autoLibraryWatch: computed(() => autoLibraryWatchState.value),
    autoActorProfileScrape: computed(() => autoActorProfileScrapeState.value),
    autoDownloadUpdates: computed(() => autoDownloadUpdatesState.value),
    launchAtLogin: computed(() => launchAtLoginState.value),
    launchAtLoginSupported: computed(() => launchAtLoginSupportedState.value),
    lanEnabled: computed(() => lanEnabledState.value),
    lanListening: computed(() => lanListeningState.value),
    lanAccessUrls: computed(() => lanAccessUrlsState.value),
    curatedFrameExportFormat: computed(() => curatedFrameExportFormatState.value),
    curatedFrameExportMode: computed(() => curatedFrameExportModeState.value),
    metadataMovieProvider: computed(() => metadataMovieProviderState.value),
    metadataMovieProviders: computed(() => metadataMovieProvidersState.value),
    metadataMovieProviderChain: computed(() => metadataMovieProviderChainState.value),
    metadataMovieScrapeMode: computed(() => metadataMovieScrapeModeState.value),
    proxy: computed(() => proxyState.value),
    aiProvider: computed(() => aiProviderState.value),
    playerSettings: computed(() => playerSettingsState.value),
    backendLog: computed(() => backendLogState.value),

    async setProxy(config: ProxySettingsDTO) {
      const seq = ++proxySaveSeq
      proxyState.value = { ...config }
      try {
        const next = await api.patchSettings({ proxy: config })
        if (seq === proxySaveSeq) {
          proxyState.value = next.proxy ?? { enabled: false }
        }
      } catch (err) {
        if (seq === proxySaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async setAIProvider(patch: PatchAIProviderBody) {
      const seq = ++aiProviderSaveSeq
      const prev = aiProviderState.value
      aiProviderState.value = {
        ...prev,
        ...(patch.baseUrl !== undefined ? { baseUrl: patch.baseUrl } : {}),
        ...(patch.apiKey !== undefined ? { apiKey: patch.apiKey } : {}),
        ...(patch.model !== undefined ? { model: patch.model } : {}),
      }
      try {
        const next = await api.patchSettings({ aiProvider: patch })
        if (seq === aiProviderSaveSeq) {
          aiProviderState.value =
            next.aiProvider ?? { kind: "openai-compatible", baseUrl: "", model: "" }
        }
      } catch (err) {
        if (seq === aiProviderSaveSeq) {
          aiProviderState.value = prev
        }
        throw err
      }
    },

    async testAIProvider(provider?: AIProviderSettingsDTO): Promise<AIProviderTestResponse> {
      return api.testAIProvider(provider ? { provider } : {})
    },

    async patchPlayerSettings(patch: PatchPlayerSettingsBody) {
      const seq = ++playerSettingsSaveSeq
      const prev = playerSettingsState.value
      const merged: PlayerSettingsDTO = {
        hardwareDecode:
          patch.hardwareDecode !== undefined ? patch.hardwareDecode : prev.hardwareDecode,
        hardwareEncoder:
          patch.hardwareEncoder !== undefined ? patch.hardwareEncoder : prev.hardwareEncoder,
        nativePlayerPreset:
          patch.nativePlayerPreset !== undefined ? patch.nativePlayerPreset : prev.nativePlayerPreset,
        nativePlayerEnabled:
          patch.nativePlayerEnabled !== undefined
            ? patch.nativePlayerEnabled
            : prev.nativePlayerEnabled,
        nativePlayerCommand:
          patch.nativePlayerCommand !== undefined
            ? patch.nativePlayerCommand
            : prev.nativePlayerCommand,
        streamPushEnabled:
          patch.streamPushEnabled !== undefined
            ? patch.streamPushEnabled
            : prev.streamPushEnabled,
        forceStreamPush:
          patch.forceStreamPush !== undefined ? patch.forceStreamPush : prev.forceStreamPush,
        ffmpegCommand:
          patch.ffmpegCommand !== undefined ? patch.ffmpegCommand : prev.ffmpegCommand,
        preferNativePlayer:
          patch.preferNativePlayer !== undefined
            ? patch.preferNativePlayer
            : prev.preferNativePlayer,
        seekForwardStepSec:
          patch.seekForwardStepSec !== undefined
            ? patch.seekForwardStepSec
            : prev.seekForwardStepSec,
        seekBackwardStepSec:
          patch.seekBackwardStepSec !== undefined
            ? patch.seekBackwardStepSec
            : prev.seekBackwardStepSec,
      }
      // Never set forceStreamPush true from streamPushEnabled; only clear force when push is off.
      if (!merged.streamPushEnabled) {
        merged.forceStreamPush = false
      }
      playerSettingsState.value = {
        ...merged,
        hardwareEncoder: normalizeHardwareEncoderPreference(merged.hardwareEncoder),
        nativePlayerPreset: normalizeNativePlayerPreset(
          merged.nativePlayerPreset,
          merged.nativePlayerCommand,
        ),
        nativePlayerCommand:
          (merged.nativePlayerCommand ?? defaultNativePlayerCommand(merged.nativePlayerPreset)).trim() ||
          defaultNativePlayerCommand(merged.nativePlayerPreset),
        ffmpegCommand: (merged.ffmpegCommand ?? "ffmpeg").trim() || "ffmpeg",
        seekForwardStepSec: Math.max(1, Number(merged.seekForwardStepSec ?? 10)),
        seekBackwardStepSec: Math.max(1, Number(merged.seekBackwardStepSec ?? 10)),
      }
      try {
        const next = await api.patchSettings({ player: patch })
        if (seq === playerSettingsSaveSeq) {
          applyPlayerSettingsFromDTO(next)
        }
      } catch (err) {
        if (seq === playerSettingsSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async refreshSettings() {
      await Promise.all([
        refreshLibraryPathsFromApi(),
        refreshLibraryPathStorageStatusesFromApi(),
        refreshCuratedFramesCountFromApi(),
      ])
    },

    async createBackup(destinationPath: string): Promise<BackupManifestDTO> {
      return api.createBackup({ destinationPath: destinationPath.trim() })
    },

    async setBackupDirectory(directory: string): Promise<void> {
      const target = directory.trim()
      const seq = ++backupDirectorySaveSeq
      const next = await api.patchSettings({ backupDirectory: target })
      if (seq === backupDirectorySaveSeq) {
        backupDirectoryState.value = next.backupDirectory?.trim() ?? ""
      }
    },

    async verifyBackup(backupPath: string): Promise<BackupVerificationDTO> {
      return api.verifyBackup({ backupPath: backupPath.trim() })
    },

    async preflightBackupRestore(backupPath: string): Promise<BackupRestorePreflightDTO> {
      return api.preflightBackupRestore({ backupPath: backupPath.trim() })
    },

    async scanLibraryHealth(): Promise<LibraryHealthReportDTO> {
      return api.scanLibraryHealth()
    },

    async startLibraryHealthRepair(body: StartLibraryHealthRepairBody): Promise<LibraryHealthRepairDTO> {
      return api.startLibraryHealthRepair(body)
    },

    async getLibraryHealthRepair(repairId: string): Promise<LibraryHealthRepairDTO> {
      return api.getLibraryHealthRepair(repairId.trim())
    },

    async startLibraryHealthAction(body: StartLibraryHealthActionBody): Promise<TaskDTO> {
      return api.startLibraryHealthAction(body)
    },

    async checkLibraryPathStorageStatus(libraryPathIds?: string[]) {
      const cleaned = libraryPathIds?.map((id) => id.trim()).filter(Boolean) ?? []
      const dto = await api.checkLibraryPathStorageStatus(
        cleaned.length > 0 ? { libraryPathIds: cleaned } : undefined,
      )
      if (cleaned.length > 0) {
        mergeLibraryPathStorageStatuses(dto.items ?? [])
      } else {
        libraryPathStorageStatusesState.value = dto.items ?? []
      }
    },

    async rebindLibraryPathStorage(id: string) {
      const trimmed = id.trim()
      if (!trimmed) return
      const dto = await api.rebindLibraryPathStorage(trimmed)
      mergeLibraryPathStorageStatuses([dto])
    },

    async reloadMoviesFromApi() {
      if (reloadMoviesDebounce) {
        clearTimeout(reloadMoviesDebounce)
        reloadMoviesDebounce = null
      }
      reloadMoviesDebounce = setTimeout(() => {
        reloadMoviesDebounce = null
        void reloadMoviesFromApiImmediate({ includeTrash: isTrashHashRoute() })
      }, 450)
    },

    async listMoviesForExport() {
      try {
        const movies = await fetchPagedMovies()
        moviesState.value = movies
        moviesLoadedState.value = true
        loadErrorState.value = null
        return movies
      } catch (err) {
        setLoadError(err, "library.loadFailed")
        console.error("[web-library-service] failed to load movies for export", err)
        throw err
      }
    },

    async ensureTrashLoaded() {
      try {
        trashedMoviesState.value = await loadTrashedMovies()
        loadErrorState.value = null
      } catch (err) {
        setLoadError(err, "library.loadFailed")
        console.error("[web-library-service] failed to load trashed movies", err)
      }
    },

    async setOrganizeLibrary(value: boolean) {
      const seq = ++organizeLibrarySaveSeq
      organizeLibraryState.value = value
      try {
        const next = await api.patchSettings({ organizeLibrary: value })
        if (seq === organizeLibrarySaveSeq) {
          organizeLibraryState.value = next.organizeLibrary
        }
      } catch (err) {
        if (seq === organizeLibrarySaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // 拉取失败时保留当前 UI 值，由上层错误提示处理
          }
        }
        throw err
      }
    },

    async setAutoLibraryWatch(value: boolean) {
      const seq = ++autoLibraryWatchSaveSeq
      autoLibraryWatchState.value = value
      try {
        const next = await api.patchSettings({ autoLibraryWatch: value })
        if (seq === autoLibraryWatchSaveSeq) {
          autoLibraryWatchState.value = next.autoLibraryWatch !== false
        }
      } catch (err) {
        if (seq === autoLibraryWatchSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // 拉取失败时保留当前 UI 值，由上层错误提示处理
          }
        }
        throw err
      }
    },

    async setAutoActorProfileScrape(value: boolean) {
      const seq = ++autoActorProfileScrapeSaveSeq
      autoActorProfileScrapeState.value = value
      try {
        const next = await api.patchSettings({ autoActorProfileScrape: value })
        if (seq === autoActorProfileScrapeSaveSeq) {
          autoActorProfileScrapeState.value = Boolean(next.autoActorProfileScrape)
        }
      } catch (err) {
        if (seq === autoActorProfileScrapeSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async refreshSavedViews() {
      const dto = await api.listSavedViews()
      savedViewsState.value = [...dto.items]
    },

    async createSavedView(name: string, filters: SavedViewFiltersV1) {
      const created = await api.createSavedView({ name, filters })
      savedViewsState.value = [...savedViewsState.value, created].sort(
        (left, right) => left.sortOrder - right.sortOrder,
      )
      return created
    },

    async updateSavedView(id: string, patch: { name?: string; filters?: SavedViewFiltersV1 }) {
      const updated = await api.patchSavedView(id, patch)
      savedViewsState.value = savedViewsState.value.map((item) =>
        item.id === updated.id ? updated : item,
      )
      return updated
    },

    async deleteSavedView(id: string) {
      await api.deleteSavedView(id)
      savedViewsState.value = savedViewsState.value
        .filter((item) => item.id !== id)
        .map((item, sortOrder) => ({ ...item, sortOrder }))
    },

    async reorderSavedViews(ids: string[]) {
      const dto = await api.reorderSavedViews({ ids })
      savedViewsState.value = [...dto.items]
    },

    async setAutoDownloadUpdates(value: boolean) {
      const seq = ++autoDownloadUpdatesSaveSeq
      autoDownloadUpdatesState.value = value
      try {
        const next = await api.patchSettings({ autoDownloadUpdates: value })
        if (seq === autoDownloadUpdatesSaveSeq) {
          autoDownloadUpdatesState.value = Boolean(next.autoDownloadUpdates)
        }
      } catch (err) {
        if (seq === autoDownloadUpdatesSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async setLaunchAtLogin(value: boolean) {
      const seq = ++launchAtLoginSaveSeq
      launchAtLoginState.value = value
      try {
        const next = await api.patchSettings({ launchAtLogin: value })
        if (seq === launchAtLoginSaveSeq) {
          launchAtLoginState.value = Boolean(next.launchAtLogin)
          launchAtLoginSupportedState.value = Boolean(next.launchAtLoginSupported)
        }
      } catch (err) {
        if (seq === launchAtLoginSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    /** 保存局域网访问偏好；改绑 HTTP 监听需完全退出后重新打开。 */
    async setLANEnabled(value: boolean) {
      const seq = ++lanEnabledSaveSeq
      lanEnabledState.value = value
      try {
        const next = await api.patchSettings({ lanEnabled: value })
        if (seq === lanEnabledSaveSeq) {
          lanEnabledState.value = Boolean(next.lanEnabled)
          lanListeningState.value = Boolean(next.lanListening)
          lanAccessUrlsState.value = Array.isArray(next.lanAccessUrls) ? [...next.lanAccessUrls] : []
        }
      } catch (err) {
        if (seq === lanEnabledSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async setCuratedFrameExportFormat(format: CuratedFrameExportFormat) {
      const seq = ++curatedFrameExportFormatSaveSeq
      curatedFrameExportFormatState.value = format
      try {
        const next = await api.patchSettings({ curatedFrameExportFormat: format })
        if (seq === curatedFrameExportFormatSaveSeq) {
          curatedFrameExportFormatState.value = next.curatedFrameExportFormat ?? "jpg"
        }
      } catch (err) {
        if (seq === curatedFrameExportFormatSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },
    async setCuratedFrameExportMode(mode: CuratedFrameExportMode) {
      const seq = ++curatedFrameExportModeSaveSeq
      curatedFrameExportModeState.value = mode
      try {
        const next = await api.patchSettings({ curatedFrameExportMode: mode })
        if (seq === curatedFrameExportModeSaveSeq) {
          curatedFrameExportModeState.value = next.curatedFrameExportMode ?? "raw"
        }
      } catch (err) {
        if (seq === curatedFrameExportModeSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
          throw err
        }
      }
    },

    async setMetadataMovieProvider(name: string) {
      const trimmed = name.trim()
      const seq = ++metadataMovieProviderSaveSeq
      metadataMovieProviderState.value = trimmed
      try {
        const next = await api.patchSettings({ metadataMovieProvider: trimmed })
        if (seq === metadataMovieProviderSaveSeq) {
          applyMetadataMovieSettingsFromDTO(next)
        }
      } catch (err) {
        if (seq === metadataMovieProviderSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async setMetadataMovieProviderChain(chain: string[]) {
      const filtered = chain.map(p => p.trim()).filter(Boolean)
      const seq = ++metadataMovieProviderChainSaveSeq
      metadataMovieProviderChainState.value = filtered
      try {
        const next = await api.patchSettings({ metadataMovieProviderChain: filtered })
        if (seq === metadataMovieProviderChainSaveSeq) {
          applyMetadataMovieSettingsFromDTO(next)
        }
      } catch (err) {
        if (seq === metadataMovieProviderChainSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async setMetadataMovieScrapeMode(mode: MetadataMovieScrapeMode) {
      const seq = ++metadataMovieScrapeModeSaveSeq
      metadataMovieScrapeModeState.value = mode
      try {
        const next = await api.patchSettings({ metadataMovieScrapeMode: mode })
        if (seq === metadataMovieScrapeModeSaveSeq) {
          applyMetadataMovieSettingsFromDTO(next)
        }
      } catch (err) {
        if (seq === metadataMovieScrapeModeSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async patchBackendLog(patch: PatchBackendLogBody) {
      const seq = ++backendLogSaveSeq
      const prev = backendLogState.value
      const merged: BackendLogSettingsDTO = {
        logDir: patch.logDir !== undefined ? patch.logDir : prev.logDir,
        logFilePrefix:
          patch.logFilePrefix !== undefined ? patch.logFilePrefix : prev.logFilePrefix,
        logMaxAgeDays:
          patch.logMaxAgeDays !== undefined ? patch.logMaxAgeDays : prev.logMaxAgeDays,
        logLevel: patch.logLevel !== undefined ? patch.logLevel : prev.logLevel,
      }
      backendLogState.value = { ...merged }
      try {
        // 不传 logFilePrefix：由后端/logging 默认前缀（curated），避免设置页覆盖手写 cfg
        const next = await api.patchSettings({
          backendLog: {
            logDir: merged.logDir ?? "",
            logMaxAgeDays: merged.logMaxAgeDays ?? 0,
            logLevel: (merged.logLevel ?? "info").trim() || "info",
          },
        })
        if (seq === backendLogSaveSeq) {
          backendLogState.value = next.backendLog ?? merged
        }
      } catch (err) {
        if (seq === backendLogSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async health() {
      return await api.health()
    },

    async listConnectedClients() {
      return await api.listConnectedClients()
    },

    async pingProxyJavbus(body) {
      return await api.pingProxyJavbus(body)
    },

    async pingProxyGoogle(body) {
      return await api.pingProxyGoogle(body)
    },

    async pingProvider(name: string) {
      return await api.pingProvider(name)
    },

    async pingAllProviders() {
      return await api.pingAllProviders()
    },

    async getHomepageDailyRecommendations(): Promise<HomepageDailyRecommendationsDTO> {
      return await api.getHomepageDailyRecommendations()
    },

    async refreshHomepageDailyRecommendations(
      body?: RefreshHomepageDailyRecommendationsBody,
    ): Promise<HomepageDailyRecommendationsDTO> {
      return await api.refreshHomepageDailyRecommendations(body)
    },

    async listHomepageRecommendationFeedback(): Promise<RecommendationFeedbackListDTO> {
      return await api.listHomepageRecommendationFeedback()
    },

    async createHomepageRecommendationFeedback(
      body: CreateRecommendationFeedbackBody,
    ): Promise<RecommendationFeedbackDTO> {
      return await api.createHomepageRecommendationFeedback(body)
    },

    async deleteHomepageRecommendationFeedback(id: string): Promise<void> {
      await api.deleteHomepageRecommendationFeedback(id)
    },

    async addLibraryPath(path: string, title?: string): Promise<TaskDTO | null> {
      const trimmed = path.trim()
      if (!trimmed) return null
      const res = await api.addLibraryPath({ path: trimmed, title: title?.trim() || undefined })
      await refreshLibraryPathsFromApi()
      await impl.checkLibraryPathStorageStatus([res.id])
      return res.scanTask ?? null
    },

    async updateLibraryPathTitle(id: string, title: string) {
      await api.updateLibraryPathTitle(id, { title: title.trim() })
      await refreshLibraryPathsFromApi()
    },

    async removeLibraryPath(id: string) {
      await api.deleteLibraryPath(id)
      libraryPathStorageStatusesState.value = libraryPathStorageStatusesState.value.filter(
        (status) => status.libraryPathId !== id,
      )
      await refreshLibraryPathsFromApi()
      await reloadMoviesFromApiImmediate({ includeTrash: isTrashHashRoute() })
    },

    async revealLibraryPathInFileManager(id: string): Promise<void> {
      await api.revealLibraryPathInFileManager(id)
    },

    async setDefaultImportLibraryPathId(id: string) {
      const nextId = id.trim()
      const seq = ++defaultImportLibraryPathSaveSeq
      defaultImportLibraryPathIdState.value = nextId
      try {
        const next = await api.patchSettings({ defaultImportLibraryPathId: nextId })
        if (seq === defaultImportLibraryPathSaveSeq) {
          defaultImportLibraryPathIdState.value = next.defaultImportLibraryPathId?.trim() ?? ""
        }
      } catch (err) {
        if (seq === defaultImportLibraryPathSaveSeq) {
          try {
            await refreshLibraryPathsFromApi()
          } catch {
            // ignore
          }
        }
        throw err
      }
    },

    async checkImportMovieCodes(names) {
      return await api.checkImportMovieCodes({ names })
    },

    async importMovies(files, options) {
      const selected = files.filter((file) => file.size >= 0)
      if (selected.length === 0) {
        return null
      }
      const resumeUploadId = options?.resumeUploadId?.trim()
      if (resumeUploadId) {
        // 成功（含 commit 完成）才移除账本；失败保留条目供下次继续
        const task = await api.importMovies(selected, options)
        removeMovieImportUploadLedgerEntry(resumeUploadId)
        return task
      }
      return await api.importMovies(selected, {
        ...options,
        onUploadSessionCreated: (upload) => {
          addMovieImportUploadLedgerEntry({
            uploadId: upload.uploadId,
            targetLibraryPathId: defaultImportLibraryPathIdState.value,
            chunkSize: upload.chunkSize,
            files: movieImportUploadFileManifests(selected),
            createdAt: new Date().toISOString(),
            lastActiveAt: new Date().toISOString(),
          })
        },
      })
    },

    async listResumableMovieImports(): Promise<ResumableMovieImportSession[]> {
      const sessions: ResumableMovieImportSession[] = []
      for (const entry of loadMovieImportUploadLedger()) {
        try {
          const dto = await api.getMovieImportUpload(entry.uploadId)
          const expiresAtMs = dto.expiresAt ? Date.parse(dto.expiresAt) : Number.NaN
          const expired = Number.isFinite(expiresAtMs) && expiresAtMs <= Date.now()
          if (dto.state !== "uploading" || expired) {
            removeMovieImportUploadLedgerEntry(entry.uploadId)
            continue
          }
          touchMovieImportUploadLedgerEntry(entry.uploadId)
          sessions.push({
            uploadId: entry.uploadId,
            files: entry.files,
            totalBytes: dto.totalBytes,
            bytesReceived: dto.bytesReceived,
            expiresAt: dto.expiresAt,
          })
        } catch (error) {
          // 会话已不存在（如被清理）时移除本地账本；网络错误保留条目下次再核对
          if (error instanceof HttpClientError && error.status === 404) {
            removeMovieImportUploadLedgerEntry(entry.uploadId)
          }
        }
      }
      return sessions
    },

    async abandonMovieImportUpload(uploadId: string): Promise<void> {
      const trimmed = uploadId.trim()
      if (!trimmed) {
        return
      }
      try {
        await api.deleteMovieImportUpload(trimmed)
      } finally {
        removeMovieImportUploadLedgerEntry(trimmed)
      }
    },

    async scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null> {
      return await api.startScan(paths?.length ? { paths } : undefined)
    },

    async getTaskStatus(taskId: string): Promise<TaskDTO> {
      return await api.getTaskStatus(taskId)
    },

    async createMovieClip(movieId, body) {
      return await api.createMovieClip(movieId, { ...body, format: body.format ?? "gif" })
    },
    async cancelMovieClip(taskId) { await api.cancelMovieClip(taskId) },
    async extractMovieFrame(movieId, positionSec) { return api.extractMovieFrame(movieId, positionSec) },

    async refreshMovieMetadata(movieId: string): Promise<TaskDTO | null> {
      return await api.refreshMovieMetadata(movieId)
    },

    async revealMovieInFileManager(movieId: string): Promise<void> {
      await api.revealMovieInFileManager(movieId)
    },

    async refreshMetadataForLibraryPaths(paths: string[]): Promise<MetadataRefreshQueuedDTO> {
      const cleaned = paths.map((p) => p.trim()).filter(Boolean)
      return await api.startMetadataRefreshByPaths({ paths: cleaned })
    },

    async getMoviePlayback(movieId: string, options?: { startPositionSec?: number; signal?: AbortSignal }): Promise<PlaybackDescriptorDTO | null> {
      const id = movieId.trim()
      if (!id) {
        return null
      }
      const prefetched = takeFreshMoviePlaybackPrefetch(id, options?.startPositionSec, options?.signal)
      if (prefetched) {
        return await prefetched
      }
      const dto = await api.getMoviePlayback(id, { clientVideoCodecs: clientVideoCodecsParam(), ...options })
      if (options?.signal?.aborted) {
        if (dto.sessionId) void api.deletePlaybackSession(dto.sessionId).catch(() => {})
        throw new DOMException("Playback cancelled", "AbortError")
      }
      if (!dto.url) {
        dto.url = moviePlaybackAbsoluteUrl(id)
      }
      return dto
    },

    prefetchMoviePlayback(movieId: string, startPositionSec?: number) {
      const id = movieId.trim()
      if (!id) return
      return prefetchMoviePlaybackRequest(id, startPositionSec)
    },

    async createPlaybackSession(
      movieId: string,
      mode: PlaybackDescriptorDTO["mode"],
      startPositionSec?: number,
      signal?: AbortSignal,
    ): Promise<PlaybackDescriptorDTO | null> {
      const id = movieId.trim()
      if (!id) {
        return null
      }
      const dto = await api.createPlaybackSession(id, {
        mode,
        startPositionSec,
      }, signal)
      if (signal?.aborted) {
        if (dto.sessionId) void api.deletePlaybackSession(dto.sessionId).catch(() => {})
        throw new DOMException("Playback cancelled", "AbortError")
      }
      if (!dto.url) {
        dto.url = moviePlaybackAbsoluteUrl(id)
      }
      return dto
    },

    async getPlaybackSession(sessionId: string): Promise<PlaybackSessionStatusDTO | null> {
      const id = sessionId.trim()
      if (!id) {
        return null
      }
      try {
        return await api.getPlaybackSession(id)
      } catch {
        return null
      }
    },

    async launchNativePlayback(movieId: string, startPositionSec?: number): Promise<NativePlaybackLaunchDTO | null> {
      const id = movieId.trim()
      if (!id) {
        return null
      }
      return await api.launchNativePlayback(id, startPositionSec)
    },

    async deletePlaybackSession(sessionId: string) {
      const id = sessionId.trim()
      if (!id) {
        return
      }
      await api.deletePlaybackSession(id)
    },

    async ensureMovieCached(movieId: string) {
      const trimmed = movieId.trim()
      if (!trimmed) return
      if (
        moviesState.value.some((m) => m.id === trimmed) ||
        trashedMoviesState.value.some((m) => m.id === trimmed)
      ) {
        return
      }
      // Deep links (sidebar resume chip, external URLs) previously waited for
      // the full paginated library load before the player page could mount.
      // Resolve the single detail request first; it merges the movie into the
      // list state, and the full library load stays a rare fallback.
      const movie = await loadMovieDetail(trimmed)
      if (movie) return
      await ensureLoaded()
    },

    mergeMovieIntoCache(movie: Movie) {
      mergeMovieIntoListState(movie)
    },

    getMovieById(movieId) {
      const id = movieId?.trim()
      if (!id) return undefined
      return (
        moviesState.value.find((m) => m.id === id) ??
        trashedMoviesState.value.find((m) => m.id === id)
      )
    },

    async loadMovieDetail(movieId: string) {
      return await loadMovieDetail(movieId)
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
            movie.actors.some((actor) => actors.has(actor.trim().toLocaleLowerCase())),
        )
        .slice(0, Math.max(0, limit))
    },

    async patchMovie(movieId: string, body: PatchMovieBody) {
      await ensureLoaded()
      const id = movieId.trim()
      if (!id) return undefined

      let snapshotActive = moviesState.value.map((m) => ({ ...m }))
      let snapshotTrashed = trashedMoviesState.value.map((m) => ({ ...m }))
      let existing =
        moviesState.value.find((m) => m.id === id) ??
        trashedMoviesState.value.find((m) => m.id === id)
      if (!existing) {
        await loadMovieDetail(id)
        snapshotActive = moviesState.value.map((m) => ({ ...m }))
        snapshotTrashed = trashedMoviesState.value.map((m) => ({ ...m }))
        existing =
          moviesState.value.find((m) => m.id === id) ??
          trashedMoviesState.value.find((m) => m.id === id)
      }

      try {
        const dto = await api.patchMovie(id, body)
        const mapped = mapMovieDetail(dto)
        mergeMovieIntoListState(mapped)
        return impl.getMovieById(id)
      } catch (err) {
        moviesState.value = snapshotActive
        trashedMoviesState.value = snapshotTrashed
        throw err
      }
    },

    async toggleFavorite(movieId, nextValue) {
      const id = movieId.trim()
      const movie = moviesState.value.find((m) => m.id === id)
      const target = typeof nextValue === "boolean" ? nextValue : !(movie?.isFavorite ?? false)
      return await impl.patchMovie(id, { isFavorite: target })
    },

    async deleteMovie(movieId: string) {
      const id = movieId.trim()
      await api.deleteMovie(id)
      moviesState.value = moviesState.value.filter((m) => m.id !== id)
      try {
        trashedMoviesState.value = await loadTrashedMovies()
      } catch {
        // 忽略回收站刷新失败，主列表已一致
      }
    },

    async restoreMovie(movieId: string) {
      const id = movieId.trim()
      await api.restoreMovie(id)
      await reloadMoviesFromApiImmediate({ includeTrash: true })
    },

    async deleteMoviePermanently(movieId: string) {
      const id = movieId.trim()
      await api.deleteMovie(id, { permanent: true })
      trashedMoviesState.value = trashedMoviesState.value.filter((m) => m.id !== id)
    },

    async listActors(params?: ListActorsParams) {
      return await api.listActors(params)
    },

    async getActorProfile(name: string) {
      return await api.getActorProfile(name)
    },

    async scrapeActorProfile(name: string) {
      return await api.scrapeActorProfile(name)
    },

    async patchActorUserTags(name: string, userTags: string[]) {
      return await api.patchActorUserTags(name.trim(), userTags)
    },

    async patchActorExternalLinks(name: string, externalLinks: string[]) {
      return await api.patchActorExternalLinks(name.trim(), externalLinks)
    },

    async previewActorMerge(body: ActorMergePreviewRequest): Promise<ActorMergePreviewDTO> {
      return await api.previewActorMerge(body)
    },

    async applyActorMerge(body: ApplyActorMergeRequest): Promise<ActorMergeAuditDTO> {
      const audit = await api.applyActorMerge(body)
      await reloadMoviesFromApiImmediate()
      return audit
    },

    async listActorMergeAudits(params?: { limit?: number; offset?: number }): Promise<ActorMergeAuditListDTO> {
      return await api.listActorMergeAudits(params)
    },

    async getPersonalInsightsOverview(params: {
      range: PersonalInsightsRange
      timezone: string
    }): Promise<PersonalInsightsOverviewDTO> {
      return await api.getPersonalInsightsOverview(params)
    },

    async getPersonalInsightsBreakdown(params: {
      range: PersonalInsightsRange
      timezone: string
      dimension: PersonalInsightsDimension
      limit?: number
    }): Promise<PersonalInsightsBreakdownDTO> {
      return await api.getPersonalInsightsBreakdown(params)
    },

    async getMovieComment(movieId: string) {
      return await api.getMovieComment(movieId.trim())
    },

    async putMovieComment(movieId: string, body) {
      return await api.putMovieComment(movieId.trim(), body)
    },

    async exportCuratedFrames(body) {
      return await api.postCuratedFramesExport(body)
    },
  }

  return impl
}

type MoviePlaybackPrefetch = {
  promise: Promise<PlaybackDescriptorDTO>
  createdAt: number
  controller: AbortController
  timer?: ReturnType<typeof setTimeout>
}

const moviePlaybackPrefetches = new Map<string, MoviePlaybackPrefetch>()
const MOVIE_PLAYBACK_PREFETCH_TTL_MS = 10_000

/**
 * Browser-reported decodable mp4-family codecs for the descriptor request, so
 * the backend can route HEVC/AV1 sources this browser cannot decode to HLS
 * instead of failing with a decode error. Null omits the parameter entirely.
 */
function clientVideoCodecsParam(): string | null {
  return clientVideoCodecsQueryParam(resolvePlaybackCapabilities())
}

/**
 * Start a playback descriptor request ahead of the player page mounting. For
 * HLS-eligible movies the GET also boots the server-side session, so the route
 * transition overlaps ffmpeg startup instead of serializing behind it. Entries
 * are consume-once and short-lived; failed requests are dropped immediately.
 */
function playbackPrefetchKey(movieId: string, startPositionSec?: number): string {
  return JSON.stringify([movieId, startPositionSec ?? null, clientVideoCodecsParam()])
}

function disposePlaybackPrefetch(key: string, entry: MoviePlaybackPrefetch): void {
  if (moviePlaybackPrefetches.get(key) !== entry) return
  moviePlaybackPrefetches.delete(key)
  clearTimeout(entry.timer)
  entry.controller.abort()
  void entry.promise.then((dto) => {
    if (dto.sessionId) return api.deletePlaybackSession(dto.sessionId)
  }).catch(() => {})
}

function prefetchMoviePlaybackRequest(movieId: string, startPositionSec?: number): () => void {
  const key = playbackPrefetchKey(movieId, startPositionSec)
  const existing = moviePlaybackPrefetches.get(key)
  if (existing) return () => disposePlaybackPrefetch(key, existing)
  const controller = new AbortController()
  let promise: Promise<PlaybackDescriptorDTO>
  try {
    promise = api
      .getMoviePlayback(movieId, { clientVideoCodecs: clientVideoCodecsParam(), startPositionSec, signal: controller.signal })
      .then((dto) => {
        if (!dto.url) {
          dto.url = moviePlaybackAbsoluteUrl(movieId)
        }
        return dto
      })
  } catch {
    return () => {}
  }
  const entry: MoviePlaybackPrefetch = { promise, createdAt: Number.POSITIVE_INFINITY, controller }
  moviePlaybackPrefetches.set(key, entry)
  void promise.then(() => {
    if (moviePlaybackPrefetches.get(key) !== entry) return
    entry.createdAt = Date.now()
    entry.timer = setTimeout(() => disposePlaybackPrefetch(key, entry), MOVIE_PLAYBACK_PREFETCH_TTL_MS)
  }).catch(() => {})
  void promise.catch(() => {
    if (moviePlaybackPrefetches.get(key) === entry) {
      moviePlaybackPrefetches.delete(key)
    }
  })
  return () => disposePlaybackPrefetch(key, entry)
}

function takeFreshMoviePlaybackPrefetch(movieId: string, startPositionSec?: number, signal?: AbortSignal): Promise<PlaybackDescriptorDTO> | null {
  const key = playbackPrefetchKey(movieId, startPositionSec)
  const entry = moviePlaybackPrefetches.get(key)
  if (!entry) return null
  if (Date.now() - entry.createdAt >= MOVIE_PLAYBACK_PREFETCH_TTL_MS) {
    disposePlaybackPrefetch(key, entry)
    return null
  }
  moviePlaybackPrefetches.delete(key)
  clearTimeout(entry.timer)
  const abort = () => entry.controller.abort()
  if (signal?.aborted) abort()
  signal?.addEventListener("abort", abort, { once: true })
  return entry.promise.then((dto) => {
    if (entry.controller.signal.aborted) {
      if (dto.sessionId) void api.deletePlaybackSession(dto.sessionId).catch(() => {})
      throw new DOMException("Playback cancelled", "AbortError")
    }
    return dto
  }).finally(() => signal?.removeEventListener("abort", abort))
}

export async function loadMovieDetail(movieId: string): Promise<Movie | undefined> {
  const id = movieId.trim()
  if (!id) return undefined
  const existing = pendingMovieDetailLoads.get(id)
  if (existing) {
    return existing
  }
  const promise = (async (): Promise<Movie | undefined> => {
    try {
      const dto = await api.getMovie(id)
      const mapped = mapMovieDetail(dto)
      mergeMovieIntoListState(mapped)
      loadErrorState.value = null
      return mapped
    } catch (err) {
      setLoadError(err, "detail.loadError")
      const extra = err instanceof HttpClientError ? ` status=${err.status}` : ""
      console.error(`[web-library-service] loadMovieDetail failed${extra}`, id, err)
      return undefined
    } finally {
      pendingMovieDetailLoads.delete(id)
    }
  })()
  pendingMovieDetailLoads.set(id, promise)
  return promise
}

/**
 * Explicitly starts the selected Web adapter. Importing this module must stay
 * network-side-effect free so the Mock adapter can be bundled without probing
 * a backend that is not part of the active runtime.
 */
export function startWebLibraryService(): Promise<void> {
  return ensureLoaded({ includeTrash: isTrashHashRoute() })
}

export const webLibraryService = createWebLibraryService()
