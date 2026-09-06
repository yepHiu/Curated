import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import { defaultAIGovernance, type AIReport } from "@/services/contracts/ai-governance-service"
import SettingsAISection from "./SettingsAISection.vue"
import { applyAIGovernance, useExperimentalAgent } from "@/lib/experimental-agent"

const mocks = vi.hoisted(() => ({ getSettings: vi.fn(), saveSettings: vi.fn(), getUsage: vi.fn(), getAudit: vi.fn(), cleanup: vi.fn(), setAIProvider: vi.fn(), testAIProvider: vi.fn() }))
vi.mock("@/services/ai-governance-service", () => ({ useAIGovernanceService: () => mocks }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({ aiProvider: { value: { kind: "openai-compatible", baseUrl: "", model: "" } }, setAIProvider: mocks.setAIProvider, testAIProvider: mocks.testAIProvider }) }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ locale: ref("en"), t: (key: string) => key }) }))

function emptyReport(): AIReport {
  return { items: [], total: 0, limit: 5, offset: 0, summary: { runs: 0, failed: 0, partial: 0, cancelled: 0, modelCalls: 0, usageCalls: 0, toolCalls: 0, promptTokens: 0, completionTokens: 0, totalTokens: 0, avgDurationMs: null, avgFirstTextMs: null } }
}
async function setup() { const wrapper = mount(SettingsAISection, { props: { useWebApi: true } }); await flushPromises(); return wrapper }
beforeEach(() => {
  vi.clearAllMocks()
  applyAIGovernance(defaultAIGovernance())
  mocks.getSettings.mockResolvedValue(defaultAIGovernance())
  mocks.saveSettings.mockImplementation(async (value) => value)
  mocks.getUsage.mockResolvedValue(emptyReport())
  mocks.getAudit.mockResolvedValue({ items: [], total: 0, limit: 10, offset: 0 })
  mocks.cleanup.mockResolvedValue({ runs: 2, audit: 1, receipts: 0 })
  mocks.setAIProvider.mockResolvedValue(undefined)
  mocks.testAIProvider.mockResolvedValue({ ok: true, latencyMs: 10 })
})
describe("SettingsAISection", () => {
  it("keeps provider configuration available while AI is disabled and shows unknown usage", async () => {
    const wrapper = await setup()
    expect(wrapper.find("#ai-base").exists()).toBe(true)
    expect(wrapper.get("[data-ai-token-total]").text()).toBe("aiSettings.unknown")
    expect(useExperimentalAgent().enabled.value).toBe(false)
    wrapper.unmount()
  })
  it("organizes governance, provider, and records into the settings card hierarchy", async () => {
    const wrapper = await setup()
    expect(wrapper.find("[data-ai-governance-card]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-provider-card]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-statistics-card]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-settings-block=policy]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-settings-block=limits]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-settings-block=filters]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-settings-block=recent-runs]").exists()).toBe(true)
    expect(wrapper.find("[data-ai-settings-block=audit]").exists()).toBe(true)
    wrapper.unmount()
  })
  it("loads five recent requests and ten audit records per page", async () => {
    const wrapper = await setup()
    expect(mocks.getUsage).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 5, offset: 0 }))
    expect(mocks.getAudit).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 10, offset: 0 }))
    wrapper.unmount()
  })
  it("compacts request and audit records to one line with semantic status colors", async () => {
    const report = emptyReport()
    report.items = [{ id: "run-success", startedAt: "2026-09-06T00:00:00Z", channel: "chat", action: "", sessionId: "", provider: "test", model: "model", promptVersion: "v1", status: "completed", errorCode: "", durationMs: 12, firstTextMs: 4, modelCalls: 1, usageCalls: 1, toolCalls: 0, promptTokens: 2, completionTokens: 3, totalTokens: 5 }, { id: "run-failure", startedAt: "2026-09-06T00:00:00Z", channel: "chat", action: "", sessionId: "", provider: "test", model: "model", promptVersion: "v1", status: "failed", errorCode: "AI_FAILED", durationMs: 12, firstTextMs: null, modelCalls: 1, usageCalls: 0, toolCalls: 0, promptTokens: 0, completionTokens: 0, totalTokens: 0 }]
    mocks.getUsage.mockResolvedValue(report)
    mocks.getAudit.mockResolvedValue({ items: [{ id: "audit-success", createdAt: "2026-09-06T00:00:00Z", channel: "chat", sessionId: "", tool: "search_movies", permission: "read", result: "ok", errorCode: "", durationMs: 1 }, { id: "audit-failure", createdAt: "2026-09-06T00:00:00Z", channel: "chat", sessionId: "", tool: "search_movies", permission: "read", result: "error", errorCode: "AI_TOOL_FAILED", durationMs: 1 }], total: 2, limit: 25, offset: 0 })
    const wrapper = await setup()
    expect(wrapper.get('[data-ai-run-row="run-success"]').classes()).toContain("min-h-9")
    expect(wrapper.get('[data-ai-run-row="run-success"] [data-slot="badge"]').classes()).toContain("text-success")
    expect(wrapper.get('[data-ai-run-row="run-failure"] [data-slot="badge"]').classes()).toContain("text-danger")
    expect(wrapper.get('[data-ai-audit-row="audit-success"] [data-slot="badge"]').classes()).toContain("text-success")
    expect(wrapper.get('[data-ai-audit-row="audit-failure"] [data-slot="badge"]').classes()).toContain("text-danger")
    wrapper.unmount()
  })
  it("applies global state only after saving and keeps the old state on failure", async () => {
    const wrapper = await setup()
    await wrapper.get("[data-ai-enabled]").trigger("click")
    expect(useExperimentalAgent().enabled.value).toBe(false)
    mocks.saveSettings.mockRejectedValueOnce(new Error("offline"))
    await wrapper.get("[data-ai-save]").trigger("click"); await flushPromises()
    expect(useExperimentalAgent().enabled.value).toBe(false)
    expect(wrapper.get("[role=alert]").text()).toContain("offline")
    await wrapper.get("[data-ai-save]").trigger("click"); await flushPromises()
    expect(mocks.saveSettings).toHaveBeenLastCalledWith({ ...defaultAIGovernance(), enabled: true })
    expect(useExperimentalAgent().enabled.value).toBe(true)
    wrapper.unmount()
  })
  it("marks partially reported usage and retries failed statistics loading", async () => {
    const report = emptyReport(); report.summary.modelCalls = 2; report.summary.usageCalls = 1; report.summary.totalTokens = 30
    mocks.getUsage.mockResolvedValue(report)
    const wrapper = await setup()
    expect(wrapper.get("[data-ai-token-total]").text()).toBe("30 (aiSettings.partialUsage)")
    mocks.getUsage.mockRejectedValueOnce(new Error("query failed"))
    await wrapper.get("[data-ai-refresh]").trigger("click"); await flushPromises()
    expect(wrapper.find("[data-ai-summary]").exists()).toBe(false)
    expect(wrapper.get("[role=alert]").text()).toContain("query failed")
    await wrapper.get("[data-ai-refresh]").trigger("click"); await flushPromises()
    expect(wrapper.find("[data-ai-summary]").exists()).toBe(true)
    wrapper.unmount()
  })
  it("rejects invalid policy locally and only cleans expired records on explicit click", async () => {
    const wrapper = await setup()
    expect(mocks.cleanup).not.toHaveBeenCalled()
    await wrapper.get("#ai-retention").setValue("1")
    expect(wrapper.get("[data-ai-save]").attributes("disabled")).toBeDefined()
    await wrapper.get("[data-ai-cleanup]").trigger("click"); await flushPromises()
    expect(mocks.cleanup).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
  it("preserves the provider failure after refreshing its recorded statistics", async () => {
    const wrapper = await setup()
    mocks.testAIProvider.mockResolvedValueOnce({ ok: false, message: "authentication failed" })
    await wrapper.get("[data-ai-provider-test]").trigger("click"); await flushPromises()
    const result = wrapper.get("[data-ai-provider-test-result]")
    expect(result.text()).toBe("settings.experimentalTestFail")
    expect(result.attributes("data-status")).toBe("failed")
    expect(wrapper.find("[data-ai-settings-error]").exists()).toBe(false)
    expect(mocks.getUsage).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })
  it("renders a successful connectivity test inside the model provider section", async () => {
    const wrapper = await setup()
    await wrapper.get("[data-ai-provider-test]").trigger("click"); await flushPromises()
    const provider = wrapper.get("[data-ai-provider-card]")
    const result = provider.get("[data-ai-provider-test-result]")
    expect(result.text()).toBe("settings.experimentalTestOk")
    expect(result.attributes("data-status")).toBe("success")
    expect(result.classes()).toContain("text-success")
    wrapper.unmount()
  })
})
