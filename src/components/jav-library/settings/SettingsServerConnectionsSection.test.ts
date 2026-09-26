import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import SettingsServerConnectionsSection from "./SettingsServerConnectionsSection.vue"

const health = vi.hoisted(() => ({ status: "online", probing: false, checkNow: vi.fn() }))
vi.mock("@/composables/use-backend-health", async () => {
  const { computed } = await import("vue")
  return { useBackendHealth: () => ({ status: computed(() => health.status), probing: computed(() => health.probing), checkNow: health.checkNow }) }
})
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const state = {
  currentServerUrl: "http://nas.local:8081", connecting: false,
  servers: [
    { id: "home", name: "家庭 NAS", url: "http://nas.local:8081" },
    { id: "office", name: "工作室", url: "https://office.example.com" },
  ],
}
let wrappers: ReturnType<typeof mount>[] = []
function render() {
  const wrapper = mount(SettingsServerConnectionsSection)
  wrappers.push(wrapper)
  return wrapper
}
function bridge() {
  const getServerConnections = vi.fn().mockResolvedValue(structuredClone(state))
  const openServerConnections = vi.fn().mockResolvedValue(undefined)
  window.javLibrary = { getServerConnections, openServerConnections }
  return { getServerConnections, openServerConnections }
}
beforeEach(() => { delete window.javLibrary; health.status = "online"; health.checkNow.mockClear() })
afterEach(() => { wrappers.forEach(wrapper => wrapper.unmount()); wrappers = []; delete window.javLibrary; vi.useRealTimers() })

describe("Desktop servers in settings", () => {
  it("hides the card in Web and older Desktop versions", () => {
    expect(render().find('[data-server-connections]').exists()).toBe(false)
    window.javLibrary = { openServerConnections: vi.fn() }
    expect(render().find('[data-server-connections]').exists()).toBe(false)
  })
  it("shows actual current origin, live status and saved records, only connecting by saved id", async () => {
    const api = bridge()
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain("家庭 NAS")
    expect(wrapper.text()).toContain("http://nas.local:8081")
    expect(wrapper.text()).toContain("settings.serverConnections.online")
    expect(wrapper.findAll('[data-saved-server]')).toHaveLength(2)
    expect(wrapper.findAll('[data-saved-server]')[0]!.find('button').exists()).toBe(false)
    await wrapper.findAll('[data-saved-server]')[1]!.get('button').trigger('click')
    expect(api.openServerConnections).toHaveBeenCalledWith("office")
  })
  it("opens local management to add servers and refreshes when focus returns", async () => {
    const api = bridge()
    const wrapper = render()
    await flushPromises()
    const add = wrapper.findAll('button').find(button => button.text().includes('serverConnections.add'))!
    await add.trigger('click')
    await flushPromises()
    expect(api.openServerConnections).toHaveBeenCalledWith(undefined)
    api.getServerConnections.mockResolvedValue({ ...state, servers: [...state.servers, { id: "new", name: "New server", url: "https://new.example.com" }] })
    window.dispatchEvent(new Event('focus'))
    await flushPromises()
    expect(wrapper.text()).toContain("New server")
  })
  it("reports list errors and retries without presenting a successful empty list", async () => {
    const api = bridge()
    api.getServerConnections.mockRejectedValueOnce(new Error("IPC unavailable"))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('serverConnections.loadError')
    expect(wrapper.text()).not.toContain('serverConnections.empty')
    await wrapper.findAll('button').find(button => button.text().includes('serverConnections.refresh'))!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('家庭 NAS')
    expect(wrapper.text()).not.toContain('serverConnections.loadError')
    expect(health.checkNow).toHaveBeenCalledOnce()
  })
  it("shows offline and empty states honestly", async () => {
    const api = bridge()
    health.status = "offline"
    api.getServerConnections.mockResolvedValue({ ...state, servers: [] })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('nav.backendOffline')
    expect(wrapper.text()).toContain('serverConnections.empty')
    expect(wrapper.text()).toContain(state.currentServerUrl)
  })
  it("prevents repeated switching and displays failures", async () => {
    const api = bridge()
    let rejectSwitch: (reason: Error) => void = () => {}
    api.openServerConnections.mockImplementation(() => new Promise((_resolve, reject) => { rejectSwitch = reject }))
    const wrapper = render()
    await flushPromises()
    const connect = wrapper.findAll('[data-saved-server]')[1]!.get('button')
    await connect.trigger('click')
    expect(connect.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('serverConnections.switching')
    rejectSwitch(new Error('offline'))
    await flushPromises()
    expect(wrapper.text()).toContain('serverConnections.actionError')
    expect(wrapper.text()).toContain('家庭 NAS')
  })
  it("cleans up polling and focus listeners on unmount", async () => {
    vi.useFakeTimers()
    const api = bridge()
    const wrapper = render()
    await flushPromises()
    wrapper.unmount()
    const calls = api.getServerConnections.mock.calls.length
    window.dispatchEvent(new Event('focus'))
    await vi.advanceTimersByTimeAsync(10000)
    expect(api.getServerConnections).toHaveBeenCalledTimes(calls)
  })
})
