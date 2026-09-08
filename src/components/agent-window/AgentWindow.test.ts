import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import AgentWindow from "./AgentWindow.vue"
import { AIServiceError, type AIChatStreamHandlers } from "@/services/contracts/ai-service"
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
const refreshSavedViewsMock = vi.hoisted(() => vi.fn())

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
    trashedMovies: { value: [] },
    listActors: async () => ({ actors: [{ name: "Ada", movieCount: 1 }], total: 1 }),
    refreshSavedViews: refreshSavedViewsMock,
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
    confirmToolMock.mockReset()
    refreshSavedViewsMock.mockReset()
    pushMock.mockReset()
    listSessionsMock.mockResolvedValue([])
    createSessionMock.mockResolvedValue({ id: "ses_new", title: "", createdAt: "", updatedAt: "" })
    confirmToolMock.mockResolvedValue({ ok: true, name: "create_saved_view", data: {} })
    refreshSavedViewsMock.mockResolvedValue(undefined)
    const { setEnabled } = useExperimentalAgent()
    setEnabled(true)
    const windowState = useAgentWindow()
    windowState.resizeTo(640)
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

  it("renders the complementary panel when open", () => {
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

  // 验证缓冲期间展示服务端进度，发布后显示来源记录数量。
  it("shows server progress and evidence after a checked answer", async () => {
    let handlers!: AIChatStreamHandlers
    let finish!: () => void
    // 保持请求打开，分别检查发布前与发布后的界面状态。
    streamChatMock.mockImplementation((_input: unknown, callbacks: AIChatStreamHandlers) => {
      handlers = callbacks
      // 测试显式控制模型完成时机。
      return new Promise<void>((resolve) => { finish = resolve })
    })
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("查询作品")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()
    handlers.onAnswerProgress?.()
    await flushPromises()
    expect(wrapper.find("[data-agent-process]").text()).toContain("agentWindow.thinking")
    expect(wrapper.find("[data-agent-thinking]").exists()).toBe(false)
    expect(wrapper.find("[data-agent-answer-evidence]").exists()).toBe(false)
    handlers.onDelta("本地记录：TEST-101")
    handlers.onAnswerEvidence?.({ version: 1, items: [{ refId: "r1", source: "local", tool: "search_movies", retrievedAt: "2026-09-09T00:00:00Z", fields: { code: "TEST-101" } }] })
    handlers.onOutcome?.({ status: "completed" })
    finish()
    await flushPromises()
    expect(wrapper.find("[data-agent-answer-evidence]").text()).toContain("agentWindow.answerEvidenceHint")
    expect(wrapper.text()).toContain("TEST-101")
  })

  it("keeps thinking mounted between streamed tool, reasoning and answer events", async () => {
    let handlers!: AIChatStreamHandlers
    let finish!: () => void
    streamChatMock.mockImplementation((_input: unknown, callbacks: AIChatStreamHandlers) => {
      handlers = callbacks
      return new Promise<void>((resolve) => { finish = resolve })
    })
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("查找影片")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    handlers.onThinking?.("先检索")
    await flushPromises()
    const thinkingNode = wrapper.find("[data-agent-thinking]").element
    const tool = { toolCallId: "lookup", name: "search_movies" }
    handlers.onToolStart?.(tool)
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").element).toBe(thinkingNode)
    const toolsNode = wrapper.find("[data-agent-process-tools]").element
    handlers.onToolResult?.({ ...tool, ok: true })
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").element).toBe(thinkingNode)
    expect(wrapper.find("[data-agent-process-tools]").element).toBe(toolsNode)
    handlers.onDelta("找到一些影片")
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").element).toBe(thinkingNode)
    handlers.onThinking?.("，再筛选")
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").element).toBe(thinkingNode)
    expect(wrapper.find("[data-agent-process-tools]").element).toBe(toolsNode)

    // A manual collapse must survive further reasoning and answer chunks.
    await wrapper.find("[data-agent-process-toggle]").trigger("click")
    expect(wrapper.find("[data-agent-thinking]").exists()).toBe(false)
    handlers.onThinking?.("，比较结果")
    handlers.onDelta("，推荐如下")
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").exists()).toBe(false)
    await wrapper.find("[data-agent-process-toggle]").trigger("click")
    const reopenedNode = wrapper.find("[data-agent-thinking]").element
    handlers.onDelta("。")
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").element).toBe(reopenedNode)

    finish()
    await flushPromises()
    expect(wrapper.find("[data-agent-thinking]").exists()).toBe(false)
    await wrapper.find("[data-agent-process-toggle]").trigger("click")
    expect(wrapper.find("[data-agent-thinking]").text()).toBe("先检索，再筛选，比较结果")
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

  it("resizes the docked panel from its left divider and stops on cancellation", async () => {
    const state = useAgentWindow()
    state.resizeTo(420)
    const wrapper = mountWindow()
    const handle = wrapper.find('[data-agent-window-resize="w"]')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-agent-window-resize]')).toHaveLength(1)
    handle.element.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 600, bubbles: true }))
    window.dispatchEvent(new PointerEvent("pointermove", { clientX: 520 }))
    expect(state.width.value).toBe(500)
    window.dispatchEvent(new PointerEvent("pointermove", { clientX: 1000 }))
    expect(state.width.value).toBe(320)
    window.dispatchEvent(new PointerEvent("pointercancel"))
    window.dispatchEvent(new PointerEvent("pointermove", { clientX: 100 }))
    expect(state.width.value).toBe(320)
    wrapper.unmount()
  })

  it("supports keyboard resizing and removes drag listeners on unmount", async () => {
    const state = useAgentWindow()
    state.resizeTo(400)
    const wrapper = mountWindow()
    const handle = wrapper.find('[data-agent-window-resize="w"]')
    await handle.trigger("keydown", { key: "ArrowLeft" })
    expect(state.width.value).toBe(416)
    await handle.trigger("keydown", { key: "ArrowRight", shiftKey: true })
    expect(state.width.value).toBe(352)
    await handle.trigger("keydown", { key: "Home" })
    expect(state.width.value).toBe(320)
    handle.element.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 600, bubbles: true }))
    wrapper.unmount()
    window.dispatchEvent(new PointerEvent("pointermove", { clientX: 500 }))
    expect(state.width.value).toBe(320)
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

  it("refreshes saved views after confirming a new bookmark", async () => {
    streamChatMock.mockImplementation(
      async (
        _input: unknown,
        handlers: {
          onOutcome?: (outcome: import("@/api/types").AIChatOutcomeDTO) => void
          onConfirmRequired?: (event: {
            name: string
            confirmToken: string
            changes: { path: string; before?: unknown; after?: unknown }[]
            arguments: Record<string, unknown>
            sessionId?: string
          }) => void
        },
      ) => {
        handlers.onConfirmRequired?.({
          name: "create_saved_view",
          confirmToken: "cfm_1",
          changes: [
            { path: "savedView.name", before: "", after: "未看完" },
            { path: "savedView.filters", before: null, after: { schemaVersion: 1, playState: "unwatched" } },
          ],
          arguments: { name: "未看完", filters: { schemaVersion: 1, playState: "unwatched" } },
          sessionId: "ses_1",
        })
        handlers.onOutcome?.({ status: "needs_confirmation", reasonCode: "confirmation_required" })
      },
    )
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("帮我存成书签")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    expect(wrapper.find("[data-agent-confirm-card]").exists()).toBe(true)
    expect(wrapper.get("[data-agent-outcome]").text()).toContain("agentWindow.outcome.needs_confirmation")
    confirmToolMock.mockRejectedValueOnce(new Error("save failed"))
    await wrapper.find("[data-agent-confirm-apply]").trigger("click")
    await flushPromises()
    expect(wrapper.get("[data-agent-confirm-card]").text()).toContain("save failed")
    expect(wrapper.get("[data-agent-outcome]").text()).toContain("agentWindow.outcome.needs_confirmation")
    expect(refreshSavedViewsMock).not.toHaveBeenCalled()
    await wrapper.find("[data-agent-confirm-apply]").trigger("click")
    await flushPromises()

    expect(confirmToolMock).toHaveBeenCalledWith({
      sessionId: "ses_1",
      name: "create_saved_view",
      arguments: { name: "未看完", filters: { schemaVersion: 1, playState: "unwatched" } },
      confirmToken: "cfm_1",
    })
    expect(refreshSavedViewsMock).toHaveBeenCalledTimes(1)
    expect(wrapper.get("[data-agent-outcome]").text()).toContain("agentWindow.outcomeSaved")
    expect(wrapper.get("[data-agent-outcome]").text()).not.toContain("agentWindow.outcome.needs_confirmation")
  })

  it("keeps the chrome header in the content column and brands the sidebar", () => {
    const wrapper = mountWindow()
    const sidebar = wrapper.find("[data-agent-window-sidebar]")
    const header = wrapper.find("[data-agent-window-header]")
    const brand = sidebar.find("[data-agent-window-brand]")
    expect(brand.exists()).toBe(true)
    expect(brand.text()).toBe("agentWindow.title")
    expect(brand.find("p").classes()).toContain("font-curated")
    expect(sidebar.find("[data-agent-window-header]").exists()).toBe(false)
    expect(wrapper.find("[data-agent-chat-wide]").find("[data-agent-window-header]").exists()).toBe(true)
    expect(header.find("[data-agent-window-sidebar-toggle]").exists()).toBe(true)
  })

  it("sends remaining @ mentions as explicit selected context", async () => {
	streamChatMock.mockImplementation(async (input: { context?: { mentions?: { kind: string; id: string }[]; contextVersion?: number; selectedMovieIds?: string[] } }, handlers: { onDelta: (d: string) => void }) => {
      expect(input.context?.mentions).toEqual([{ kind: "movie", id: "m1", label: "Hello" }])
	  expect(input.context?.contextVersion).toBe(1)
	  expect(input.context?.selectedMovieIds).toEqual(["m1"])
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

  it("shows and lets the user remove page context before sending", async () => {
    streamChatMock.mockImplementation(async (input: { context?: { route?: string } }, handlers: { onDelta: (d: string) => void }) => {
      expect(input.context?.route).toBeUndefined()
      handlers.onDelta("收到")
    })
    const wrapper = mountWindow()
    await flushPromises()
    const chip = wrapper.find('[data-agent-context-chip="route"]')
    expect(chip.exists()).toBe(true)
    expect(chip.attributes("data-size")).toBe("sm")
    expect(chip.classes()).toContain("h-8")
    expect(chip.classes()).not.toContain("min-h-11")
    await chip.trigger("click")
    expect(wrapper.find('[data-agent-context-chip="route"]').exists()).toBe(false)
    await wrapper.find("[data-agent-window-input]").setValue("不带页面上下文")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()
    expect(streamChatMock).toHaveBeenCalled()
  })

  it("renders a partial outcome with its evidence scope", async () => {
    streamChatMock.mockImplementation(async (_input: unknown, handlers: {
      onToolStart?: (event: { toolCallId: string; name: string }) => void
      onToolResult?: (event: { toolCallId: string; name: string; ok: boolean; evidence?: { source: string; truncated?: boolean; filters?: Record<string, string> }; providerRows?: { code: string; title: string; provider: string; score: number; inLibrary: boolean }[] }) => void
      onDelta: (delta: string) => void
      onOutcome?: (outcome: { status: "partial"; reason: string; retryable: boolean }) => void
    }) => {
      handlers.onToolStart?.({ toolCallId: "provider", name: "search_provider_titles" })
      handlers.onToolResult?.({
        toolCallId: "provider",
        name: "search_provider_titles",
        ok: true,
        evidence: { source: "provider", truncated: true, filters: { actorName: "Ada" } },
        providerRows: [{ code: "ABC-123", title: "Provider title", provider: "Metatube", score: 4.2, inLibrary: false }],
      })
      handlers.onDelta("这是已确认的结果。")
      handlers.onOutcome?.({ status: "partial", reason: "结果分页被截断。", retryable: true })
    })
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("她还拍过什么")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    expect(wrapper.find("[data-agent-outcome]").text()).toContain("agentWindow.outcome.partial")
    await wrapper.find("[data-agent-process-toggle]").trigger("click")
    expect(wrapper.find("[data-agent-evidence-cards]").text()).toContain("agentWindow.evidenceProvider")
    expect(wrapper.find("[data-agent-evidence-cards]").text()).toContain("agentWindow.evidenceTruncated")
    expect(wrapper.find("[data-agent-provider-rows]").text()).toContain("agentWindow.providerOffLibrary")
  })

  it("requires a candidate selection before continuing an ambiguous entity request", async () => {
    let calls = 0
    streamChatMock.mockImplementation(async (input: { context?: { selectedMovieIds?: string[] } }, handlers: {
      onToolStart?: (event: { toolCallId: string; name: string }) => void
      onToolResult?: (event: { toolCallId: string; name: string; ok: boolean; resolution?: unknown }) => void
      onOutcome?: (outcome: { status: "needs_input" | "completed" }) => void
      onDelta: (delta: string) => void
    }) => {
      calls += 1
      if (calls === 1) {
        handlers.onToolStart?.({ toolCallId: "resolve", name: "resolve_entities" })
        handlers.onToolResult?.({
          toolCallId: "resolve",
          name: "resolve_entities",
          ok: true,
          resolution: {
            query: "Same",
            kind: "movie",
            status: "ambiguous",
            candidates: [
              { kind: "movie", movieId: "m1", title: "Same", code: "ABC-001" },
              { kind: "movie", movieId: "m2", title: "Same", code: "ABC-002" },
            ],
          },
        })
        handlers.onDelta("我找到了多个匹配，请选择一个。")
        handlers.onOutcome?.({ status: "needs_input" })
        return
      }
      expect(input.context?.selectedMovieIds).toEqual(["m2"])
      handlers.onDelta("已按你选择的影片继续。")
      handlers.onOutcome?.({ status: "completed" })
    })
    const wrapper = mountWindow()
    await flushPromises()
    await wrapper.find("[data-agent-window-input]").setValue("查 Same")
    await wrapper.find("[data-agent-window-send]").trigger("click")
    await flushPromises()

    expect(wrapper.find("[data-agent-entity-resolution]").text()).toContain("agentWindow.resolutionAmbiguous")
    await wrapper.find('[data-agent-entity-candidate="m2"]').trigger("click")
    await flushPromises()
    expect(calls).toBe(2)
    expect(wrapper.text()).toContain("已按你选择的影片继续")
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
    useAgentWindow().resizeTo(800)
    await flushPromises()
    expect(wrapper.find("[data-agent-chat-wide]").attributes("data-agent-chat-wide")).toBe("true")
  })

  it("keeps the message scroller on the window edge instead of the content column", async () => {
    useAgentWindow().resizeTo(800)
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
