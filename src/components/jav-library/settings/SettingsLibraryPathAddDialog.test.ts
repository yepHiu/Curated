import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsLibraryPathAddDialog from "./SettingsLibraryPathAddDialog.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  FolderOpen: { name: "FolderOpen", template: "<span />" },
  FolderPlus: { name: "FolderPlus", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" :title=\"$attrs.title\" @click=\"$emit('click', $event)\"><slot /></button>",
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
  DialogDescription: { name: "DialogDescription", template: "<div><slot /></div>" },
  DialogFooter: { name: "DialogFooter", template: "<div><slot /></div>" },
  DialogHeader: { name: "DialogHeader", template: "<div><slot /></div>" },
  DialogTitle: { name: "DialogTitle", template: "<h2><slot /></h2>" },
  DialogTrigger: { name: "DialogTrigger", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue", "input"],
    template:
      "<input :value=\"modelValue\" @input=\"$emit('input', $event); $emit('update:modelValue', $event.target.value)\" />",
  },
}))

const baseProps = {
  open: true,
  newPath: "D:/Media/JAV",
  newPathTitle: "Main",
  pickDirectoryBusy: false,
  directoryHintDisplay: "selected folder",
  pathAddError: "",
  addBusy: false,
  canSaveNewPath: true,
  contentClass: "dialog-content",
}

describe("SettingsLibraryPathAddDialog", () => {
  it("renders add path form and emits field updates", async () => {
    const wrapper = mount(SettingsLibraryPathAddDialog, {
      props: baseProps,
    })
    const inputs = wrapper.findAll("input")

    expect(wrapper.text()).toContain("settings.addPath")
    expect(wrapper.text()).toContain("settings.addPathDialogTitle")
    expect(wrapper.text()).toContain("settings.addPathDialogDesc")
    expect(wrapper.text()).toContain("selected folder")

    await inputs[0]!.setValue("E:/Movies")
    await inputs[1]!.setValue("Movies")

    expect(wrapper.emitted("update:newPath")).toEqual([["E:/Movies"]])
    expect(wrapper.emitted("update:newPathTitle")).toEqual([["Movies"]])
    expect(wrapper.emitted("clearError")).toHaveLength(1)
  })

  it("emits browse, submit, and open updates", async () => {
    const wrapper = mount(SettingsLibraryPathAddDialog, {
      props: baseProps,
    })

    await wrapper.get("[data-add-path-browse]").trigger("click")
    await wrapper.get("[data-add-path-submit]").trigger("click")
    wrapper.getComponent({ name: "Dialog" }).vm.$emit("update:open", false)

    expect(wrapper.emitted("browse")).toHaveLength(1)
    expect(wrapper.emitted("submit")).toHaveLength(1)
    expect(wrapper.emitted("update:open")).toEqual([[false]])
  })

  it("renders invalid path and save disabled states", () => {
    const wrapper = mount(SettingsLibraryPathAddDialog, {
      props: {
        ...baseProps,
        newPath: "relative/path",
        pathAddError: "cannot add",
        addBusy: false,
        canSaveNewPath: false,
      },
    })

    expect(wrapper.text()).toContain("settings.notAbsolute")
    expect(wrapper.text()).toContain("cannot add")
    expect(wrapper.get("[data-add-path-submit]").attributes("disabled")).toBeDefined()
    expect(wrapper.get("[data-add-path-submit]").attributes("title")).toBe(
      "settings.savePathDisabledTitle",
    )
  })
})
