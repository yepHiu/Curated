import { computed } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsPhotoLibraryPathsSection from "./SettingsPhotoLibraryPathsSection.vue"
import SettingsComicLibraryPathsSection from "./SettingsComicLibraryPathsSection.vue"

const access = vi.hoisted(() => ({ allowed: true }))
vi.mock("@/composables/use-library-path-access", () => ({
  useLibraryPathAccess: () => ({ canManagePaths: computed(() => access.allowed) }),
}))
beforeEach(() => { access.allowed = true })

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Check: { name: "Check", template: "<span />" },
  Download: { name: "Download", template: "<span />" },
  FolderArchive: { name: "FolderArchive", template: "<span />" },
  MoreVertical: { name: "MoreVertical", template: "<span />" },
  RefreshCw: { name: "RefreshCw", template: "<span />" },
  Trash2: { name: "Trash2", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled", "ariaLabel"],
    emits: ["click"],
    template:
      "<button v-bind='$attrs' :disabled='disabled' :aria-label='ariaLabel' @click=\"$emit('click', $event)\"><slot /></button>",
  },
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

vi.mock("./SettingsLibraryPathAddDialog.vue", () => ({
  default: {
    name: "SettingsLibraryPathAddDialog",
    props: [
      "open",
      "newPath",
      "newPathTitle",
      "pickDirectoryBusy",
      "directoryHintDisplay",
      "pathAddError",
      "addBusy",
      "canSaveNewPath",
      "contentClass",
      "triggerLabel",
      "dialogTitle",
      "dialogDescription",
      "pathLabel",
      "pathPlaceholder",
      "pathInputId",
      "titleLabel",
      "titlePlaceholder",
      "titleInputId",
      "examplePaths",
    ],
    emits: [
      "update:open",
      "update:newPath",
      "update:newPathTitle",
      "clearError",
      "browse",
      "submit",
    ],
    template:
      "<div data-add-dialog><span data-trigger-label>{{ triggerLabel }}</span><span data-dialog-title>{{ dialogTitle }}</span><span data-dialog-description>{{ dialogDescription }}</span><button data-add-open @click=\"$emit('update:open', true)\">open</button><button data-new-path @click=\"$emit('update:newPath', 'E:/Comics')\">path</button><button data-new-title @click=\"$emit('update:newPathTitle', 'Comics')\">title</button><button data-clear-error @click=\"$emit('clearError')\">clear</button><button data-browse @click=\"$emit('browse')\">browse</button><button data-submit @click=\"$emit('submit')\">submit</button></div>",
  },
}))

const comicPath = {
  id: "comic-path-a",
  title: "Main comics",
  path: "D:/Comics",
}

const secondaryComicPath = {
  id: "comic-path-b",
  title: "Inbox",
  path: "E:/Inbox/Comics",
}

const baseProps = {
  paths: [comicPath, secondaryComicPath],
  defaultImportLibraryPathId: "comic-path-a",
  addPathDialogOpen: false,
  newPath: "",
  newPathTitle: "",
  pickDirectoryBusy: false,
  directoryHintDisplay: "",
  pathAddError: "",
  addBusy: false,
  canSaveNewPath: false,
  defaultSaving: false,
  scanPathBusy: null,
  dialogContentClass: "dialog-content",
}

