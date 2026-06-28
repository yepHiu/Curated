import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsComicLibraryPathsSection from "./SettingsComicLibraryPathsSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  FolderArchive: { name: "FolderArchive", template: "<span />" },
  Trash2: { name: "Trash2", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/select", () => ({
  Select: {
    name: "Select",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<div class='select-stub' :data-model-value='modelValue' :data-disabled='String(!!disabled)'><slot /></div>",
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
  dialogContentClass: "dialog-content",
}

describe("SettingsComicLibraryPathsSection", () => {
  it("uses the shared dialog-style storage path add flow", () => {
    const wrapper = mount(SettingsComicLibraryPathsSection, {
      props: baseProps,
    })

    expect(wrapper.find("[data-add-dialog]").exists()).toBe(true)
    expect(wrapper.findAll("input")).toHaveLength(0)
    expect(wrapper.get("[data-trigger-label]").text()).toBe("settings.comicLibraryPathAdd")
    expect(wrapper.get("[data-dialog-title]").text()).toBe("settings.comicLibraryPathDialogTitle")
    expect(wrapper.get("[data-dialog-description]").text()).toBe(
      "settings.comicLibraryPathDialogDesc",
    )
  })

  it("renders the default import path with title and disk path like video storage", () => {
    const wrapper = mount(SettingsComicLibraryPathsSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.comicDefaultImportPath")
    expect(wrapper.text()).toContain("Main comics 路 D:/Comics")
    expect(wrapper.text()).toContain("E:/Inbox/Comics")
  })

  it("forwards add dialog, default selection, and remove events", async () => {
    const wrapper = mount(SettingsComicLibraryPathsSection, {
      props: baseProps,
    })

    wrapper.getComponent({ name: "Select" }).vm.$emit("update:modelValue", "comic-path-b")
    await wrapper.get("[data-add-open]").trigger("click")
    await wrapper.get("[data-new-path]").trigger("click")
    await wrapper.get("[data-new-title]").trigger("click")
    await wrapper.get("[data-clear-error]").trigger("click")
    await wrapper.get("[data-browse]").trigger("click")
    await wrapper.get("[data-submit]").trigger("click")
    await wrapper.get("[data-remove-comic-path='comic-path-a']").trigger("click")

    expect(wrapper.emitted("changeDefaultImportPath")).toEqual([["comic-path-b"]])
    expect(wrapper.emitted("update:addPathDialogOpen")).toEqual([[true]])
    expect(wrapper.emitted("update:newPath")).toEqual([["E:/Comics"]])
    expect(wrapper.emitted("update:newPathTitle")).toEqual([["Comics"]])
    expect(wrapper.emitted("clearError")).toHaveLength(1)
    expect(wrapper.emitted("browse")).toHaveLength(1)
    expect(wrapper.emitted("submit")).toHaveLength(1)
    expect(wrapper.emitted("removePath")).toEqual([["comic-path-a"]])
  })
})
