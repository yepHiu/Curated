import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import ImportMenu from "./ImportMenu.vue"

const comicServiceState = vi.hoisted(() => ({
  comicLibraryEnabled: false,
  comicLibraryPaths: [
    { id: "comic-path-a", path: "D:/Comics", title: "Comics" },
  ],
  defaultComicImportLibraryPathId: "comic-path-a",
  refreshSettings: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  BookOpen: { name: "BookOpen", template: "<span />" },
  FileArchive: { name: "FileArchive", template: "<span />" },
  UploadCloud: { name: "UploadCloud", template: "<span />" },
  X: { name: "X", template: "<span />" },
}))

vi.mock("./MovieImportDialog.vue", () => ({
  default: { name: "MovieImportDialog", template: "<div data-import-menu-movie />" },
}))

vi.mock("./ComicImportDialog.vue", () => ({
  default: { name: "ComicImportDialog", template: "<div data-import-menu-comic />" },
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => ({
    comicLibraryEnabled: {
      get value() {
        return comicServiceState.comicLibraryEnabled
      },
    },
    comicLibraryPaths: {
      get value() {
        return comicServiceState.comicLibraryPaths
      },
    },
    defaultComicImportLibraryPathId: {
      get value() {
        return comicServiceState.defaultComicImportLibraryPathId
      },
    },
    refreshSettings: comicServiceState.refreshSettings,
  }),
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

beforeEach(() => {
  comicServiceState.comicLibraryEnabled = false
  comicServiceState.comicLibraryPaths = [
    { id: "comic-path-a", path: "D:/Comics", title: "Comics" },
  ]
  comicServiceState.defaultComicImportLibraryPathId = "comic-path-a"
  comicServiceState.refreshSettings.mockReset()
})

describe("ImportMenu", () => {
  it("contains only the movie import entry when comics are disabled", async () => {
    comicServiceState.comicLibraryEnabled = false

    const wrapper = mount(ImportMenu, {
      global: {
        stubs: {
          MovieImportDialog: { template: "<div data-import-menu-movie />" },
          ComicImportDialog: { template: "<div data-import-menu-comic />" },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find("[data-import-menu-movie]").exists()).toBe(true)
    expect(wrapper.find("[data-import-menu-comic]").exists()).toBe(false)
  })

  it("contains movie and comic import entries when comics are enabled", async () => {
    comicServiceState.comicLibraryEnabled = true

    const wrapper = mount(ImportMenu, {
      global: {
        stubs: {
          MovieImportDialog: { template: "<div data-import-menu-movie />" },
          ComicImportDialog: { template: "<div data-import-menu-comic />" },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find("[data-import-menu-movie]").exists()).toBe(true)
    expect(wrapper.find("[data-import-menu-comic]").exists()).toBe(true)
  })
})
