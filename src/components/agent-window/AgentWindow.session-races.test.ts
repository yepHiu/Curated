import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import AgentWindow from "@/components/agent-window/AgentWindow.vue"

import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAgentWindow } from "@/composables/use-agent-window"
import type { AIChatStreamHandlers, AIChatStreamRequest } from "@/services/contracts/ai-service"
import type { AIChatSessionDetailDTO } from "@/api/types"

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

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => ({
    comicLibraryEnabled: { value: false },
    comics: { value: [] },
  }),
}))

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => ({
    photoLibraryEnabled: { value: false },
    photos: { value: [] },
  }),
}))

function mountWindow() {
  return mount(AgentWindow, {
    global: {
      stubs: {
        Teleport: true,
        AgentChatThread: { name: 'AgentChatThread', props: ['entries', 'errorMessage', 'hasOlder', 'loadingOlder'], emits: ['loadOlder'], template: '<div />', methods: { scrollToEnd() {}, captureAnchor() { return () => {} } } },
        AgentChatComposer: { name: 'AgentChatComposer', props: ['modelValue', 'streaming', 'disabled'], emits: ['update:modelValue', 'send'], template: `<div><input data-agent-window-input :value="modelValue" @input="$emit('update:modelValue', $event.target.value)" /><button data-agent-window-send @click="$emit('send')" /></div>`, methods: { focus() {} } },
      },
    },
  })
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: Error) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

function sessionDetail(id: string): AIChatSessionDetailDTO {
  return { id, title: id, createdAt: "", updatedAt: "", messages: [
    { id: `${id}-user`, sessionId: id, role: "user", content: `history-${id}`, seq: 1, createdAt: "" },
  ] }
}

