import { flushPromises, shallowMount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import DevDebugDialog from "./DevDebugDialog.vue"

const mocks = vi.hoisted(() => ({ service: vi.fn(), access: vi.fn() }))
vi.mock("@/services/library-service", () => ({ useLibraryService: mocks.service }))
vi.mock("@/composables/use-server-local-access", () => ({ useServerLocalAccess: mocks.access }))
vi.mock("vue-i18n", async (importOriginal) => ({ ...await importOriginal<typeof import("vue-i18n")>(), useI18n: () => ({ t: (key: string) => key }) }))

const playerSettings = ref({ forceStreamPush: false, streamPushEnabled: true })
const isServerLocal = ref(true)
const refreshSettings = vi.fn()
const patchPlayerSettings = vi.fn()
const pingProxyJavbus = vi.fn()
const pingProxyGoogle = vi.fn()
let wrapper: ReturnType<typeof render> | undefined
function render(open = true) {
  return shallowMount(DevDebugDialog, {
    props: { open },
    global: { renderStubDefaultSlot: true },
  })
}
function button(key: string) {
  return wrapper!.findAllComponents({ name: "Button" }).find((item) => item.text() === key)!
}
beforeEach(() => {
  vi.stubEnv("VITE_USE_WEB_API", "true")
  vi.clearAllMocks()
  refreshSettings.mockResolvedValue(undefined)
  patchPlayerSettings.mockResolvedValue(undefined)
  playerSettings.value = { forceStreamPush: false, streamPushEnabled: true }
  isServerLocal.value = true
  mocks.access.mockReturnValue({ isServerLocal })
  mocks.service.mockReturnValue({ playerSettings, refreshSettings, patchPlayerSettings, pingProxyJavbus, pingProxyGoogle })
})
afterEach(() => { wrapper?.unmount(); vi.unstubAllEnvs() })

describe("debug tools", () => {
  it("does not fetch while closed or trigger any mutation when opened", async () => {
    wrapper = render(false)
    expect(refreshSettings).not.toHaveBeenCalled()
    await wrapper.setProps({ open: true })
    await flushPromises()
    expect(refreshSettings).toHaveBeenCalledTimes(1)
    expect(patchPlayerSettings).not.toHaveBeenCalled()
    expect(pingProxyJavbus).not.toHaveBeenCalled()
    expect(wrapper.findComponent({ name: "SettingsLoggingSection" }).exists()).toBe(true)
  })

  it("keeps configuration controls unavailable after a read failure and supports retry", async () => {
    refreshSettings.mockRejectedValueOnce(new Error("offline"))
    wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain("offline")
    expect(wrapper.findComponent({ name: "SettingsLoggingSection" }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: "Switch" }).attributes("disabled")).toBe("true")
    await button("debug.retry").vm.$emit("click")
    await flushPromises()
    expect(wrapper.findComponent({ name: "SettingsLoggingSection" }).exists()).toBe(true)
  })

  it("patches only force HLS and reports failures without changing the saved value", async () => {
    patchPlayerSettings.mockRejectedValueOnce(new Error("save failed"))
    wrapper = render()
    await flushPromises()
    wrapper.findComponent({ name: "Switch" }).vm.$emit("update:modelValue", true)
    await flushPromises()
    expect(patchPlayerSettings).toHaveBeenCalledWith({ forceStreamPush: true })
    expect(playerSettings.value.forceStreamPush).toBe(false)
    expect(wrapper.text()).toContain("save failed")
  })

  it("does not expose force HLS for remote servers", async () => {
    isServerLocal.value = false
    wrapper = render()
    await flushPromises()
    expect(wrapper.findComponent({ name: "Switch" }).exists()).toBe(false)
  })

  it("prevents forcing HLS when streaming is disabled", async () => {
    playerSettings.value.streamPushEnabled = false
    wrapper = render()
    await flushPromises()
    wrapper.findComponent({ name: "Switch" }).vm.$emit("update:modelValue", true)
    expect(patchPlayerSettings).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("debug.streamDisabled")
  })

  it("runs probes only on request, blocks duplicates and shows failed probe results", async () => {
    let complete!: (result: { ok: boolean; latencyMs: number; message: string }) => void
    pingProxyJavbus.mockReturnValue(new Promise((resolve) => { complete = resolve }))
    wrapper = render()
    await flushPromises()
    button("debug.metadata").vm.$emit("click")
    button("debug.metadata").vm.$emit("click")
    expect(pingProxyJavbus).toHaveBeenCalledTimes(1)
    expect(pingProxyJavbus).toHaveBeenCalledWith()
    complete({ ok: false, latencyMs: 30, message: "unreachable" })
    await flushPromises()
    expect(wrapper.text()).toContain("unreachable")
    expect(wrapper.text()).toContain("debug.failed")
  })

  it("disables network actions in Mock mode", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")
    wrapper = render()
    await flushPromises()
    button("debug.network").vm.$emit("click")
    expect(pingProxyGoogle).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("debug.webOnly")
  })
})
