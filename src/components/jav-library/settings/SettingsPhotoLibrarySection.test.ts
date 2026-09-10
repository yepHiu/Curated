import { computed } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { PhotoLibraryService } from "@/services/contracts/photo-library-service"
import SettingsPhotoLibrarySection from "./SettingsPhotoLibrarySection.vue"

type MockFunction = ReturnType<typeof vi.fn>
type TestPhotoService = PhotoLibraryService & {
  setPhotoLibraryEnabled: MockFunction
  setAutoPhotoLibraryWatch: MockFunction
  addPhotoLibraryPath: MockFunction
  updatePhotoLibraryPathTitle: MockFunction
  removePhotoLibraryPath: MockFunction
  setDefaultPhotoImportLibraryPathId: MockFunction
  patchPhotoViewer: MockFunction
  patchPhotoCache: MockFunction
  scanPhotos: MockFunction
}

const mockState = vi.hoisted<{
  photoService: TestPhotoService | null
}>(() => ({
  photoService: null,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  FolderArchive: { name: "FolderArchive", template: "<span />" },
  FolderOpen: { name: "FolderOpen", template: "<span />" },
  FolderPlus: { name: "FolderPlus", template: "<span />" },
  Images: { name: "Images", template: "<span />" },
  Database: { name: "Database", template: "<span />" },
  MoreVertical: { name: "MoreVertical", template: "<span />" },
  PanelsTopLeft: { name: "PanelsTopLeft", template: "<span />" },
  RefreshCw: { name: "RefreshCw", template: "<span />" },
  Trash2: { name: "Trash2", template: "<span />" },
}))

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => mockState.photoService,
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
    props: ["disabled", "variant", "size", "ariaLabel"],
    emits: ["click"],
    template:
      "<button v-bind='$attrs' :disabled='disabled' :aria-label='ariaLabel' @click=\"$emit('click', $event)\"><slot /></button>",
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

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div><slot /></div>" },
  DropdownMenuContent: { name: "DropdownMenuContent", template: "<div><slot /></div>" },
  DropdownMenuGroup: { name: "DropdownMenuGroup", template: "<div><slot /></div>" },
  DropdownMenuTrigger: { name: "DropdownMenuTrigger", template: "<div><slot /></div>" },
  DropdownMenuItem: {
    name: "DropdownMenuItem",
    props: ["disabled", "variant"],
    emits: ["click"],
    template:
      "<button v-bind='$attrs' :disabled='disabled' :data-variant='variant' @click=\"$emit('click', $event)\"><slot /></button>",
  },
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
  SelectItem: { name: "SelectItem", props: ["value"], template: "<button><slot /></button>" },
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

function createPhotoServiceMock(
  overrides: Partial<TestPhotoService> = {},
): TestPhotoService {
  return {
    photoLibraryEnabled: computed(() => false),
    autoPhotoLibraryWatch: computed(() => true),
    photoLibraryPaths: computed(() => []),
    defaultPhotoImportLibraryPathId: computed(() => ""),
    photoViewer: computed(() => ({
      mode: "page" as const,
      fit: "contain" as const,
      direction: "ltr" as const,
    })),
    photoCache: computed(() => ({
      maxBytes: 5 * 1024 * 1024 * 1024,
    })),
    refreshSettings: vi.fn().mockResolvedValue(undefined),
    setPhotoLibraryEnabled: vi.fn().mockResolvedValue(undefined),
    setAutoPhotoLibraryWatch: vi.fn().mockResolvedValue(undefined),
    addPhotoLibraryPath: vi.fn().mockResolvedValue(null),
    updatePhotoLibraryPathTitle: vi.fn().mockResolvedValue(undefined),
    removePhotoLibraryPath: vi.fn().mockResolvedValue(undefined),
    setDefaultPhotoImportLibraryPathId: vi.fn().mockResolvedValue(undefined),
    patchPhotoViewer: vi.fn().mockResolvedValue(undefined),
    patchPhotoCache: vi.fn().mockResolvedValue(undefined),
    loadPhotoDetail: vi.fn().mockResolvedValue(undefined),
    scanPhotos: vi.fn().mockResolvedValue(null),
    ...overrides,
  } as TestPhotoService
}

