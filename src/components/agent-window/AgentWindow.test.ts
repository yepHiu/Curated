import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import AgentWindow from "./AgentWindow.vue"
import { AIServiceError } from "@/services/contracts/ai-service"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAgentWindow } from "@/composables/use-agent-window"

const streamChatMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  RouterLink: {
    name: "RouterLink",
    props: ["to"],
    template: "<a :href=\"typeof to === 'string' ? to : '#'\"><slot /></a>",
  },
}))

vi.mock("@/services/ai-service", () => ({
  useAIService: () => ({ streamChat: streamChatMock }),
}))

function mountWindow() {
  return mount(AgentWindow, {
    global: {
      stubs: {
        Teleport: true,
      },
    },
  })
}

describe("AgentWindow", () => {
  beforeEach(() => {
    localStorage.clear()
    streamChatMock.mockReset()
    const { setEnabled } = useExperimentalAgent()
    setEnabled(true)
    useAgentWindow().openWindow()
  })

  afterEach(() => {
    useAgentWindow().closeWindow()
    const { setEnabled } = useExperimentalAgent()
    setEnabled(false)
  })

  it("renders the dialog when open", () => {
    const wrapper = mountWindow()
    expect(wrapper.find("[data-agent-window]").exists()).toBe(true)
    expect(wrapper.find("[data-agent-window-input]").exists()).toBe(true)
  })

  it("does not render when closed", () => {
    useAgentWindow().closeWindow()
    const wrapper = mountWindow()
    expect(wrapper.find("[data-agent-window]").exists()).toBe(false)
  })

  it("streams an assistant reply for a sent message", async () => {
    streamChatMock.mockImplementation(
      async (_messages: unknown, handlers: { onDelta: (d: string) => void }) => {
        handlers.onDelta("你好")
        handlers.onDelta("，世界")
      },
    )
    const wrapper = mountWindow()

    await wrapper.find("[data-agent-window-input]").setValue("打个招呼")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    const entries = wrapper.findAll("[data-agent-entry]")
    expect(entries.length).toBe(2)
    expect(entries[0]!.attributes("data-agent-entry")).toBe("user")
    expect(entries[0]!.text()).toContain("打个招呼")
    expect(entries[1]!.attributes("data-agent-entry")).toBe("assistant")
    expect(entries[1]!.text()).toContain("你好，世界")
  })

  it("shows setup guidance when the provider is unconfigured", async () => {
    streamChatMock.mockRejectedValue(
      new AIServiceError("provider baseUrl and model are required", "AI_PROVIDER_UNAVAILABLE"),
    )
    const wrapper = mountWindow()

    await wrapper.find("[data-agent-window-input]").setValue("hi")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    expect(wrapper.find("[data-agent-window-unconfigured]").exists()).toBe(true)
    expect(wrapper.find("[data-agent-window-error]").exists()).toBe(false)
  })

  it("shows an error message for other failures", async () => {
    streamChatMock.mockRejectedValue(new AIServiceError("HTTP 500", "AI_CHAT_FAILED"))
    const wrapper = mountWindow()

    await wrapper.find("[data-agent-window-input]").setValue("hi")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    const error = wrapper.find("[data-agent-window-error]")
    expect(error.exists()).toBe(true)
    expect(error.text()).toContain("HTTP 500")
    expect(wrapper.find("[data-agent-window-unconfigured]").exists()).toBe(false)
  })

  it("closes on Escape", async () => {
    const wrapper = mountWindow()
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
    await flushPromises()
    expect(wrapper.find("[data-agent-window]").exists()).toBe(false)
  })
})