describe.each([
  ["comic", SettingsComicLibraryPathsSection],
  ["photo", SettingsPhotoLibraryPathsSection],
] as const)("%s library paths", (kind, component) => {
  it("uses the shared dialog-style storage path add flow", () => {
    const wrapper = mount(component, {
      props: baseProps,
    })

    expect(wrapper.find("[data-add-dialog]").exists()).toBe(true)
    expect(wrapper.findAll("input")).toHaveLength(0)
    expect(wrapper.get("[data-trigger-label]").text()).toBe(`settings.${kind}LibraryPathAdd`)
    expect(wrapper.get("[data-dialog-title]").text()).toBe(`settings.${kind}LibraryPathDialogTitle`)
    expect(wrapper.get("[data-dialog-description]").text()).toBe(
      `settings.${kind}LibraryPathDialogDesc`,
    )
  })

  it("renders the default import path with title and disk path like video storage", () => {
    const wrapper = mount(component, {
      props: baseProps,
    })

    expect(wrapper.find("[role=combobox], .select-stub").exists()).toBe(false)
    expect(wrapper.findAll("[data-default-import-path]")).toHaveLength(1)
    expect(wrapper.get("[data-default-import-path]").attributes("data-library-path")).toBe("comic-path-a")
    expect(wrapper.get("[data-default-import-path]").text()).toContain("settings.defaultImportPathLabel")
    expect(wrapper.text()).toContain("E:/Inbox/Comics")
  })

  it("forwards add dialog, default selection, scan, and remove events", async () => {
    const wrapper = mount(component, {
      props: baseProps,
    })

    await wrapper.get("[data-library-path='comic-path-b'] [data-set-default-import-path]").trigger("click")
    await wrapper.get("[data-add-open]").trigger("click")
    await wrapper.get("[data-new-path]").trigger("click")
    await wrapper.get("[data-new-title]").trigger("click")
    await wrapper.get("[data-clear-error]").trigger("click")
    await wrapper.get("[data-browse]").trigger("click")
    await wrapper.get("[data-submit]").trigger("click")
    await wrapper.get(`[data-scan-${kind}-path='comic-path-a']`).trigger("click")
    await wrapper.get(`[data-remove-${kind}-path='comic-path-a']`).trigger("click")

    expect(wrapper.emitted("changeDefaultImportPath")).toEqual([["comic-path-b"]])
    expect(wrapper.emitted("update:addPathDialogOpen")).toEqual([[true]])
    expect(wrapper.emitted("update:newPath")).toEqual([["E:/Comics"]])
    expect(wrapper.emitted("update:newPathTitle")).toEqual([["Comics"]])
    expect(wrapper.emitted("clearError")).toHaveLength(1)
    expect(wrapper.emitted("browse")).toHaveLength(1)
    expect(wrapper.emitted("submit")).toHaveLength(1)
    expect(wrapper.emitted("scanPath")).toEqual([[comicPath]])
    expect(wrapper.emitted("removePath")).toEqual([["comic-path-a"]])
  })

  it("blocks redundant and concurrent default changes and updates the selected row", async () => {
    const wrapper = mount(component, { props: baseProps })
    const defaultAction = "[data-library-path='comic-path-a'] [data-set-default-import-path]"
    const otherAction = "[data-library-path='comic-path-b'] [data-set-default-import-path]"
    expect(wrapper.get(defaultAction).attributes("disabled")).toBeDefined()
    await wrapper.setProps({ defaultSaving: true })
    expect(wrapper.get(otherAction).attributes("disabled")).toBeDefined()
    await wrapper.get(otherAction).trigger("click")
    expect(wrapper.emitted("changeDefaultImportPath")).toBeUndefined()
    await wrapper.setProps({ defaultSaving: false, defaultImportLibraryPathId: "comic-path-b" })
    expect(wrapper.get("[data-default-import-path]").attributes("data-library-path")).toBe("comic-path-b")
    expect(wrapper.get(defaultAction).attributes("disabled")).toBeUndefined()
    wrapper.unmount()
  })

  it("uses the more-actions menu instead of a visible remove button for comic paths", () => {
    const wrapper = mount(component, {
      props: baseProps,
    })

    expect(wrapper.find('button[aria-label="settings.moreActions"]').exists()).toBe(true)
    expect(wrapper.find(`[data-scan-${kind}-path='comic-path-a']`).exists()).toBe(true)
    expect(wrapper.find(`[data-remove-${kind}-path='comic-path-a']`).exists()).toBe(true)
  })
})


it("shows only server path information remotely even when edit dialogs were open", () => {
  access.allowed = false
  const wrapper = mount(SettingsComicLibraryPathsSection, {
    props: { ...baseProps, addPathDialogOpen: true },
  })
  expect(wrapper.find("[data-readonly-library-paths]").exists()).toBe(true)
  expect(wrapper.find("button, input, select, [role=combobox]").exists()).toBe(false)
  for (const path of baseProps.paths) expect(wrapper.text()).toContain(path.path)
  expect(wrapper.text()).toContain("settings.libraryPathsReadOnly")
  expect(wrapper.emitted()).toEqual({})
  wrapper.unmount()
})


it("also makes photo directories and the default import target read-only remotely", () => {
  access.allowed = false
  const wrapper = mount(SettingsPhotoLibraryPathsSection, { props: { ...baseProps, addPathDialogOpen: true } })
  expect(wrapper.find("[data-readonly-library-paths]").exists()).toBe(true)
  expect(wrapper.find("button, input, select, [role=combobox]").exists()).toBe(false)
  for (const path of baseProps.paths) expect(wrapper.text()).toContain(path.path)
  wrapper.unmount()
})
