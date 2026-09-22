import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsWishlistSection from "./SettingsWishlistSection.vue"

const mocks = vi.hoisted(() => ({ save: vi.fn(), wishlist: { integrationsAvailable: true } }))
const enabled = ref(false)
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({
  wishlist: mocks.wishlist, browserPluginEnabled: enabled, setBrowserPluginEnabled: mocks.save,
}) }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

beforeEach(() => {
  enabled.value = false
  mocks.wishlist.integrationsAvailable = true
  mocks.save.mockReset()
})

describe("browser plugin integration setting", () => {
  it("waits for saving, prevents duplicate changes, and allows retry after failure", async () => {
    let fail!: (error: Error) => void
    mocks.save.mockImplementationOnce(() => new Promise<void>((_, reject) => { fail = reject }))
    const wrapper = mount(SettingsWishlistSection)
    const toggle = wrapper.get('[role="switch"]')
    await toggle.trigger("click")
    expect(mocks.save).toHaveBeenCalledWith(true)
    expect(toggle.attributes("disabled")).toBeDefined()
    expect(toggle.attributes("aria-checked")).toBe("false")
    expect(wrapper.get('[role="group"]').attributes("aria-busy")).toBe("true")
    expect(wrapper.get('[role="status"]').text()).toBe("common.saving")
    fail(new Error("disk unavailable"))
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toBe("wishlist.pluginSaveError")
    expect(toggle.attributes("disabled")).toBeUndefined()
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    mocks.save.mockImplementationOnce(async (value: boolean) => { enabled.value = value })
    await toggle.trigger("click")
    await flushPromises()
    expect(toggle.attributes("aria-checked")).toBe("true")
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.find('input[type="password"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it("disables real plugin integration in Mock mode", () => {
    mocks.wishlist.integrationsAvailable = false
    const wrapper = mount(SettingsWishlistSection)
    expect(wrapper.get('[role="switch"]').attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("wishlist.mockHint")
    wrapper.unmount()
  })
})
