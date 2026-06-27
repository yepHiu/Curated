import { computed, nextTick } from "vue"
import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import SettingsPage from "./SettingsPage.vue"

type MockFunction = ReturnType<typeof vi.fn>
type TestLibraryService = Record<string, unknown> & {
  listMoviesForExport: MockFunction
  refreshSettings: MockFunction
}

const mockState = vi.hoisted<{
  libraryService: TestLibraryService | null
  pushAppToast: MockFunction
  triggerDownloadBlob: MockFunction
}>(() => ({
  libraryService: null,
  pushAppToast: vi.fn(),
  triggerDownloadBlob: vi.fn(),
}))

vi.mock("vue-i18n", async () => {
  const vue = await vi.importActual<typeof import("vue")>("vue")
  return {
    useI18n: () => ({
      locale: vue.ref("zh-CN"),
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: { section: "library" } }),
  useRouter: () => ({ replace: vi.fn().mockResolvedValue(undefined) }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => mockState.libraryService,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: mockState.pushAppToast,
}))

vi.mock("@/lib/curated-frames/export-file", () => ({
  triggerDownloadBlob: mockState.triggerDownloadBlob,
}))

vi.mock("@/composables/use-theme", async () => {
  const vue = await vi.importActual<typeof import("vue")>("vue")
  return {
    useTheme: () => ({
      themePreference: vue.ref("system"),
      setThemePreference: vi.fn(),
    }),
  }
})

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => ({ start: vi.fn() }),
}))

vi.mock("@/composables/use-connected-clients", async () => {
  const vue = await vi.importActual<typeof import("vue")>("vue")
  return {
    useConnectedClients: () => ({
      clients: vue.computed(() => []),
      total: vue.computed(() => 0),
      localCount: vue.computed(() => 0),
      remoteCount: vue.computed(() => 0),
      loading: vue.computed(() => false),
      error: vue.computed(() => ""),
      sampledAt: vue.computed(() => ""),
      refresh: vi.fn(),
    }),
  }
})

vi.mock("@/composables/use-settings-scroll-preserve", () => ({
  SETTINGS_SCROLL_EL_KEY: Symbol("settingsScrollEl"),
  SETTINGS_SCROLL_ROOT_ID: "settings-scroll-root",
  useSettingsScrollPreserve: () => ({
    withPreservedScroll: async <T>(fn: () => Promise<T>) => await fn(),
    withSyncPreservedScroll: (fn: () => void) => fn(),
  }),
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: { name: "Tabs", template: "<div><slot /></div>" },
  TabsContent: { name: "TabsContent", template: "<div><slot /></div>" },
  TabsList: { name: "TabsList", template: "<div><slot /></div>" },
  TabsTrigger: { name: "TabsTrigger", template: "<button><slot /></button>" },
}))

vi.mock("@/lib/pick-directory", () => ({
  pickLibraryDirectory: vi.fn(),
}))

vi.mock("@/lib/curated-frames/db", () => ({
  getStoredDirectoryHandle: vi.fn().mockResolvedValue(null),
  setStoredDirectoryHandle: vi.fn().mockResolvedValue(undefined),
  supportsFileSystemAccess: () => false,
}))

vi.mock("@/lib/curated-frames/settings-storage", () => ({
  getCuratedCaptureKeyCode: () => "KeyC",
  getCuratedFrameSaveMode: () => "app",
  setCuratedFrameSaveMode: vi.fn(),
}))

vi.mock("@/lib/player-shortcuts", () => ({
  formatCuratedCaptureKeyLabel: () => "C",
}))

vi.mock("@/lib/playback-watch-time-storage", async () => {
  const vue = await vi.importActual<typeof import("vue")>("vue")
  const heatmap = await vi.importActual<typeof import("@/lib/watch-time-heatmap")>(
    "@/lib/watch-time-heatmap",
  )
  return {
    watchTimeRevision: vue.ref(0),
    listDailyWatchTime: vi.fn().mockResolvedValue(heatmap.createEmptyDailyWatchTimeSummary()),
  }
})

