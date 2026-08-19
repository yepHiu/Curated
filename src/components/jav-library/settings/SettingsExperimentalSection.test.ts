import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

import SettingsExperimentalSection from "./SettingsExperimentalSection.vue"
import { useExperimentalAgent } from "@/lib/experimental-agent"

const mocks = vi.hoisted(() => ({
  provider: {
    kind: "openai-compatible",
    baseUrl: "",
    apiKey: "",
    model: "",
  },
  setAIProvider: vi.fn(),
  testAIProvider: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    aiProvider: {
      get value() {
        return mocks.provider
      },
    },
    setAIProvider: mocks.setAIProvider,
    testAIProvider: mocks.testAIProvider,
  }),
}))

function mountSection(useWebApi = true) {
  return mount(SettingsExperimentalSection, { props: { useWebApi } })
}

async function enableAgent(wrapper: ReturnType<typeof mountSection>) {
  await wrapper.find("[data-experimental-agent-toggle]").trigger("click")
}

describe("SettingsExperimentalSection", () => {
  beforeEach(() => {
    localStorage.clear()
    mocks.provider = { kind: "openai-compatible", baseUrl: "", apiKey: "", model: "" }
    mocks.setAIProvider.mockReset().mockResolvedValue(undefined)
    mocks.testAIProvider.mockReset().mockResolvedValue({ ok: true, latencyMs: 42 })
    useExperimentalAgent().setEnabled(false)
  })

  it("defaults the agent toggle to off and hides the provider form", () => {
    const wrapper = mountSection()
    expect(wrapper.find("[data-experimental-agent-toggle]").exists()).toBe(true)
    expect(wrapper.find("[data-experimental-base-url]").exists()).toBe(false)
  })

  it("shows the provider form and persists the toggle when enabled", async () => {
    const wrapper = mountSection()
    await enableAgent(wrapper)

    expect(wrapper.find("[data-experimental-base-url]").exists()).toBe(true)
    expect(localStorage.getItem("curated-agent-experimental-v1")).toBe("true")
  })

  it("saves trimmed provider drafts through the service", async () => {
    const wrapper = mountSection()
    await enableAgent(wrapper)

    await wrapper.find("[data-experimental-base-url]").setValue("  http://127.0.0.1:11434/v1  ")
    await wrapper.find("[data-experimental-api-key]").setValue("secret")
    await wrapper.find("[data-experimental-model]").setValue("  qwen3  ")
    await wrapper.find("[data-experimental-save]").trigger("click")
    await flushPromises()

    expect(mocks.setAIProvider).toHaveBeenCalledWith({
      baseUrl: "http://127.0.0.1:11434/v1",
      apiKey: "secret",
      model: "qwen3",
    })
    expect(wrapper.find("[data-experimental-status]").text()).toContain("experimentalSaved")
  })

  it("tests the draft provider without saving", async () => {
    const wrapper = mountSection()
    await enableAgent(wrapper)

    await wrapper.find("[data-experimental-base-url]").setValue("http://127.0.0.1:11434/v1")
    await wrapper.find("[data-experimental-model]").setValue("qwen3")
    await wrapper.find("[data-experimental-test]").trigger("click")
    await flushPromises()

    expect(mocks.testAIProvider).toHaveBeenCalledWith(
      expect.objectContaining({
        baseUrl: "http://127.0.0.1:11434/v1",
        model: "qwen3",
      }),
    )
    expect(mocks.setAIProvider).not.toHaveBeenCalled()
    expect(wrapper.find("[data-experimental-status]").text()).toContain("experimentalTestOk")
  })

  it("initializes drafts from the service state", async () => {
    mocks.provider = {
      kind: "openai-compatible",
      baseUrl: "http://localhost:1234/v1",
      apiKey: "k",
      model: "m1",
    }
    const wrapper = mountSection()
    await enableAgent(wrapper)

    expect((wrapper.find("[data-experimental-base-url]").element as HTMLInputElement).value).toBe(
      "http://localhost:1234/v1",
    )
    expect((wrapper.find("[data-experimental-model]").element as HTMLInputElement).value).toBe("m1")
  })
})
