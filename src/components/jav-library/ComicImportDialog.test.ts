import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import ComicImportDialog from "./ComicImportDialog.vue"

const serviceState = vi.hoisted(() => ({
  comicLibraryPaths: [
    { id: "comic-path-a", path: "D:/Comics", title: "Comics" },
  ],
  defaultComicImportLibraryPathId: "comic-path-a",
  comicLibraryPathStorageStatuses: [] as Array<Record<string, unknown>>,
  refreshSettings: vi.fn(),
  checkComicLibraryPathStorageStatus: vi.fn(),
  importComics: vi.fn(),
}))

const tracker = vi.hoisted(() => ({
  start: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  BookOpen: { name: "BookOpen", template: "<span data-book-open />" },
  FileArchive: { name: "FileArchive", template: "<span />" },
  UploadCloud: { name: "UploadCloud", template: "<span />" },
  X: { name: "X", template: "<span />" },
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => ({
    comicLibraryPaths: computed(() => serviceState.comicLibraryPaths),
    defaultComicImportLibraryPathId: computed(
      () => serviceState.defaultComicImportLibraryPathId,
    ),
    comicLibraryPathStorageStatuses: computed(
      () => serviceState.comicLibraryPathStorageStatuses,
    ),
    refreshSettings: serviceState.refreshSettings,
    checkComicLibraryPathStorageStatus: serviceState.checkComicLibraryPathStorageStatus,
    importComics: serviceState.importComics,
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
  serviceState.comicLibraryPaths = [
    { id: "comic-path-a", path: "D:/Comics", title: "Comics" },
  ]
  serviceState.defaultComicImportLibraryPathId = "comic-path-a"
  serviceState.comicLibraryPathStorageStatuses = []
  serviceState.refreshSettings.mockReset()
  serviceState.checkComicLibraryPathStorageStatus.mockReset()
  serviceState.importComics.mockReset()
  tracker.start.mockReset()
})

describe("ComicImportDialog", () => {
  it("accepts zip and cbz files while skipping unsupported archives", async () => {
    const wrapper = mount(ComicImportDialog)
    const zip = new File(["zip"], "Book One.zip", { type: "application/zip" })
    const cbz = new File(["cbz"], "Book Two.cbz", { type: "application/vnd.comicbook+zip" })
    const rar = new File(["rar"], "Book Three.rar", { type: "application/vnd.rar" })
    const input = wrapper.get<HTMLInputElement>("[data-comic-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [zip, cbz, rar],
      configurable: true,
    })

    await input.trigger("change")

    expect(wrapper.text()).toContain("Book One.zip")
    expect(wrapper.text()).toContain("Book Two.cbz")
    expect(wrapper.text()).not.toContain("Book Three.rar")
    expect(wrapper.text()).toContain("import.comicSkippedUnsupported")
  })

  it("disables submit without a default comic import path", () => {
    serviceState.defaultComicImportLibraryPathId = ""
    serviceState.comicLibraryPaths = []

    const wrapper = mount(ComicImportDialog)

    expect(wrapper.get("[data-comic-import-submit]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("import.comicNoDefaultPath")
  })

  it("disables submit when default comic import storage is unavailable", async () => {
    serviceState.comicLibraryPathStorageStatuses = [
      {
        libraryPathId: "comic-path-a",
        path: "D:/Comics",
        title: "Comics",
        status: "offline",
        message: "drive is offline",
        canImport: false,
        canRescan: false,
      },
    ]
    const wrapper = mount(ComicImportDialog)
    const file = new File(["cbz"], "Book One.cbz", { type: "application/vnd.comicbook+zip" })
    const input = wrapper.get<HTMLInputElement>("[data-comic-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [file],
      configurable: true,
    })

    await input.trigger("change")

    expect(wrapper.get("[data-comic-import-submit]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("import.storageUnavailable")
    expect(serviceState.importComics).not.toHaveBeenCalled()
  })

  it("imports selected comic archives and starts task tracking", async () => {
    serviceState.importComics.mockResolvedValueOnce({
      taskId: "comic-import-1",
      type: "import.comics",
      status: "completed",
      createdAt: "2026-06-28T00:00:00.000Z",
      progress: 100,
    })
    const wrapper = mount(ComicImportDialog)
    const file = new File(["cbz"], "Book One.cbz", { type: "application/vnd.comicbook+zip" })
    const input = wrapper.get<HTMLInputElement>("[data-comic-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [file],
      configurable: true,
    })

    await input.trigger("change")
    await wrapper.get("[data-comic-import-submit]").trigger("click")
    await flushPromises()

    expect(serviceState.checkComicLibraryPathStorageStatus).toHaveBeenCalledWith(["comic-path-a"])
    expect(serviceState.importComics).toHaveBeenCalledWith(
      [file],
      expect.objectContaining({ onUploadProgress: expect.any(Function) }),
    )
    expect(tracker.start).toHaveBeenCalledWith("comic-import-1")
  })

  it("shows upload progress while a comic import is running", async () => {
    let resolveImport: (value: {
      taskId: string
      type: "import.comics"
      status: "completed"
      createdAt: string
      progress: number
    }) => void
    serviceState.importComics.mockImplementationOnce((_files, options) => {
      options.onUploadProgress({ loaded: 4, total: 10, percent: 40 })
      return new Promise((resolve) => {
        resolveImport = resolve
      })
    })
    const wrapper = mount(ComicImportDialog)
    const file = new File(["cbz"], "Book One.cbz", { type: "application/vnd.comicbook+zip" })
    const input = wrapper.get<HTMLInputElement>("[data-comic-import-file-input]")
    Object.defineProperty(input.element, "files", {
      value: [file],
      configurable: true,
    })

    await input.trigger("change")
    await wrapper.get("[data-comic-import-submit]").trigger("click")
    await flushPromises()

    const triggerProgress = wrapper.get("[data-comic-import-trigger-progress]")
    expect(triggerProgress.attributes("aria-valuenow")).toBe("40")
    expect(wrapper.find("[data-progress]").exists()).toBe(true)
    expect(wrapper.text()).toContain("import.copyingProgress")

    resolveImport!({
      taskId: "comic-import-1",
      type: "import.comics",
      status: "completed",
      createdAt: "2026-06-28T00:00:00.000Z",
      progress: 100,
    })
    await flushPromises()

    expect(wrapper.find("[data-comic-import-trigger-progress]").exists()).toBe(false)
    expect(wrapper.get("[data-comic-import-trigger]").find("[data-book-open]").exists()).toBe(true)
  })
})
