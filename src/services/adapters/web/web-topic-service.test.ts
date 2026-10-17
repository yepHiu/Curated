import { afterEach, describe, expect, it, vi } from "vitest"
afterEach(() => vi.unstubAllGlobals())
const state = vi.hoisted(() => ({ locale: { value: "zh-CN" }, delete: vi.fn().mockResolvedValue(undefined), post: vi.fn().mockResolvedValue({}), get: vi.fn().mockResolvedValue({ total: 3, organized: 1, unorganized: 1, outdated: 1, needsAttention: 0, unresolved: 0 }) }))
vi.mock("@/i18n", () => ({ i18n: { global: { locale: state.locale } } }))
vi.mock("@/api/http-client", () => ({ httpClient: { delete: state.delete, post: state.post, get: state.get } }))
import { webTagOrganization } from "./web-topic-service"

describe("topic label language", () => {
  it("submits organization with a UUID on LAN HTTP without randomUUID", async () => {
    const source = globalThis.crypto
    vi.stubGlobal("crypto", { getRandomValues: source.getRandomValues.bind(source) })
    await webTagOrganization.startTagOrganization("selected", ["synthetic"], ["title"])
    expect(state.post).toHaveBeenLastCalledWith("/ai/tag-organizations", expect.objectContaining({
      scope: "selected", movieIds: ["synthetic"], tasks: ["title"],
      requestId: expect.stringMatching(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/),
    }))
  })
  it("paginates issue records without starting a model task", async () => {
    state.get.mockResolvedValueOnce({ items: [{ movieId: 'problem', reason: 'SOURCE_TOO_LONG' }] })
    const calls = state.post.mock.calls.length
    expect((await webTagOrganization.getTagOrganizationIssues(25))[0]?.movieId).toBe('problem')
    expect(state.get).toHaveBeenLastCalledWith('/ai/tag-organizations/issues?limit=25&offset=25')
    expect(state.post.mock.calls.length).toBe(calls)
  })
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


it("deletes the encoded history ID", async () => {
 await webTagOrganization.deleteTagOrganization("job/one")
 expect(state.delete).toHaveBeenCalledWith("/ai/tag-organizations/job%2Fone")
})

it("forwards the chosen actions without implicitly enabling tags", async () => {
 await webTagOrganization.startTagOrganization("selected", ["synthetic"], ["title", "summary"])
 expect(state.post).toHaveBeenLastCalledWith("/ai/tag-organizations", expect.objectContaining({ tasks: ["title", "summary"] }))
})