describe("SettingsPhotoLibrarySection", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockState.photoService = createPhotoServiceMock()
  })

  it("renders disabled state with an enable action and empty path state", () => {
    const wrapper = mount(SettingsPhotoLibrarySection)

    expect(wrapper.text()).toContain("settings.photoLibraryTitle")
    expect(wrapper.text()).toContain("settings.photoLibraryDisabledDesc")
    expect(wrapper.text()).toContain("settings.photoLibraryPathsEmpty")
    expect(wrapper.find("[data-photo-enable]").exists()).toBe(true)
  })

  it("requires at least one photo path before enabling", async () => {
    const wrapper = mount(SettingsPhotoLibrarySection)

    await wrapper.get("[data-photo-enable]").trigger("click")

    expect(wrapper.text()).toContain("settings.photoLibraryPathRequired")
    expect(mockState.photoService?.setPhotoLibraryEnabled).not.toHaveBeenCalled()
  })

  it("enables the photo library through photo settings when a path exists", async () => {
    mockState.photoService = createPhotoServiceMock({
      photoLibraryPaths: computed(() => [
        {
          id: "photo-path-1",
          path: "D:/Photos",
          title: "Photos",
        },
      ]),
    })
    const wrapper = mount(SettingsPhotoLibrarySection)

    await wrapper.get("[data-photo-enable]").trigger("click")

    expect(mockState.photoService?.setPhotoLibraryEnabled).toHaveBeenCalledWith(true)
  })

  it("renders photo path controls and removes a configured path", async () => {
    mockState.photoService = createPhotoServiceMock({
      photoLibraryEnabled: computed(() => true),
      photoLibraryPaths: computed(() => [
        {
          id: "photo-path-1",
          path: "D:/Photos",
          title: "Photos",
        },
      ]),
      defaultPhotoImportLibraryPathId: computed(() => "photo-path-1"),
    })
    const wrapper = mount(SettingsPhotoLibrarySection)

    expect(wrapper.text()).toContain("settings.photoDefaultImportPath")
    await wrapper.get("[data-remove-photo-path='photo-path-1']").trigger("click")

    expect(mockState.photoService?.removePhotoLibraryPath).toHaveBeenCalledWith("photo-path-1")
    expect(mockState.photoService?.refreshSettings).toHaveBeenCalled()
  })

  it("scans a configured photo path from the path actions menu", async () => {
    mockState.photoService = createPhotoServiceMock({
      photoLibraryEnabled: computed(() => true),
      photoLibraryPaths: computed(() => [
        {
          id: "photo-path-1",
          path: "D:/Photos",
          title: "Photos",
        },
      ]),
      defaultPhotoImportLibraryPathId: computed(() => "photo-path-1"),
    })
    const wrapper = mount(SettingsPhotoLibrarySection)

    await wrapper.get("[data-scan-photo-path='photo-path-1']").trigger("click")

    expect(mockState.photoService?.scanPhotos).toHaveBeenCalledWith(["D:/Photos"])
  })

  it("toggles automatic photo library watching independently", async () => {
    mockState.photoService = createPhotoServiceMock({
      photoLibraryEnabled: computed(() => true),
      autoPhotoLibraryWatch: computed(() => false),
    })
    const wrapper = mount(SettingsPhotoLibrarySection)

    await wrapper.get("[data-photo-auto-watch-switch]").trigger("click")

    expect(mockState.photoService?.setAutoPhotoLibraryWatch).toHaveBeenCalledWith(true)
    expect(mockState.photoService?.setPhotoLibraryEnabled).not.toHaveBeenCalled()
  })

  it("shows viewer defaults when the photo library is enabled", () => {
    mockState.photoService = createPhotoServiceMock({
      photoLibraryEnabled: computed(() => true),
    })

    const wrapper = mount(SettingsPhotoLibrarySection)

    expect(wrapper.find("[data-photo-viewer]").exists()).toBe(true)
    expect(wrapper.find("[data-photo-viewer-settings-list]").exists()).toBe(true)
    expect(wrapper.text()).toContain("settings.photoViewerMode")
    expect(wrapper.text()).toContain("settings.photoViewerFit")
    expect(wrapper.text()).toContain("settings.photoViewerDirection")
  })
})
