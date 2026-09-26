import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { useLibraryPathAccess } from "./use-library-path-access"

const health = vi.hoisted(() => vi.fn())
vi.mock("@/api/endpoints", () => ({ api: { health } }))

function render() {
  return mount(defineComponent({
    setup() { return useLibraryPathAccess() },
    template: "<span>{{ canManagePaths }}</span>",
  }))
}

beforeEach(() => {
  vi.stubEnv("VITE_USE_WEB_API", "true")
  vi.stubEnv("VITE_API_BASE_URL", "http://localhost:8080/api")
  delete window.javLibrary
  health.mockReset().mockResolvedValue({ canManageLibraryPaths: true })
})
afterEach(() => { vi.unstubAllEnvs(); delete window.javLibrary })

describe("library path access", () => {
  it("starts read-only and enables a verified local browser", async () => {
    const wrapper = render()
    expect(wrapper.text()).toBe("false")
    await flushPromises()
    expect(wrapper.text()).toBe("true")
    wrapper.unmount()
  })

  it.each(["legacy", "desktop"])("permits a verified local %s Desktop, independently of update distribution", async (distribution) => {
    window.javLibrary = { getDesktopInfo: vi.fn().mockResolvedValue({ serverOrigin: "http://localhost:8080", distribution }) }
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toBe("true")
    wrapper.unmount()
  })

  it.each(["remote-api", "remote-desktop", "old-bridge", "bridge-failure", "remote-permission", "old-server", "health-failure"])("keeps %s read-only", async (kind) => {
    if (kind === "remote-api") vi.stubEnv("VITE_API_BASE_URL", "http://192.168.1.20:8081/api")
    if (kind === "remote-desktop") window.javLibrary = { getDesktopInfo: vi.fn().mockResolvedValue({ serverOrigin: "http://192.168.1.20:8081" }) }
    if (kind === "old-bridge") window.javLibrary = {}
    if (kind === "bridge-failure") window.javLibrary = { getDesktopInfo: vi.fn().mockRejectedValue(new Error("offline")) }
    if (kind === "remote-permission") health.mockResolvedValue({ canManageLibraryPaths: false })
    if (kind === "old-server") health.mockResolvedValue({})
    if (kind === "health-failure") health.mockRejectedValue(new Error("offline"))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toBe("false")
    wrapper.unmount()
  })

  it("keeps Mock directory editing available without contacting Server", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toBe("true")
    expect(health).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
