import { describe, expect, it, vi } from "vitest"
const state = vi.hoisted(() => ({ locale: { value: "zh-CN" }, post: vi.fn().mockResolvedValue({}), get: vi.fn().mockResolvedValue({ total: 3, organized: 1, unorganized: 1, outdated: 1, unresolved: 0 }) }))
vi.mock("@/i18n", () => ({ i18n: { global: { locale: state.locale } } }))
vi.mock("@/api/http-client", () => ({ httpClient: { post: state.post, get: state.get } }))
import { webTagOrganization } from "./web-topic-service"

describe("topic label language", () => {
  it("reads server coverage and forwards the selected remaining scope", async () => {
    expect((await webTagOrganization.getTagOrganizationStats()).outdated).toBe(1)
    expect(state.get).toHaveBeenCalledWith("/ai/tag-organizations/stats")
    await webTagOrganization.startTagOrganization("unorganized")
    expect(state.post).toHaveBeenLastCalledWith("/ai/tag-organizations", expect.objectContaining({ scope: "unorganized" }))
  })
  it.each(["zh-CN", "en", "ja"])("captures %s at explicit task creation", async (locale) => {
    state.locale.value = locale
    await webTagOrganization.startTagOrganization("selected", ["synthetic"])
    expect(state.post).toHaveBeenLastCalledWith("/ai/tag-organizations", expect.objectContaining({ locale, scope: "selected", movieIds: ["synthetic"] }))
  })
})