vi.mock("@/components/jav-library/settings/SettingsLibraryPathsSection.vue", () => ({
  default: {
    name: "SettingsLibraryPathsSection",
    props: [
      "movieCsvExportBusy",
      "movieCsvExportError",
    ],
    emits: ["exportMoviesCsv"],
    template:
      "<section data-settings-library-paths :data-busy=\"String(movieCsvExportBusy)\" :data-error=\"movieCsvExportError\"><button data-export-trigger @click=\"$emit('exportMoviesCsv')\">export</button></section>",
  },
}))

vi.mock("@/components/jav-library/settings/SettingsAboutSection.vue", () => ({
  default: { name: "SettingsAboutSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsCuratedSection.vue", () => ({
  default: { name: "SettingsCuratedSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsComicLibrarySection.vue", () => ({
  default: { name: "SettingsComicLibrarySection", template: "<section data-comic-settings />" },
}))
vi.mock("@/components/jav-library/settings/SettingsGeneralSection.vue", () => ({
  default: { name: "SettingsGeneralSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsMaintenanceSection.vue", () => ({
  default: { name: "SettingsMaintenanceSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsMetadataSection.vue", () => ({
  default: { name: "SettingsMetadataSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsNetworkSection.vue", () => ({
  default: { name: "SettingsNetworkSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsOrganizeSection.vue", () => ({
  default: { name: "SettingsOrganizeSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsOverviewSection.vue", () => ({
  default: { name: "SettingsOverviewSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsPlaybackSection.vue", () => ({
  default: { name: "SettingsPlaybackSection", template: "<section />" },
}))
vi.mock("@/components/jav-library/settings/SettingsSecuritySection.vue", () => ({
  default: { name: "SettingsSecuritySection", template: "<section />" },
}))

function movie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-1",
    title: "Export title",
    code: "ABC-123",
    studio: "Studio One",
    actors: ["Alice"],
    tags: [],
    userTags: [],
    runtimeMinutes: 90,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: "2026-01-01T00:00:00.000Z",
    location: "D:/Movies/ABC-123.mp4",
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    ...overrides,
  }
}

function createLibraryServiceMock(overrides: Partial<TestLibraryService> = {}): TestLibraryService {
  return {
    movies: computed(() => []),
    moviesLoaded: computed(() => true),
    loadError: computed(() => null),
    trashedMovies: computed(() => []),
    libraryStats: computed(() => []),
    libraryPaths: computed(() => []),
    libraryPathStorageStatuses: computed(() => []),
    defaultImportLibraryPathId: computed(() => ""),
    organizeLibrary: computed(() => false),
    autoLibraryWatch: computed(() => false),
    autoActorProfileScrape: computed(() => false),
    autoDownloadUpdates: computed(() => false),
    launchAtLogin: computed(() => false),
    launchAtLoginSupported: computed(() => false),
    curatedFrameExportFormat: computed(() => "jpg"),
    metadataMovieProvider: computed(() => ""),
    metadataMovieProviders: computed(() => []),
    metadataMovieProviderChain: computed(() => []),
    metadataMovieScrapeMode: computed(() => "auto"),
    proxy: computed(() => ({ enabled: false })),
    refreshSettings: vi.fn().mockResolvedValue(undefined),
    listMoviesForExport: vi.fn().mockResolvedValue([]),
    checkLibraryPathStorageStatus: vi.fn().mockResolvedValue(undefined),
    rebindLibraryPathStorage: vi.fn().mockResolvedValue(undefined),
    updateLibraryPathTitle: vi.fn().mockResolvedValue(undefined),
    addLibraryPath: vi.fn().mockResolvedValue(null),
    removeLibraryPath: vi.fn().mockResolvedValue(undefined),
    revealLibraryPathInFileManager: vi.fn().mockResolvedValue(undefined),
    setDefaultImportLibraryPathId: vi.fn().mockResolvedValue(undefined),
    scanLibraryPaths: vi.fn().mockResolvedValue(null),
    refreshMetadataForLibraryPaths: vi.fn().mockResolvedValue({
      queued: 0,
      skipped: 0,
      invalidPaths: [],
    }),
    setOrganizeLibrary: vi.fn().mockResolvedValue(undefined),
    setAutoLibraryWatch: vi.fn().mockResolvedValue(undefined),
    setAutoActorProfileScrape: vi.fn().mockResolvedValue(undefined),
    setAutoDownloadUpdates: vi.fn().mockResolvedValue(undefined),
    setLaunchAtLogin: vi.fn().mockResolvedValue(undefined),
    setCuratedFrameExportFormat: vi.fn().mockResolvedValue(undefined),
    setMetadataMovieProvider: vi.fn().mockResolvedValue(undefined),
    setMetadataMovieProviderChain: vi.fn().mockResolvedValue(undefined),
    setMetadataMovieScrapeMode: vi.fn().mockResolvedValue(undefined),
    pingProxyJavbus: vi.fn().mockResolvedValue({ ok: true }),
    pingProxyGoogle: vi.fn().mockResolvedValue({ ok: true }),
    pingProvider: vi.fn().mockResolvedValue({ ok: true }),
    pingAllProviders: vi.fn().mockResolvedValue({ providers: [] }),
    health: vi.fn().mockResolvedValue(null),
    ...overrides,
  } as TestLibraryService
}

async function mountSettingsPage() {
  const wrapper = mount(SettingsPage)
  await flushPromises()
  await nextTick()
  return wrapper
}

describe("SettingsPage movie CSV export", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockState.libraryService = createLibraryServiceMock()
  })

  it("shows the comic library settings navigation item", async () => {
    const wrapper = await mountSettingsPage()

    expect(wrapper.text()).toContain("settings.navComics")
  })

  it("downloads a CSV and shows a success toast when the storage section requests export", async () => {
    const movies = [movie()]
    let resolveExport: (movies: readonly Movie[]) => void = () => {}
    mockState.libraryService = createLibraryServiceMock({
      listMoviesForExport: vi.fn().mockImplementation(
        () =>
          new Promise<readonly Movie[]>((resolve) => {
            resolveExport = resolve
          }),
      ),
    })

    const wrapper = await mountSettingsPage()
    await wrapper.get("[data-export-trigger]").trigger("click")
    await nextTick()

    expect(wrapper.get("[data-settings-library-paths]").attributes("data-busy")).toBe("true")

    resolveExport(movies)
    await flushPromises()
    await nextTick()

    expect(mockState.libraryService.listMoviesForExport).toHaveBeenCalledTimes(1)
    expect(mockState.triggerDownloadBlob).toHaveBeenCalledTimes(1)
    expect(mockState.triggerDownloadBlob.mock.calls[0][1]).toMatch(
      /^curated-movies-\d{8}-\d{6}\.csv$/,
    )
    expect(mockState.pushAppToast).toHaveBeenCalledWith(
      'settings.movieCsvExportSuccess:{"count":1}',
      expect.objectContaining({ variant: "success" }),
    )
    expect(wrapper.get("[data-settings-library-paths]").attributes("data-busy")).toBe("false")
    expect(wrapper.get("[data-settings-library-paths]").attributes("data-error")).toBe("")
  })

  it("passes a localized export error to the storage section when export fails", async () => {
    mockState.libraryService = createLibraryServiceMock({
      listMoviesForExport: vi.fn().mockRejectedValue(new Error("network failed")),
    })

    const wrapper = await mountSettingsPage()
    await wrapper.get("[data-export-trigger]").trigger("click")
    await flushPromises()
    await nextTick()

    expect(mockState.triggerDownloadBlob).not.toHaveBeenCalled()
    expect(wrapper.get("[data-settings-library-paths]").attributes("data-error")).toBe(
      "settings.movieCsvExportFailed",
    )
    expect(wrapper.get("[data-settings-library-paths]").attributes("data-busy")).toBe("false")
  })
})
