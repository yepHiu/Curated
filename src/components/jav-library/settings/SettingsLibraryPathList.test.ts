import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsLibraryPathList from "./SettingsLibraryPathList.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.title ? `${key}:${params.title}` : key,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue", "keydown"],
    template:
      "<input :value=\"modelValue\" @input=\"$emit('update:modelValue', $event.target.value)\" @keydown=\"$emit('keydown', $event)\" />",
  },
}))

vi.mock("./SettingsLibraryPathActions.vue", () => ({
  default: {
    name: "SettingsLibraryPathActions",
    props: ["path", "revealBusy", "scanBusy"],
    emits: ["reveal", "edit", "rescan", "remove"],
    template:
      "<div><button data-row-reveal @click=\"$emit('reveal', path)\">reveal</button><button data-row-edit @click=\"$emit('edit', path)\">edit</button><button data-row-rescan @click=\"$emit('rescan', path)\">rescan</button><button data-row-remove @click=\"$emit('remove', path)\">remove</button></div>",
  },
}))

const paths = [
  { id: "a", title: "Archive A", path: "D:/Media/A" },
  { id: "b", title: "Archive B", path: "E:/Media/B" },
]

const baseProps = {
  paths,
  batchMode: true,
  selectedMetadataRefreshPaths: ["D:/Media/A"],
  editingLibraryPathId: null,
  editLibraryTitleDraft: "",
  editTitleBusy: false,
  editTitleError: "",
  revealPathBusy: "",
  scanPathBusy: "",
}

describe("SettingsLibraryPathList", () => {
  it("renders path rows with batch checkboxes and row action events", async () => {
    const wrapper = mount(SettingsLibraryPathList, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("Archive A")
    expect(wrapper.text()).toContain("D:/Media/A")
    expect(wrapper.get("input[type='checkbox']").attributes("checked")).toBeDefined()

    await wrapper.get("input[type='checkbox']").setValue(false)
    await wrapper.get("[data-row-reveal]").trigger("click")
    await wrapper.get("[data-row-edit]").trigger("click")
    await wrapper.get("[data-row-rescan]").trigger("click")
    await wrapper.get("[data-row-remove]").trigger("click")

    expect(wrapper.emitted("toggleMetadataPathSelection")).toEqual([["D:/Media/A"]])
    expect(wrapper.emitted("reveal")).toEqual([[paths[0]]])
    expect(wrapper.emitted("edit")).toEqual([[paths[0]]])
    expect(wrapper.emitted("rescan")).toEqual([[paths[0]]])
    expect(wrapper.emitted("remove")).toEqual([[paths[0]]])
  })

  it("renders edit mode and emits title save/cancel actions", async () => {
    const wrapper = mount(SettingsLibraryPathList, {
      props: {
        ...baseProps,
        editingLibraryPathId: "a",
        editLibraryTitleDraft: "Draft title",
        editTitleError: "save failed",
      },
    })
    const input = wrapper.get("input")

    expect(wrapper.text()).toContain("settings.pathReadonly")
    expect(wrapper.text()).toContain("D:/Media/A")
    expect(wrapper.text()).toContain("save failed")

    await input.setValue("Renamed")
    await wrapper.get("[data-save-library-path-title='a']").trigger("click")
    await wrapper.get("[data-cancel-library-path-title='a']").trigger("click")

    expect(wrapper.emitted("update:editLibraryTitleDraft")).toEqual([["Renamed"]])
    expect(wrapper.emitted("saveTitle")).toEqual([["a"]])
    expect(wrapper.emitted("cancelEdit")).toHaveLength(1)
  })

  it("hides checkboxes outside batch mode", () => {
    const wrapper = mount(SettingsLibraryPathList, {
      props: {
        ...baseProps,
        batchMode: false,
      },
    })

    expect(wrapper.find("input[type='checkbox']").exists()).toBe(false)
  })
})