describe("AgentWindow session request ownership", () => {
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



  it("discards old stream events and cleanup after switching and starting a new reply", async () => {
    const oldReply = deferred<void>()
    const newReply = deferred<void>()
    const callbacks: AIChatStreamHandlers[] = []
    const requests: AIChatStreamRequest[] = []
    listSessionsMock.mockResolvedValue(["ses_a", "ses_b"].map(id => ({ id, title: id, createdAt: "", updatedAt: "" })))
    getSessionMock.mockImplementation(async (id: string) => sessionDetail(id))
    streamChatMock.mockImplementation((input: AIChatStreamRequest, handlers: AIChatStreamHandlers) => {
      requests.push(input)
      callbacks.push(handlers)
      return callbacks.length === 1 ? oldReply.promise : newReply.promise
    })
    const wrapper = mountWindow()
    try {
      await flushPromises()
      await wrapper.find("[data-agent-window-input]").setValue("question A")
      await wrapper.find("[data-agent-window-send]").trigger("click")
      await wrapper.find('[data-agent-window-session="ses_b"]').trigger("click")
      await flushPromises()
      expect(callbacks[0]!.signal?.aborted).toBe(true)
      const sidebar = wrapper.findComponent({ name: "AgentChatSidebar" })
      const thread = wrapper.findComponent({ name: "AgentChatThread" })
      expect(sidebar.props("activeId")).toBe("ses_b")
      await wrapper.find("[data-agent-window-input]").setValue("question B")
      await wrapper.find("[data-agent-window-send]").trigger("click")
      callbacks[0]!.onSession?.("ses_a")
      callbacks[0]!.onDelta("stale response")
      callbacks[0]!.onOutcome?.({ status: "completed", reason: "stale outcome" })
      callbacks[0]!.onConfirmRequired?.({ toolCallId: "old", name: "save_movie_comment", confirmToken: "old", changes: [], arguments: {}, sessionId: "ses_a" })
      oldReply.reject(new Error("stale stream failure"))
      await flushPromises()
      expect(sidebar.props("activeId")).toBe("ses_b")
      expect(requests[1]!.sessionId).toBe("ses_b")
      expect(requests[1]!.messages.map(message => message.content)).toEqual(["question B"])
      expect(wrapper.findComponent({ name: "AgentChatComposer" }).props("streaming")).toBe(true)
      expect(thread.props("errorMessage")).toBe("")
      expect(JSON.stringify(thread.props("entries"))).not.toContain("stale")
      expect(JSON.stringify(thread.props("entries"))).not.toContain("confirmToken")
      callbacks[1]!.onDelta("current response")
      newReply.resolve()
      await flushPromises()
      expect(wrapper.findComponent({ name: "AgentChatComposer" }).props("streaming")).toBe(false)
      expect(JSON.stringify(thread.props("entries"))).toContain("current response")
    } finally {
      oldReply.resolve()
      newReply.resolve()
      wrapper.unmount()
    }
  })

  it("keeps the last selected history when earlier loads finish out of order", async () => {
    listSessionsMock.mockResolvedValue(["ses_a", "ses_b", "ses_c"].map(id => ({ id, title: id, createdAt: "", updatedAt: "" })))
    getSessionMock.mockImplementation(async (id: string) => sessionDetail(id))
    const wrapper = mountWindow()
    try {
      await flushPromises()
      const slow = deferred<AIChatSessionDetailDTO>()
      getSessionMock.mockImplementation((id: string) => id === "ses_b" ? slow.promise : Promise.resolve(sessionDetail(id)))
      await wrapper.find('[data-agent-window-session="ses_b"]').trigger("click")
      expect(wrapper.findComponent({ name: "AgentChatComposer" }).props("disabled")).toBe(true)
      await wrapper.find('[data-agent-window-session="ses_c"]').trigger("click")
      await flushPromises()
      slow.resolve(sessionDetail("ses_b"))
      await flushPromises()
      expect(wrapper.findComponent({ name: "AgentChatSidebar" }).props("activeId")).toBe("ses_c")
      expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).toContain("history-ses_c")
      expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).not.toContain("history-ses_b")
    } finally { wrapper.unmount() }
  })

  it("prepends older pages without dropping new replies and ignores a stale page after switching", async () => {
    listSessionsMock.mockResolvedValue(["ses_a", "ses_b"].map(id => ({ id, title: id, createdAt: "", updatedAt: "" })))
    getSessionMock.mockImplementation(async (id: string) => ({ ...sessionDetail(id), nextCursor: "older" }))
    streamChatMock.mockImplementation(async (_input: AIChatStreamRequest, handlers: AIChatStreamHandlers) => handlers.onDelta("live answer"))
    const wrapper = mountWindow()
    try {
      await flushPromises()
      await wrapper.find("[data-agent-window-input]").setValue("new question")
      await wrapper.find("[data-agent-window-send]").trigger("click")
      await flushPromises()
      const older = deferred<AIChatSessionDetailDTO>()
      getSessionMock.mockReturnValueOnce(older.promise)
      wrapper.findComponent({ name: "AgentChatThread" }).vm.$emit("loadOlder")
      await flushPromises()
      expect(getSessionMock).toHaveBeenLastCalledWith("ses_a", "older")
      older.resolve({ ...sessionDetail("ses_a"), nextCursor: "even-older", messages: [{ ...sessionDetail("ses_a").messages[0]!, id: "earlier", content: "earlier question" }] })
      await flushPromises()
      await vi.waitFor(() => expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).toContain("earlier question"))
      expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).toContain("live answer")
      const stale = deferred<AIChatSessionDetailDTO>()
      getSessionMock.mockReturnValueOnce(stale.promise)
      wrapper.findComponent({ name: "AgentChatThread" }).vm.$emit("loadOlder")
      await flushPromises()
      await wrapper.find('[data-agent-window-session="ses_b"]').trigger("click")
      await flushPromises()
      stale.resolve({ ...sessionDetail("ses_a"), messages: [{ ...sessionDetail("ses_a").messages[0]!, content: "stale history" }] })
      await flushPromises()
      expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).not.toContain("stale history")
      expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).toContain("history-ses_b")
    } finally { wrapper.unmount() }
  })

  it("retains memory status after the answer completes", async () => {
    streamChatMock.mockImplementation(async (_input: AIChatStreamRequest, handlers: AIChatStreamHandlers) => {
      handlers.onContextStatus?.({ phase: "compacting" })
      handlers.onContextStatus?.({ phase: "ready" })
      handlers.onDelta("continued answer")
      handlers.onOutcome?.({ status: "completed" })
    })
    const wrapper = mountWindow()
    try {
      await flushPromises()
      await wrapper.find("[data-agent-window-input]").setValue("continue the task")
      await wrapper.find("[data-agent-window-send]").trigger("click")
      await flushPromises()
      const entries = wrapper.findComponent({ name: "AgentChatThread" }).props("entries") as { kind: string; contextPhase?: string; thinkingActive?: boolean }[]
      expect(entries.find(entry => entry.kind === "process")).toMatchObject({ contextPhase: "ready", thinkingActive: false })
    } finally { wrapper.unmount() }
  })

  it.each(["needs_input", "partial", "error"])("removes empty answer but retains evidence after %s", async (status) => {
    streamChatMock.mockImplementation(async (_input: AIChatStreamRequest, handlers: AIChatStreamHandlers) => {
      handlers.onToolStart?.({ name: "search_movies", toolCallId: "read" })
      if (status === "error") throw new Error("connection lost")
      handlers.onOutcome?.({ status: status as "needs_input" | "partial", reason: "context budget reached" })
    })
    const wrapper = mountWindow()
    try {
      await flushPromises()
      await wrapper.find("[data-agent-window-input]").setValue("question")
      await wrapper.find("[data-agent-window-send]").trigger("click")
      await flushPromises()
      const entries = wrapper.findComponent({ name: "AgentChatThread" }).props("entries")
      expect(entries).not.toEqual(expect.arrayContaining([expect.objectContaining({ kind: "assistant", content: "" })]))
      expect(entries).toEqual(expect.arrayContaining([expect.objectContaining({ kind: "process", tools: [expect.objectContaining({ pending: false })] })]))
      expect(entries).toEqual(expect.arrayContaining([expect.objectContaining({ kind: "outcome" })]))
    } finally { wrapper.unmount() }
  })

  it("retains partial output and restores the draft after a stream failure", async () => {
    streamChatMock.mockImplementation(async (_input: AIChatStreamRequest, handlers: AIChatStreamHandlers) => {
      handlers.onToolStart?.({ name: "search_movies", toolCallId: "interrupted" })
      handlers.onDelta("partial answer")
      throw new Error("connection lost")
    })
    const wrapper = mountWindow()
    try {
      await flushPromises()
      await wrapper.find("[data-agent-window-input]").setValue("question")
      await wrapper.find("[data-agent-window-send]").trigger("click")
      await flushPromises()
      const entries = wrapper.findComponent({ name: "AgentChatThread" }).props("entries")
      expect(entries).toEqual(expect.arrayContaining([expect.objectContaining({ kind: "assistant", content: "partial answer" })]))
      expect(entries).toEqual(expect.arrayContaining([expect.objectContaining({ kind: "process", tools: [expect.objectContaining({ pending: false })] })]))
      expect(wrapper.findComponent({ name: "AgentChatComposer" }).props("modelValue")).toBe("question")
      expect(wrapper.findComponent({ name: "AgentChatComposer" }).props("streaming")).toBe(false)
    } finally { wrapper.unmount() }
  })

  it("ignores a delayed new-chat creation after the user selects an existing chat", async () => {
    listSessionsMock.mockResolvedValue(["ses_a", "ses_b"].map(id => ({ id, title: id, createdAt: "", updatedAt: "" })))
    getSessionMock.mockImplementation(async (id: string) => sessionDetail(id))
    const created = deferred<{ id: string; title: string; createdAt: string; updatedAt: string }>()
    createSessionMock.mockReturnValue(created.promise)
    const wrapper = mountWindow()
    try {
      await flushPromises()
      await wrapper.find("[data-agent-window-new]").trigger("click")
      await wrapper.find('[data-agent-window-session="ses_b"]').trigger("click")
      await flushPromises()
      created.resolve({ id: "ses_new", title: "", createdAt: "", updatedAt: "" })
      await flushPromises()
      expect(wrapper.findComponent({ name: "AgentChatSidebar" }).props("activeId")).toBe("ses_b")
      expect(JSON.stringify(wrapper.findComponent({ name: "AgentChatThread" }).props("entries"))).toContain("history-ses_b")
    } finally { wrapper.unmount() }
  })
})

