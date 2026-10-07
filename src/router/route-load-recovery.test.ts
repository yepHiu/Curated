import { beforeEach, describe, expect, it, vi } from "vitest"
import { createMemoryHistory, createRouter } from "vue-router"
import { dismissAppToast, pushAppToast } from "@/composables/use-app-toast"
import { installRouteLoadRecovery, reloadRouteDocument } from "./route-load-recovery"

vi.mock("@/i18n", () => ({ i18n: { global: { t: (key: string) => key } } }))
vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
  dismissAppToast: vi.fn(),
}))

beforeEach(() => vi.clearAllMocks())

describe("route load recovery", () => {
  it.each(["wishlist", "curated-frames"])("makes a failed %s import actionable while preserving the current page", async (page) => {
    const error = new TypeError("Failed to fetch dynamically imported module")
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: "/", component: { template: "<div>Home</div>" } },
        { path: `/${page}`, component: () => Promise.reject(error) },
      ],
    })
    installRouteLoadRecovery(router)
    await router.push("/")
    vi.clearAllMocks()

    await expect(router.push(`/${page}?q=actor`)).rejects.toThrow(error)

    expect(router.currentRoute.value.fullPath).toBe("/")
    expect(pushAppToast).toHaveBeenCalledWith("app.routeLoadFailed", expect.objectContaining({
      id: "route-load-failed",
      variant: "destructive",
      durationMs: 9000,
      action: { label: "app.reload", onClick: expect.any(Function) },
    }))
    expect(dismissAppToast).not.toHaveBeenCalled()

    router.addRoute({ path: "/other", component: { template: "<div>Other</div>" } })
    await router.push("/other")
    expect(dismissAppToast).toHaveBeenCalledWith("route-load-failed")
  })

  it("reloads the entry document at the intended hash route, including query parameters", () => {
    const location = {
      href: "http://192.168.1.10:8081/#/library",
      replace: vi.fn(),
      reload: vi.fn(),
    }
    reloadRouteDocument("#/wishlist?status=all&q=actor", location)
    expect(location.replace).toHaveBeenCalledWith("http://192.168.1.10:8081/#/wishlist?status=all&q=actor")
    expect(location.reload).toHaveBeenCalledOnce()
    expect(location.replace.mock.invocationCallOrder[0]).toBeLessThan(location.reload.mock.invocationCallOrder[0]!)
  })
})
