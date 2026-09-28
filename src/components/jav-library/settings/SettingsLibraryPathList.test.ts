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
    props: ["path", "revealBusy", "scanBusy", "isDefault", "defaultImportPathSaving", "canRebind", "scanDisabled"],
    emits: ["reveal", "edit", "rescan", "remove", "setDefault", "rebindStorage"],
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
  storageStatuses: [],
  storageBindingBusy: "",
  defaultImportLibraryPathId: "a",
  defaultImportPathSaving: false,
  editingLibraryPathId: null,
  editLibraryTitleDraft: "",
  editTitleBusy: false,
  editTitleError: "",
  revealPathBusy: "",
  scanPathBusy: "",
}

describe("SettingsLibraryPathList", () => {
  it("renders path rows without batch selection and forwards row actions", async () => {
    const wrapper = mount(SettingsLibraryPathList, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("Archive A")
    expect(wrapper.text()).toContain("D:/Media/A")
    expect(wrapper.find("input[type='checkbox']").exists()).toBe(false)

    await wrapper.get("[data-row-reveal]").trigger("click")
    await wrapper.get("[data-row-edit]").trigger("click")
    await wrapper.get("[data-row-rescan]").trigger("click")
    await wrapper.get("[data-row-remove]").trigger("click")

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

  it("moves the unique default marker when the saved directory changes", async () => {
    const wrapper = mount(SettingsLibraryPathList, { props: baseProps })
    expect(wrapper.findAll("[data-default-import-path]")).toHaveLength(1)
    expect(wrapper.get("[data-default-import-path]").attributes("data-library-path")).toBe("a")
    wrapper.getComponent({ name: "SettingsLibraryPathActions" }).vm.$emit("setDefault", "b")
    expect(wrapper.emitted("changeDefaultImportLibraryPath")).toEqual([["b"]])
    await wrapper.setProps({ defaultImportLibraryPathId: "b", defaultImportPathSaving: true })
    expect(wrapper.findAll("[data-default-import-path]")).toHaveLength(1)
    expect(wrapper.get("[data-default-import-path]").attributes("data-library-path")).toBe("b")
    for (const menu of wrapper.findAllComponents({ name: "SettingsLibraryPathActions" })) {
      expect(menu.props("defaultImportPathSaving")).toBe(true)
    }
  })

  it("renders storage status labels and emits rebind actions", async () => {
    const wrapper = mount(SettingsLibraryPathList, {
      props: {
        ...baseProps,
        storageStatuses: [
          {
            libraryPathId: "a",
            path: "D:/Media/A",
            title: "Archive A",
            status: "online",
            message: "online",
            checkedAt: "2026-05-11T00:00:00Z",
            canRescan: true,
            canImport: true,
          },
          {
            libraryPathId: "b",
            path: "E:/Media/B",
            title: "Archive B",
            status: "volume_mismatch",
            message: "wrong disk",
            checkedAt: "2026-05-11T00:00:00Z",
            expectedVolumeId: "OLD",
            currentVolumeId: "NEW",
            canRescan: false,
            canImport: false,
          },
        ],
      },
    })

    expect(wrapper.text()).toContain("settings.storageStatusOnline")
    expect(wrapper.text()).toContain("settings.storageStatusVolumeMismatch")
    expect(wrapper.find('[title="settings.storageStatusMessages.volume_mismatch"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain("wrong disk")

    const actions = wrapper.findAllComponents({ name: "SettingsLibraryPathActions" })[1]!
    expect(actions.props("scanDisabled")).toBe(true)
    expect(actions.props("canRebind")).toBe(true)
    actions.vm.$emit("rebindStorage", paths[1])

    expect(wrapper.emitted("rebindStorage")).toEqual([[paths[1]]])
  })
})
