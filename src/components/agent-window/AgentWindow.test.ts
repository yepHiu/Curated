import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import AgentWindow from "./AgentWindow.vue"
import { AIServiceError } from "@/services/contracts/ai-service"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAgentWindow } from "@/composables/use-agent-window"

const streamChatMock = vi.hoisted(() => vi.fn())
const listSessionsMock = vi.hoisted(() => vi.fn())
const createSessionMock = vi.hoisted(() => vi.fn())
const getSessionMock = vi.hoisted(() => vi.fn())
const deleteSessionMock = vi.hoisted(() => vi.fn())
const pushMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => ({ name: "library", params: {}, query: {} }),
  useRouter: () => ({ push: pushMock }),
  RouterLink: {
    name: "RouterLink",
    props: ["to"],
    template: "<a :href=\"typeof to === 'string' ? to : '#'\"><slot /></a>",
  },
}))

const confirmToolMock = vi.hoisted(() => vi.fn())

vi.mock("@/services/ai-service", () => ({
  useAIService: () => ({
    streamChat: streamChatMock,
    listSessions: listSessionsMock,
    createSession: createSessionMock,
    getSession: getSessionMock,
    deleteSession: deleteSessionMock,
    runAction: vi.fn(),
    confirmTool: confirmToolMock,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: {
      value: [
        {
          id: "m1",
          title: "Hello",
          code: "ABC-001",
          actors: ["Ada"],
          tags: ["轻松"],
          userTags: [],
        },
      ],
    },
    listActors: async () => ({ actors: [{ name: "Ada", movieCount: 1 }], total: 1 }),
  }),
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

async function openMentionPicker(wrapper: ReturnType<typeof mountWindow>) {
  const input = wrapper.find("[data-agent-window-input]")
  await input.setValue("@")
  await flushPromises()
}

describe("AgentWindow", () => {
  beforeEach(() => {
    localStorage.clear()
    streamChatMock.mockReset()
    listSessionsMock.mockReset()
    createSessionMock.mockReset()
    getSessionMock.mockReset()
    deleteSessionMock.mockReset()
    pushMock.mockReset()
    listSessionsMock.mockResolvedValue([])
    createSessionMock.mockResolvedValue({ id: "ses_new", title: "", createdAt: "", updatedAt: "" })
    const { setEnabled } = useExperimentalAgent()
    setEnabled(true)
    const windowState = useAgentWindow()
    windowState.moveTo(40, 40)
    windowState.resizeTo(640, 560)
    windowState.setSidebarOpen(true)
    windowState.openWindow()
    getSessionMock.mockResolvedValue({
      id: "ses_1",
      title: "",
      createdAt: "",
      updatedAt: "",
      messages: [],
    })
    deleteSessionMock.mockResolvedValue(undefined)
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
      async (
        _input: unknown,
        handlers: { onDelta: (d: string) => void; onSession?: (id: string) => void },
      ) => {
        handlers.onSession?.("ses_1")
        handlers.onDelta("你好")
        handlers.onDelta("，世界")
      },
    )
    const wrapper = mountWindow()
    await flushPromises()

    await wrapper.find("[data-agent-window-input]").setValue("打个招呼")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    const entries = wrapper.findAll("[data-agent-entry]")
    expect(entries.length).toBe(2)
    expect(entries[0]!.attributes("data-agent-entry")).toBe("user")
    expect(entries[0]!.text()).toContain("打个招呼")
    expect(entries[0]!.find("[data-agent-user-bubble]").exists()).toBe(true)
    expect(entries[1]!.attributes("data-agent-entry")).toBe("assistant")
    expect(entries[1]!.text()).toContain("你好，世界")
  })

  it("renders markdown in an assistant reply", async () => {
    streamChatMock.mockImplementation(
      async (
        _input: unknown,
        handlers: { onDelta: (d: string) => void },
      ) => {
        handlers.onDelta("**粗体** 和 `code`")
      },
    )
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("格式化")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    const assistant = wrapper.find('[data-agent-entry="assistant"]')
    expect(assistant.find("strong").text()).toBe("粗体")
    expect(assistant.find("code").text()).toBe("code")
  })

  it("folds lookup into a productized process bar", async () => {
    streamChatMock.mockImplementation(
      async (
        _input: unknown,
        handlers: {
          onDelta: (d: string) => void
          onThinking?: (d: string) => void
          onToolStart?: (e: { toolCallId: string; name: string }) => void
          onToolResult?: (e: { toolCallId: string; name: string; ok: boolean; summary: string }) => void
        },
      ) => {
        handlers.onThinking?.("先找未看")
        handlers.onToolStart?.({ toolCallId: "call_1", name: "search_movies" })
        handlers.onToolResult?.({
          toolCallId: "call_1",
          name: "search_movies",
          ok: true,
          summary: "total: 5",
        })
        handlers.onToolStart?.({ toolCallId: "call_2", name: "present_movies" })
        handlers.onToolResult?.({
          toolCallId: "call_2",
          name: "present_movies",
          ok: true,
          summary: "1 movies",
        })
        handlers.onDelta("有 5 部未看")
      },
    )
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("今晚看什么")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    const process = wrapper.find("[data-agent-process]")
    expect(process.exists()).toBe(true)
    expect(process.text()).toContain("agentWindow.toolOk")
    expect(process.text()).not.toContain("search_movies")
    expect(process.text()).not.toContain("total: 5")
    expect(wrapper.find("[data-agent-process-tools]").exists()).toBe(false)

    await wrapper.find("[data-agent-process-toggle]").trigger("click")
    await flushPromises()
    expect(wrapper.find("[data-agent-process-tools]").text()).toContain("agentWindow.tools.searchMovies")
    expect(wrapper.find("[data-agent-process-tools]").text()).not.toContain("present_movies")
    expect(wrapper.find("[data-agent-thinking]").text()).toContain("先找未看")
  })

  it("renders movie cards from a present_movies slate and opens detail", async () => {
    streamChatMock.mockImplementation(
      async (
        _input: unknown,
        handlers: {
          onDelta: (d: string) => void
          onToolStart?: (e: { toolCallId: string; name: string }) => void
          onToolResult?: (e: { toolCallId: string; name: string; ok: boolean; summary: string }) => void
          onMovieCards?: (movies: { movieId: string; title: string; reason: string }[]) => void
        },
      ) => {
        handlers.onToolStart?.({ toolCallId: "call_1", name: "search_movies" })
        handlers.onToolResult?.({
          toolCallId: "call_1",
          name: "search_movies",
          ok: true,
          summary: "total: 1",
        })
        handlers.onMovieCards?.([{ movieId: "m1", title: "Hello", reason: "轻松短片" }])
        handlers.onDelta("今晚看这部")
      },
    )
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("今晚看什么")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    const card = wrapper.find('[data-agent-movie-card="m1"]')
    expect(card.exists()).toBe(true)
    expect(card.text()).toContain("Hello")
    expect(card.text()).toContain("轻松短片")
    await card.trigger("click")
    expect(pushMock).toHaveBeenCalledWith({ name: "detail", params: { id: "m1" } })
    expect(wrapper.find("[data-agent-window]").exists()).toBe(true)
  })

  it("shows setup guidance when the provider is unconfigured", async () => {
    streamChatMock.mockRejectedValue(
      new AIServiceError("provider baseUrl and model are required", "AI_PROVIDER_UNAVAILABLE"),
    )
    const wrapper = mountWindow()
    await flushPromises()

    await wrapper.find("[data-agent-window-input]").setValue("hi")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    expect(wrapper.find("[data-agent-window-unconfigured]").exists()).toBe(true)
    expect(wrapper.find("[data-agent-window-error]").exists()).toBe(false)
  })

  it("shows an error message for other failures", async () => {
    streamChatMock.mockRejectedValue(new AIServiceError("HTTP 500", "AI_CHAT_FAILED"))
    const wrapper = mountWindow()
    await flushPromises()

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

  it("renders desktop resize handles", () => {
    const wrapper = mountWindow()
    expect(wrapper.find('[data-agent-window-resize="se"]').exists()).toBe(true)
    expect(wrapper.find('[data-agent-window-resize="e"]').exists()).toBe(true)
    expect(wrapper.find('[data-agent-window-resize="s"]').exists()).toBe(true)
  })

  it("resizes from the bottom-right handle", async () => {
    const { moveTo, resizeTo } = useAgentWindow()
    moveTo(40, 40)
    resizeTo(420, 560)
    const wrapper = mountWindow()
    const handle = wrapper.find('[data-agent-window-resize="se"]')
    handle.element.dispatchEvent(
      new PointerEvent("pointerdown", { button: 0, clientX: 460, clientY: 600, bubbles: true }),
    )
    window.dispatchEvent(new PointerEvent("pointermove", { clientX: 540, clientY: 680 }))
    window.dispatchEvent(new PointerEvent("pointerup", { clientX: 540, clientY: 680 }))
    await flushPromises()

    const style = wrapper.find("[data-agent-window]").attributes("style") ?? ""
    expect(style).toContain("width: 500px")
    expect(style).toContain("height: 640px")
  })

  it("renders a history sidebar with persisted chats", async () => {
    listSessionsMock.mockResolvedValue([
      {
        id: "ses_1",
        title: "今晚看什么",
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      },
    ])
    getSessionMock.mockResolvedValue({
      id: "ses_1",
      title: "今晚看什么",
      createdAt: "",
      updatedAt: "",
      messages: [
        { id: "m1", sessionId: "ses_1", role: "user", content: "已有问题", seq: 1, createdAt: "" },
      ],
    })
    const wrapper = mountWindow()
    await flushPromises()

    expect(wrapper.find("[data-agent-window-sidebar]").exists()).toBe(true)
    expect(wrapper.find('[data-agent-window-session="ses_1"]').text()).toContain("今晚看什么")
    expect(wrapper.find('[data-agent-entry="user"]').text()).toContain("已有问题")
  })

  it("loads a selected chat from the sidebar", async () => {
    listSessionsMock.mockResolvedValue([
      { id: "ses_a", title: "A", createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      { id: "ses_b", title: "B", createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
    ])
    getSessionMock.mockImplementation(async (id: string) => ({
      id,
      title: id === "ses_b" ? "B" : "A",
      createdAt: "",
      updatedAt: "",
      messages: [
        { id: `m-${id}`, sessionId: id, role: "user", content: id === "ses_b" ? "第二问" : "第一问", seq: 1, createdAt: "" },
      ],
    }))
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find('[data-agent-window-session="ses_b"]').trigger("click")
    await flushPromises()

    expect(getSessionMock).toHaveBeenCalledWith("ses_b")
    expect(wrapper.find('[data-agent-entry="user"]').text()).toContain("第二问")
  })

  it("deletes a chat from the sidebar", async () => {
    listSessionsMock.mockResolvedValue([
      { id: "ses_1", title: "可删", createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
    ])
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find('[data-agent-window-delete="ses_1"]').trigger("click")
    await flushPromises()

    expect(deleteSessionMock).toHaveBeenCalledWith("ses_1")
    expect(wrapper.find('[data-agent-window-session="ses_1"]').exists()).toBe(false)
  })

  it("toggles the history sidebar", async () => {
    const wrapper = mountWindow()
    expect(wrapper.find("[data-agent-window-sidebar]").exists()).toBe(true)
    await wrapper.find("[data-agent-window-sidebar-toggle]").trigger("click")
    expect(wrapper.find("[data-agent-window-sidebar]").exists()).toBe(false)
  })

  it("sends remaining @ mentions in chat context", async () => {
    streamChatMock.mockImplementation(async (input: { context?: { mentions?: { kind: string; id: string }[] } }, handlers: { onDelta: (d: string) => void }) => {
      expect(input.context?.mentions).toEqual([{ kind: "movie", id: "m1", label: "Hello" }])
      handlers.onDelta("收到")
    })
    const wrapper = mountWindow()
    await flushPromises()
    await openMentionPicker(wrapper)
    await wrapper.find('[data-agent-mention-item="movie:m1"]').trigger("mousedown")
    await flushPromises()
    expect((wrapper.find("[data-agent-window-input]").element as HTMLTextAreaElement).value).toContain("@Hello")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()
    expect(streamChatMock).toHaveBeenCalled()
  })

  it("keeps the window open when Escape closes the mention picker", async () => {
    const wrapper = mountWindow()
    await flushPromises()
    await openMentionPicker(wrapper)
    expect(wrapper.find("[data-agent-mention-picker]").exists()).toBe(true)

    await wrapper.find("[data-agent-window-input]").trigger("keydown", { key: "Escape" })
    await flushPromises()
    expect(wrapper.find("[data-agent-mention-picker]").exists()).toBe(false)
    expect(wrapper.find("[data-agent-window]").exists()).toBe(true)
  })

  it("adds chat side padding when the window is wide", async () => {
    const wrapper = mountWindow()
    expect(wrapper.find("[data-agent-chat-wide]").attributes("data-agent-chat-wide")).toBe("false")
    useAgentWindow().resizeTo(800, 560)
    await flushPromises()
    expect(wrapper.find("[data-agent-chat-wide]").attributes("data-agent-chat-wide")).toBe("true")
  })

  it("keeps the message scroller on the window edge instead of the content column", async () => {
    useAgentWindow().resizeTo(800, 560)
    const wrapper = mountWindow()
    await flushPromises()
    const scroller = wrapper.find("[data-agent-window-messages]")
    const inner = wrapper.find("[data-agent-window-messages-inner]")
    expect(scroller.classes()).toContain("overflow-y-auto")
    expect(scroller.classes().some((name) => name.includes("max-w-"))).toBe(false)
    expect(inner.classes().some((name) => name.includes("max-w-"))).toBe(true)
  })

  it("uses theme surface tokens for the window chrome", () => {
    const wrapper = mountWindow()
    expect(wrapper.find("[data-agent-window]").classes()).toContain("bg-background")
    expect(wrapper.find("[data-agent-window-sidebar]").classes()).toContain("bg-sidebar")
    expect(wrapper.find("[data-agent-window-composer]").classes()).toContain("bg-muted/40")
  })
})
