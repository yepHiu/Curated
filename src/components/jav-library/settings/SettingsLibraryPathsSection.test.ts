import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsLibraryPathsSection from "./SettingsLibraryPathsSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Database: { name: "Database", template: "<span />" },
  RefreshCw: { name: "RefreshCw", template: "<span />" },
  CircleHelp: { name: "CircleHelp", template: "<span />" },
}))

vi.mock("reka-ui", () => ({
  TooltipContent: { name: "TooltipContent", template: "<div><slot /></div>" },
  TooltipPortal: { name: "TooltipPortal", template: "<div><slot /></div>" },
  TooltipProvider: { name: "TooltipProvider", template: "<div><slot /></div>" },
  TooltipRoot: { name: "TooltipRoot", template: "<div><slot /></div>" },
  TooltipTrigger: { name: "TooltipTrigger", template: "<div><slot /></div>" },
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
    props: ["disabled"],
    emits: ["click"],
    template: "<button :disabled='disabled' @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/select", () => ({
  Select: {
    name: "Select",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<div class=\"select-stub\" :data-model-value=\"modelValue\" :data-disabled=\"String(!!disabled)\"><slot /></div>",
  },
  SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
  SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
  SelectTrigger: { name: "SelectTrigger", template: "<div><slot /></div>" },
  SelectValue: {
    name: "SelectValue",
    props: ["placeholder"],
    template: "<div><slot>{{ placeholder }}</slot></div>",
  },
}))

vi.mock("./SettingsLibraryPathToolbar.vue", () => ({
  default: {
    name: "SettingsLibraryPathToolbar",
    props: [
      "batchMode",
      "libraryPathsCount",
      "hasMetadataPathSelection",
      "metadataRefreshBusy",
    ],
    emits: ["enterBatchMode", "selectAll", "clearSelection", "refreshMetadata", "exitBatchMode"],
    template:
      "<div data-toolbar><button data-enter @click=\"$emit('enterBatchMode')\">enter</button><button data-select-all @click=\"$emit('selectAll')\">select</button><button data-clear @click=\"$emit('clearSelection')\">clear</button><button data-refresh @click=\"$emit('refreshMetadata')\">refresh</button><button data-exit @click=\"$emit('exitBatchMode')\">exit</button></div>",
  },
}))

vi.mock("./SettingsLibraryPathRemoveDialog.vue", () => ({
  default: {
    name: "SettingsLibraryPathRemoveDialog",
    props: ["open", "pending", "busy", "contentClass"],
    emits: ["update:open", "confirm"],
    template:
      "<div data-remove-dialog><button data-remove-open @click=\"$emit('update:open', false)\">close</button><button data-remove-confirm @click=\"$emit('confirm')\">confirm</button></div>",
  },
}))

vi.mock("./SettingsLibraryPathList.vue", () => ({
  default: {
    name: "SettingsLibraryPathList",
    props: [
      "paths",
      "storageStatuses",
      "storageBindingBusy",
      "batchMode",
      "selectedMetadataRefreshPaths",
      "editingLibraryPathId",
      "editLibraryTitleDraft",
      "editTitleBusy",
      "editTitleError",
      "revealPathBusy",
      "scanPathBusy",
    ],
    emits: [
      "update:editLibraryTitleDraft",
      "saveTitle",
      "cancelEdit",
      "toggleMetadataPathSelection",
      "reveal",
      "edit",
      "rescan",
      "rebindStorage",
      "remove",
    ],
    template:
      "<div data-list><button data-draft @click=\"$emit('update:editLibraryTitleDraft', 'Renamed')\">draft</button><button data-save @click=\"$emit('saveTitle', paths[0].id)\">save</button><button data-cancel @click=\"$emit('cancelEdit')\">cancel</button><button data-toggle @click=\"$emit('toggleMetadataPathSelection', paths[0].path)\">toggle</button><button data-reveal @click=\"$emit('reveal', paths[0])\">reveal</button><button data-edit @click=\"$emit('edit', paths[0])\">edit</button><button data-rescan @click=\"$emit('rescan', paths[0])\">rescan</button><button data-rebind @click=\"$emit('rebindStorage', paths[0])\">rebind</button><button data-remove @click=\"$emit('remove', paths[0])\">remove</button></div>",
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
      "<div data-add-dialog><button data-add-open @click=\"$emit('update:open', true)\">open</button><button data-new-path @click=\"$emit('update:newPath', 'E:/Movies')\">path</button><button data-new-title @click=\"$emit('update:newPathTitle', 'Movies')\">title</button><button data-clear-error @click=\"$emit('clearError')\">clear</button><button data-browse @click=\"$emit('browse')\">browse</button><button data-submit @click=\"$emit('submit')\">submit</button></div>",
  },
}))

const libraryPath = {
  id: "library-a",
  title: "Primary archive",
  path: "D:/Media/JAV/Main",
}

const secondaryLibraryPath = {
  id: "library-b",
  title: "Downloads",
  path: "E:/Inbox/Movies",
}

const baseProps = {
  scanFeedbackError: "",
  paths: [libraryPath, secondaryLibraryPath],
  storageStatuses: [],
  storageStatusBusy: false,
  storageStatusError: "",
  storageBindingBusy: null,
  defaultImportLibraryPathId: "library-a",
  defaultImportPathSaving: false,
  defaultImportPathError: "",
  batchMode: true,
  hasMetadataPathSelection: true,
  metadataRefreshBusy: false,
  metadataRefreshSuccess: "",
  metadataRefreshError: "",
  selectedMetadataRefreshPaths: ["D:/Media/JAV/Main"],
  removePathDialogOpen: false,
  removePathPending: libraryPath,
  removePathBusy: false,
  editingLibraryPathId: null,
  editLibraryTitleDraft: "",
  editTitleBusy: false,
  editTitleError: "",
  revealPathBusy: null,
  scanPathBusy: null,
  addPathDialogOpen: false,
  newPath: "",
  newPathTitle: "",
  pickDirectoryBusy: false,
  directoryHintDisplay: "",
  pathAddError: "",
  addBusy: false,
  canSaveNewPath: false,
  dialogContentClass: "dialog-content",
}

describe("SettingsLibraryPathsSection", () => {
  it("renders storage copy and path feedback states", () => {
    const wrapper = mount(SettingsLibraryPathsSection, {
      props: {
        ...baseProps,
        scanFeedbackError: "scan failed",
        metadataRefreshSuccess: "metadata queued",
        metadataRefreshError: "metadata failed",
        defaultImportPathError: "default path failed",
        storageStatusError: "storage failed",
      },
    })

    expect(wrapper.text()).toContain("settings.storageCardTitle")
    expect(wrapper.text()).toContain("settings.storageCardDesc")
    expect(wrapper.text()).toContain("settings.defaultImportPathLabel")
    expect(wrapper.text()).toContain("Primary archive · D:/Media/JAV/Main")
    expect(wrapper.text()).toContain("settings.defaultImportPathDesc")
    expect(wrapper.text()).toContain("settings.defaultImportPathHelpAria")
    expect(wrapper.text()).toContain("scan failed")
    expect(wrapper.text()).toContain("metadata queued")
    expect(wrapper.text()).toContain("metadata failed")
    expect(wrapper.text()).toContain("default path failed")
    expect(wrapper.text()).toContain("storage failed")
  })

  it("emits default import path changes", () => {
    const wrapper = mount(SettingsLibraryPathsSection, {
      props: baseProps,
    })

    wrapper.getComponent({ name: "Select" }).vm.$emit("update:modelValue", "library-b")

    expect(wrapper.get(".select-stub").attributes("data-model-value")).toBe("library-a")
    expect(wrapper.emitted("changeDefaultImportLibraryPath")).toEqual([["library-b"]])
  })

  it("disables default import path selection when no library paths exist", () => {
    const wrapper = mount(SettingsLibraryPathsSection, {
      props: {
        ...baseProps,
        paths: [],
        defaultImportLibraryPathId: "",
      },
    })

    expect(wrapper.get(".select-stub").attributes("data-disabled")).toBe("true")
    expect(wrapper.text()).toContain("settings.defaultImportPathNone")
  })

  it("forwards toolbar, list, remove dialog, and add dialog events", async () => {
    const wrapper = mount(SettingsLibraryPathsSection, {
      props: baseProps,
    })

    await wrapper.get("[data-enter]").trigger("click")
    await wrapper.get("[data-select-all]").trigger("click")
    await wrapper.get("[data-clear]").trigger("click")
    await wrapper.get("[data-refresh]").trigger("click")
    await wrapper.get("[data-check-storage-status]").trigger("click")
    await wrapper.get("[data-exit]").trigger("click")
    await wrapper.get("[data-remove-open]").trigger("click")
    await wrapper.get("[data-remove-confirm]").trigger("click")
    await wrapper.get("[data-draft]").trigger("click")
    await wrapper.get("[data-save]").trigger("click")
    await wrapper.get("[data-cancel]").trigger("click")
    await wrapper.get("[data-toggle]").trigger("click")
    await wrapper.get("[data-reveal]").trigger("click")
    await wrapper.get("[data-edit]").trigger("click")
    await wrapper.get("[data-rescan]").trigger("click")
    await wrapper.get("[data-rebind]").trigger("click")
    await wrapper.get("[data-remove]").trigger("click")
    await wrapper.get("[data-add-open]").trigger("click")
    await wrapper.get("[data-new-path]").trigger("click")
    await wrapper.get("[data-new-title]").trigger("click")
    await wrapper.get("[data-clear-error]").trigger("click")
    await wrapper.get("[data-browse]").trigger("click")
    await wrapper.get("[data-submit]").trigger("click")

    expect(wrapper.emitted("enterBatchMode")).toHaveLength(1)
    expect(wrapper.emitted("selectAll")).toHaveLength(1)
    expect(wrapper.emitted("clearSelection")).toHaveLength(1)
    expect(wrapper.emitted("refreshMetadata")).toHaveLength(1)
    expect(wrapper.emitted("checkStorage")).toHaveLength(1)
    expect(wrapper.emitted("exitBatchMode")).toHaveLength(1)
    expect(wrapper.emitted("update:removePathDialogOpen")).toEqual([[false]])
    expect(wrapper.emitted("confirmRemove")).toHaveLength(1)
    expect(wrapper.emitted("update:editLibraryTitleDraft")).toEqual([["Renamed"]])
    expect(wrapper.emitted("saveTitle")).toEqual([["library-a"]])
    expect(wrapper.emitted("cancelEdit")).toHaveLength(1)
    expect(wrapper.emitted("toggleMetadataPathSelection")).toEqual([["D:/Media/JAV/Main"]])
    expect(wrapper.emitted("reveal")).toEqual([[libraryPath]])
    expect(wrapper.emitted("edit")).toEqual([[libraryPath]])
    expect(wrapper.emitted("rescan")).toEqual([[libraryPath]])
    expect(wrapper.emitted("rebindStorage")).toEqual([[libraryPath]])
    expect(wrapper.emitted("remove")).toEqual([[libraryPath]])
    expect(wrapper.emitted("update:addPathDialogOpen")).toEqual([[true]])
    expect(wrapper.emitted("update:newPath")).toEqual([["E:/Movies"]])
    expect(wrapper.emitted("update:newPathTitle")).toEqual([["Movies"]])
    expect(wrapper.emitted("clearError")).toHaveLength(1)
    expect(wrapper.emitted("browse")).toHaveLength(1)
    expect(wrapper.emitted("submit")).toHaveLength(1)
  })

  it("places the storage check action next to the add path action", () => {
    const wrapper = mount(SettingsLibraryPathsSection, {
      props: baseProps,
    })

    const addActionParent = wrapper.get("[data-add-dialog]").element.parentElement
    const checkActionParent = wrapper.get("[data-check-storage-status]").element.parentElement

    expect(checkActionParent).toBe(addActionParent)
  })

  it("uses the standard storage action button size for the storage check action", () => {
    const wrapper = mount(SettingsLibraryPathsSection, {
      props: baseProps,
    })

    const checkStorageClass = wrapper.get("[data-check-storage-status]").attributes("class")

    expect(checkStorageClass).toContain("h-8")
    expect(checkStorageClass).toContain("min-w-28")
    expect(checkStorageClass).toContain("px-3")
  })
})
