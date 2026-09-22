import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import WishlistPluginStatus from "./WishlistPluginStatus.vue"

const service = vi.hoisted(() => ({
  wishlist: { integrationsAvailable: true },
  listConnectedClients: vi.fn(),
}))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => service }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

afterEach(() => {
  service.wishlist.integrationsAvailable = true
  vi.clearAllMocks()
  vi.useRealTimers()
})

describe("wishlist plugin connection status", () => {
  it("recognizes recent plugin traffic and expires it without treating ordinary browser traffic as a plugin", async () => {
    vi.useFakeTimers()
    const dto = (sampledAt: string) => ({
      clients: [
        { browser: "Curated Plugin", lastSeen: "2026-09-23T10:00:00Z" },
        { browser: "Chrome", lastSeen: sampledAt },
      ],
      sampledAt, total: 2, localCount: 2, remoteCount: 0,
    })
    service.listConnectedClients.mockResolvedValueOnce(dto("2026-09-23T10:04:59Z"))
      .mockResolvedValueOnce(dto("2026-09-23T10:05:01Z"))
      .mockRejectedValueOnce(new Error("offline"))
    const wrapper = mount(WishlistPluginStatus)
    expect(wrapper.text()).toContain("pluginStatus.checking")
    await flushPromises()
    expect(wrapper.text()).toContain("pluginStatus.connected")
    expect(wrapper.find("button").exists()).toBe(false)
    await vi.advanceTimersByTimeAsync(15_000)
    expect(wrapper.text()).toContain("pluginStatus.disconnected")
    await vi.advanceTimersByTimeAsync(15_000)
    expect(wrapper.text()).toContain("pluginStatus.unknown")
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(15_000)
    expect(service.listConnectedClients).toHaveBeenCalledTimes(3)
  })

  it("does not request or imply a real connection in Mock mode", async () => {
    service.wishlist.integrationsAvailable = false
    const wrapper = mount(WishlistPluginStatus)
    await flushPromises()
    expect(wrapper.text()).toContain("pluginStatus.unavailable")
    expect(service.listConnectedClients).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
