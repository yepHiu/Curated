import { computed } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { ComicLibraryService } from "@/services/contracts/comic-library-service"
import SettingsComicLibrarySection from "./SettingsComicLibrarySection.vue"

type MockFunction = ReturnType<typeof vi.fn>
type TestComicService = ComicLibraryService & {
  setComicLibraryEnabled: MockFunction
  patchComicReader: MockFunction
  patchComicCache: MockFunction
  cleanupComicCache: MockFunction
}

const mockState = vi.hoisted<{
  comicService: TestComicService | null
}>(() => ({
  comicService: null,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  BookOpen: { name: "BookOpen", template: "<span />" },
  Database: { name: "Database", template: "<span />" },
  FolderOpen: { name: "FolderOpen", template: "<span />" },
  FolderPlus: { name: "FolderPlus", template: "<span />" },
  FolderArchive: { name: "FolderArchive", template: "<span />" },
  PanelsTopLeft: { name: "PanelsTopLeft", template: "<span />" },
  Trash2: { name: "Trash2", template: "<span />" },
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => mockState.comicService,
}))

vi.mock("@/lib/pick-directory", () => ({
  pickLibraryDirectory: vi.fn(),
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<p><slot /></p>" },
  CardHeader: { name: "CardHeader", template: "<header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h3><slot /></h3>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled", "variant", "size"],
    emits: ["click"],
    template: "<button :disabled='disabled' @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: {
    name: "Dialog",
    props: ["open"],
    emits: ["update:open"],
    template: "<div><slot /></div>",
  },
  DialogClose: { name: "DialogClose", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<div><slot /></div>" },
  DialogDescription: { name: "DialogDescription", template: "<p><slot /></p>" },
  DialogFooter: { name: "DialogFooter", template: "<footer><slot /></footer>" },
  DialogHeader: { name: "DialogHeader", template: "<header><slot /></header>" },
  DialogTitle: { name: "DialogTitle", template: "<h3><slot /></h3>" },
  DialogTrigger: { name: "DialogTrigger", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<input :value='modelValue' @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
}))

vi.mock("@/components/ui/select", () => ({
  Select: {
    name: "Select",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template:
      "<div class='select-stub' :data-model-value='String(modelValue)'><slot /></div>",
  },
  SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
  SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
  SelectTrigger: { name: "SelectTrigger", template: "<div><slot /></div>" },
  SelectValue: {
    name: "SelectValue",
    props: ["placeholder"],
    template: "<span><slot>{{ placeholder }}</slot></span>",
  },
}))

vi.mock("@/components/ui/switch", () => ({
  Switch: {
    name: "Switch",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<button class='switch-stub' :disabled='disabled' @click=\"$emit('update:modelValue', !modelValue)\"><slot /></button>",
  },
}))

function createComicServiceMock(
  overrides: Partial<TestComicService> = {},
): TestComicService {
  const service = {
    comics: computed(() => []),
    comicsLoaded: computed(() => true),
    loadError: computed(() => null),
    comicLibraryEnabled: computed(() => false),
    comicLibraryPaths: computed(() => []),
    defaultComicImportLibraryPathId: computed(() => ""),
    comicReader: computed(() => ({
      mode: "page" as const,
      fit: "contain" as const,
      direction: "rtl" as const,
    })),
    comicCache: computed(() => ({
      maxBytes: 2 * 1024 * 1024 * 1024,
    })),
    refreshSettings: vi.fn().mockResolvedValue(undefined),
    setComicLibraryEnabled: vi.fn().mockResolvedValue(undefined),
    addComicLibraryPath: vi.fn().mockResolvedValue(null),
    updateComicLibraryPathTitle: vi.fn().mockResolvedValue(undefined),
    removeComicLibraryPath: vi.fn().mockResolvedValue(undefined),
    setDefaultComicImportLibraryPathId: vi.fn().mockResolvedValue(undefined),
    patchComicReader: vi.fn().mockResolvedValue(undefined),
    patchComicCache: vi.fn().mockResolvedValue(undefined),
    reloadComicsFromApi: vi.fn().mockResolvedValue(undefined),
    getComicById: vi.fn(),
    loadComicDetail: vi.fn().mockResolvedValue(undefined),
    patchComic: vi.fn().mockResolvedValue(undefined),
    deleteComic: vi.fn().mockResolvedValue(undefined),
    revealComicSource: vi.fn().mockResolvedValue(undefined),
    scanComics: vi.fn().mockResolvedValue(null),
    getComicProgress: vi.fn(),
    saveComicProgress: vi.fn(),
    resetComicProgress: vi.fn(),
    getComicPreferences: vi.fn(),
    saveComicPreferences: vi.fn(),
    getComicCacheStatus: vi.fn().mockResolvedValue({
      maxBytes: 2 * 1024 * 1024 * 1024,
      usedBytes: 0,
      entryCount: 0,
    }),
    cleanupComicCache: vi.fn().mockResolvedValue({
      maxBytes: 2 * 1024 * 1024 * 1024,
      usedBytes: 0,
      entryCount: 0,
    }),
    ...overrides,
  } as TestComicService
  return service
}

describe("SettingsComicLibrarySection", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockState.comicService = createComicServiceMock()
  })

  it("renders disabled state with an enable action", () => {
    const wrapper = mount(SettingsComicLibrarySection)

    expect(wrapper.text()).toContain("settings.comicLibraryTitle")
    expect(wrapper.text()).toContain("settings.comicLibraryDisabledDesc")
    expect(wrapper.find("[data-comic-enable]").exists()).toBe(true)
  })

  it("requires at least one comic path before enabling", async () => {
    const wrapper = mount(SettingsComicLibrarySection)

    await wrapper.get("[data-comic-enable]").trigger("click")

    expect(wrapper.text()).toContain("settings.comicLibraryPathRequired")
    expect(mockState.comicService?.setComicLibraryEnabled).not.toHaveBeenCalled()
  })

  it("renders path, reader, and cache sections when enabled", () => {
    mockState.comicService = createComicServiceMock({
      comicLibraryEnabled: computed(() => true),
      comicLibraryPaths: computed(() => [
        {
          id: "comic-path-1",
          path: "D:/Comics",
          title: "Comics",
        },
      ]),
      defaultComicImportLibraryPathId: computed(() => "comic-path-1"),
    })

    const wrapper = mount(SettingsComicLibrarySection)

    expect(wrapper.find("[data-comic-paths]").exists()).toBe(true)
    expect(wrapper.find("[data-comic-reader]").exists()).toBe(true)
    expect(wrapper.find("[data-comic-cache]").exists()).toBe(true)
    expect(wrapper.text()).toContain("settings.comicDefaultImportPath")
  })

  it("disables comic library without deleting comic data or cache", async () => {
    mockState.comicService = createComicServiceMock({
      comicLibraryEnabled: computed(() => true),
      comicLibraryPaths: computed(() => [
        {
          id: "comic-path-1",
          path: "D:/Comics",
          title: "Comics",
        },
      ]),
    })
    const wrapper = mount(SettingsComicLibrarySection)

    await wrapper.get("[data-comic-disable]").trigger("click")

    expect(mockState.comicService?.setComicLibraryEnabled).toHaveBeenCalledWith(false)
    expect(mockState.comicService?.cleanupComicCache).not.toHaveBeenCalled()
  })

  it("offers expected comic cache size presets", () => {
    mockState.comicService = createComicServiceMock({
      comicLibraryEnabled: computed(() => true),
      comicLibraryPaths: computed(() => [
        {
          id: "comic-path-1",
          path: "D:/Comics",
          title: "Comics",
        },
      ]),
    })

    const wrapper = mount(SettingsComicLibrarySection)

    expect(wrapper.text()).toContain("settings.comicCachePreset1gb")
    expect(wrapper.text()).toContain("settings.comicCachePreset2gb")
    expect(wrapper.text()).toContain("settings.comicCachePreset5gb")
    expect(wrapper.text()).toContain("settings.comicCachePreset10gb")
    expect(wrapper.text()).toContain("settings.comicCachePresetUnlimited")
  })
})
