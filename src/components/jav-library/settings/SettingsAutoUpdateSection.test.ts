import { computed } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsAutoUpdateSection from "./SettingsAutoUpdateSection.vue"

const localPermission = vi.hoisted(() => ({ value: true }))
vi.mock("@/composables/use-app-update", () => ({ useAppUpdate: () => ({ localUpdateAllowed: computed(() => localPermission.value) }) }))
beforeEach(() => { localPermission.value = true })

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("SettingsAutoUpdateSection", () => {
  it("hides Server auto-download preferences on remote or unknown connections", () => {
    localPermission.value = false
    const wrapper = mount(SettingsAutoUpdateSection, { props: { enabled: true, saving: false, error: "" } })
    expect(wrapper.find('[role="switch"]').exists()).toBe(false)
    expect(wrapper.text()).toBe("")
  })

  it("shows save state and emits a changed preference", async () => {
    const wrapper = mount(SettingsAutoUpdateSection, {
      props: { enabled: false, saving: true, error: "save failed" },
    })

    expect(wrapper.text()).toContain("settings.autoDownloadUpdatesTitle")
    expect(wrapper.text()).toContain("settings.autoDownloadUpdatesSyncing")
    expect(wrapper.get('[role="alert"]').text()).toBe("save failed")

    wrapper.findComponent({ name: "Switch" }).vm.$emit("update:modelValue", true)
    expect(wrapper.emitted("change")).toEqual([[true]])
  })
})
