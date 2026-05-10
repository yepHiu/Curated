import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import MovieImportDialog from "./MovieImportDialog.vue"

const serviceState = vi.hoisted(() => ({
  libraryPaths: [
    { id: "library-a", path: "D:/Media/JAV/Main", title: "Primary archive" },
  ],
  defaultImportLibraryPathId: "library-a",
  libraryPathStorageStatuses: [] as Array<Record<string, unknown>>,
  refreshSettings: vi.fn(),
  checkLibraryPathStorageStatus: vi.fn(),
  importMovies: vi.fn(),
}))

const tracker = vi.hoisted(() => ({
  start: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  FilePlus2: { name: "FilePlus2", template: "<span data-file-plus />" },
  FolderInput: { name: "FolderInput", template: "<span />" },
  UploadCloud: { name: "UploadCloud", template: "<span />" },
  X: { name: "X", template: "<span />" },
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    libraryPaths: computed(() => serviceState.libraryPaths),
    defaultImportLibraryPathId: computed(() => serviceState.defaultImportLibraryPathId),
    libraryPathStorageStatuses: computed(() => serviceState.libraryPathStorageStatuses),
    refreshSettings: serviceState.refreshSettings,
    checkLibraryPathStorageStatus: serviceState.checkLibraryPathStorageStatus,
    importMovies: serviceState.importMovies,
  }),
}))

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => tracker,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: { name: "Dialog", template: "<div><slot /></div>" },
  DialogTrigger: { name: "DialogTrigger", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<section><slot /></section>" },
  DialogDescription: { name: "DialogDescription", template: "<p><slot /></p>" },
  DialogFooter: { name: "DialogFooter", template: "<footer><slot /></footer>" },
  DialogHeader: { name: "DialogHeader", template: "<header><slot /></header>" },
  DialogTitle: { name: "DialogTitle", template: "<h2><slot /></h2>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled"],
    template: "<button :disabled='disabled'><slot /></button>",
  },
}))

vi.mock("@/components/ui/progress", () => ({
  Progress: { name: "Progress", props: ["modelValue"], template: "<div data-progress />" },
}))

beforeEach(() => {
  serviceState.libraryPaths = [
    { id: "library-a", path: "D:/Media/JAV/Main", title: "Primary archive" },
  ]
  serviceState.defaultImportLibraryPathId = "library-a"
  serviceState.libraryPathStorageStatuses = []
  serviceState.refreshSettings.mockReset()
  serviceState.checkLibraryPathStorageStatus.mockReset()
  serviceState.importMovies.mockReset()
  tracker.start.mockReset()
})

describe("MovieImportDialog", () => {
  it("disables submit without a default import path", () => {
    serviceState.defaultImportLibraryPathId = ""
    const wrapper = mount(MovieImportDialog)

    expect(wrapper.get("[data-import-submit]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("import.noDefaultPath")
  })

  it("imports selected files and starts task tracking", async () => {
    serviceState.importMovies.mockResolvedValueOnce({
      taskId: "import-1",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-01T00:00:00.000Z",
      progress: 100,
    })
    const wrapper = mount(MovieImportDialog)
    const file = new File(["movie"], "IMP-001.mp4", { type: "video/mp4" })
    const input = wrapper.get<HTMLInputElement>("[data-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [file],
      configurable: true,
    })

    await input.trigger("change")
    await wrapper.get("[data-import-submit]").trigger("click")
    await flushPromises()

    expect(serviceState.importMovies).toHaveBeenCalledWith(
      [file],
      expect.objectContaining({ onUploadProgress: expect.any(Function) }),
    )
    expect(tracker.start).toHaveBeenCalledWith("import-1")
  })

  it("shows circular upload progress on the trigger while an import is still running", async () => {
    let resolveImport: (value: {
      taskId: string
      type: "import.movies"
      status: "completed"
      createdAt: string
      progress: number
    }) => void
    serviceState.importMovies.mockImplementationOnce((_files, options) => {
      options.onUploadProgress({ loaded: 4, total: 10, percent: 40 })
      return new Promise((resolve) => {
        resolveImport = resolve
      })
    })
    const wrapper = mount(MovieImportDialog)
    const file = new File(["movie"], "IMP-001.mp4", { type: "video/mp4" })
    const input = wrapper.get<HTMLInputElement>("[data-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [file],
      configurable: true,
    })

    await input.trigger("change")
    await wrapper.get("[data-import-submit]").trigger("click")
    await flushPromises()

    const triggerProgress = wrapper.get("[data-import-trigger-progress]")
    expect(triggerProgress.attributes("aria-valuenow")).toBe("40")
    expect(wrapper.get("[data-import-trigger]").find("[data-file-plus]").exists()).toBe(false)

    resolveImport!({
      taskId: "import-1",
      type: "import.movies",
      status: "completed",
      createdAt: "2026-05-01T00:00:00.000Z",
      progress: 100,
    })
    await flushPromises()

    expect(wrapper.find("[data-import-trigger-progress]").exists()).toBe(false)
    expect(wrapper.get("[data-import-trigger]").find("[data-file-plus]").exists()).toBe(true)
  })

  it("does not expose the removed local-copy mode", () => {
    const wrapper = mount(MovieImportDialog)

    expect(wrapper.find("[data-import-mode-local]").exists()).toBe(false)
    expect(wrapper.find("[data-import-local-paths]").exists()).toBe(false)
  })

  it("disables submit when the default import storage is offline", async () => {
    serviceState.libraryPathStorageStatuses = [
      {
        libraryPathId: "library-a",
        path: "D:/Media/JAV/Main",
        title: "Primary archive",
        status: "offline",
        message: "drive is offline",
        canImport: false,
        canRescan: false,
      },
    ]
    const wrapper = mount(MovieImportDialog)
    const file = new File(["movie"], "IMP-001.mp4", { type: "video/mp4" })
    const input = wrapper.get<HTMLInputElement>("[data-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [file],
      configurable: true,
    })

    await input.trigger("change")

    expect(wrapper.get("[data-import-submit]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("import.storageUnavailable")
  })
})
