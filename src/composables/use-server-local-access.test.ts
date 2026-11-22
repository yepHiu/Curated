import { defineComponent, h } from "vue"
import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import { api } from "@/api/endpoints"
import { useServerLocalAccess } from "./use-server-local-access"

vi.mock("@/api/endpoints", () => {
  // 测试不得访问开发机的实际 Server。
  return { api: { health: vi.fn() } }
})

const AccessProbe = defineComponent({
  /** 使用真实生命周期触发权限检查，观察最终可管理状态。 */
  setup() {
    const access = useServerLocalAccess()
    // 只投影权限状态，隔离设置页的展示结构。
    return () => h("output", String(access.isServerLocal.value))
  },
})

afterEach(() => {
  // 恢复配置的默认环境，不让 Web 个例污染后续 Mock 测试。
  vi.unstubAllEnvs()
  vi.clearAllMocks()
})

describe("server local access test isolation", () => {
  // 即使外部 shell 指向真实 Server，默认单元测试也不应请求 health。
  it("defaults to local Mock access without contacting a Server", async () => {
    const wrapper = mount(AccessProbe)
    try {
      await flushPromises()
      expect(wrapper.text()).toBe("true")
      expect(api.health).not.toHaveBeenCalled()
    } finally { wrapper.unmount() }
  })

  // Web 权限测试仍可显式启用真实分支，并完全控制 Server 响应。
  it("allows explicit Web-mode tests with a mocked health boundary", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    vi.mocked(api.health).mockResolvedValue({ canManageLibraryPaths: true } as Awaited<ReturnType<typeof api.health>>)
    const wrapper = mount(AccessProbe)
    try {
      expect(wrapper.text()).toBe("false")
      await flushPromises()
      expect(api.health).toHaveBeenCalledTimes(1)
      expect(wrapper.text()).toBe("true")
    } finally { wrapper.unmount() }
  })
})
