import { describe, expect, it, vi } from "vitest"
const state = vi.hoisted(() => ({ locale: { value: "zh-CN" }, post: vi.fn().mockResolvedValue({}) }))
vi.mock("@/i18n", () => ({ i18n: { global: { locale: state.locale } } }))
vi.mock("@/api/http-client", () => ({ httpClient: { post: state.post } }))
import { webTagOrganization } from "./web-topic-service"

describe("topic label language", () => {
  it.each(["zh-CN", "en", "ja"])("captures %s at explicit task creation", async (locale) => {
    state.locale.value = locale
    await webTagOrganization.startTagOrganization("selected", ["synthetic"])
    expect(state.post).toHaveBeenLastCalledWith("/ai/tag-organizations", expect.objectContaining({ locale, scope: "selected", movieIds: ["synthetic"] }))
  })
})
