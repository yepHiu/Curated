<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { onKeyStroke } from "@vueuse/core"
import { PanelLeft, Plus, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import type { AIAgentMovieCardDTO, AIChatContextDTO, AIChatMessageDTO, AIChatSessionDTO, AIChatStoredMessageDTO, AIEntityCandidateDTO } from "@/api/types"
import { agentPageContext } from "@/lib/agent-page-context"
import { restoreChatHistory } from "./restore-history"
import { isAgentProcessTool } from "@/lib/agent-tool-labels"
import { mentionsStillInText, type AgentMention } from "@/lib/agent-mentions"
import {
  AGENT_WINDOW_CHAT_WIDE_MIN,
  AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH,
  useAgentWindow,
} from "@/composables/use-agent-window"
import { useAIService } from "@/services/ai-service"
import { useLibraryService } from "@/services/library-service"
import { AIServiceError } from "@/services/contracts/ai-service"
import AgentChatComposer from "./AgentChatComposer.vue"
import AgentChatSidebar from "./AgentChatSidebar.vue"
import AgentChatThread from "./AgentChatThread.vue"
import type { AgentChatEntry } from "./types"

const AI_PROVIDER_UNAVAILABLE_CODE = "AI_PROVIDER_UNAVAILABLE"
const SESSION_STORAGE_KEY = "curated-agent-session-id-v1"

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const aiService = useAIService()
const libraryService = useLibraryService()
const {
  open,
  position,
  size,
  sidebarOpen,
  isMobileViewport,
  closeWindow,
  moveTo,
  resizeTo,
  setSidebarOpen,
} = useAgentWindow()

const entries = ref<AgentChatEntry[]>([])
const sessions = ref<AIChatSessionDTO[]>([])
const sessionId = ref("")
const draft = ref("")
const mentions = ref<AgentMention[]>([])
const omittedContext = ref<string[]>([])
const streaming = ref(false)
const loadingSession = ref(false)
const loadingOlder = ref(false)
const historyCursor = ref("")
let storedHistory: AIChatStoredMessageDTO[] = []
let historyEntryIds = new Set<string>()
const providerUnconfigured = ref(false)
const errorMessage = ref("")
const threadRef = ref<{ scrollToEnd: () => void; captureAnchor?: () => (() => void) } | null>(null)
const composerRef = ref<{ focus: () => void; mentionOpen?: boolean; closeMentions?: () => void } | null>(null)

let abortController: AbortController | null = null
let streamSeq = 0
let sessionLoadSeq = 0
let entrySeq = 0

const sidebarOverlays = computed(
  () => isMobileViewport.value || size.value.width < AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH,
)

const chatWide = computed(
  () => !isMobileViewport.value && size.value.width >= AGENT_WINDOW_CHAT_WIDE_MIN,
)

const headerTitle = computed(() => {
  const title = sessions.value.find((item) => item.id === sessionId.value)?.title?.trim()
  return title || t("agentWindow.sessionPlaceholder")
})

function nextEntryId(prefix: string) {
  entrySeq += 1
  return `${prefix}-${entrySeq}`
}

onKeyStroke("Escape", (e) => {
  if (!open.value) return
  if (e.defaultPrevented) return
  e.preventDefault()
  if (composerRef.value?.mentionOpen) {
    composerRef.value.closeMentions?.()
    return
  }
  if (sidebarOpen.value && sidebarOverlays.value) {
    setSidebarOpen(false)
    return
  }
  close()
})

onBeforeUnmount(() => {
  sessionLoadSeq += 1
  abortController?.abort()
})

watch(open, async (isOpen) => {
  if (isOpen) {
    await refreshSessions()
    await nextTick()
    composerRef.value?.focus()
    return
  }
  abortController?.abort()
}, { immediate: true })

function close() {
  abortController?.abort()
  closeWindow()
}

async function scrollListToEnd() {
  await nextTick()
  threadRef.value?.scrollToEnd()
}

function persistSessionId(id: string) {
  if (sessionId.value !== id) resetHistoryPage()
  sessionId.value = id
  if (id) {
    localStorage.setItem(SESSION_STORAGE_KEY, id)
  } else {
    localStorage.removeItem(SESSION_STORAGE_KEY)
  }
}

function resetHistoryPage() {
  historyCursor.value = ""
  loadingOlder.value = false
  storedHistory = []
  historyEntryIds = new Set()
}

async function loadOlder() {
  if (!historyCursor.value || loadingOlder.value || loadingSession.value || streaming.value) return
  const seq = sessionLoadSeq
  const id = sessionId.value
  loadingOlder.value = true
  errorMessage.value = ""
  try {
    const page = await aiService.getSession(id, historyCursor.value)
    if (seq !== sessionLoadSeq || id !== sessionId.value) return
    const restoreAnchor = threadRef.value?.captureAnchor?.()
    const known = new Set(storedHistory.map(message => message.id))
    storedHistory = [...page.messages.filter(message => !known.has(message.id)), ...storedHistory]
    const live = entries.value.filter(entry => !historyEntryIds.has(entry.id))
    const previous = new Map(entries.value.map(entry => [entry.id, entry]))
    const restored = restoreChatHistory(storedHistory)
    for (const entry of restored) {
      const old = previous.get(entry.id)
      if (entry.kind === "resolution" && old?.kind === "resolution") entry.selected = old.selected
      if (entry.kind === "process" && old?.kind === "process") entry.open = old.open
    }
    historyEntryIds = new Set(restored.map(entry => entry.id))
    entries.value = [...restored, ...live]
    historyCursor.value = page.nextCursor ?? ""
    await nextTick()
    restoreAnchor?.()
  } catch {
    if (seq === sessionLoadSeq) errorMessage.value = t("agentWindow.historyLoadFailed")
  } finally {
    if (seq === sessionLoadSeq) loadingOlder.value = false
  }
}

async function refreshSessions() {
  const loadSeq = sessionLoadSeq
  try {
    const rows = await aiService.listSessions()
    if (loadSeq !== sessionLoadSeq) return
    sessions.value = rows
  } catch {
    if (loadSeq !== sessionLoadSeq) return
    sessions.value = []
  }
  if (loadingSession.value) return
  const remembered = sessionId.value || localStorage.getItem(SESSION_STORAGE_KEY) || ""
  if (remembered && sessions.value.some((item) => item.id === remembered)) {
    if (sessionId.value !== remembered || entries.value.length === 0) {
      await loadSession(remembered)
    }
    return
  }
  if (!sessionId.value && sessions.value[0]) {
    await loadSession(sessions.value[0].id)
  }
}

async function loadSession(id: string) {
  abandonStream()
  resetHistoryPage()
  const loadSeq = ++sessionLoadSeq
  loadingSession.value = true
  persistSessionId(id)
  entries.value = []
  errorMessage.value = ""
  providerUnconfigured.value = false
  try {
    const detail = await aiService.getSession(id)
    if (loadSeq !== sessionLoadSeq) return
    entries.value = restoreChatHistory(detail.messages)
    storedHistory = detail.messages
    historyEntryIds = new Set(entries.value.map(entry => entry.id))
    historyCursor.value = detail.nextCursor ?? ""
  } catch {
    if (loadSeq !== sessionLoadSeq) return
    entries.value = []
  } finally {
    if (loadSeq === sessionLoadSeq) loadingSession.value = false
  }
  void scrollListToEnd()
}

async function selectSession(id: string) {
  await loadSession(id)
  if (sidebarOverlays.value) {
    setSidebarOpen(false)
  }
}

async function startNewChat() {
  abandonStream()
  const loadSeq = ++sessionLoadSeq
  loadingSession.value = true
  persistSessionId("")
  entries.value = []
  try {
    const created = await aiService.createSession()
    if (loadSeq !== sessionLoadSeq) return
    sessions.value = [created, ...sessions.value.filter((item) => item.id !== created.id)]
    persistSessionId(created.id)
  } catch {
    if (loadSeq !== sessionLoadSeq) return
    persistSessionId("")
  } finally {
    if (loadSeq === sessionLoadSeq) loadingSession.value = false
  }
  entries.value = []
  providerUnconfigured.value = false
  errorMessage.value = ""
  mentions.value = []
  draft.value = ""
  if (sidebarOverlays.value) {
    setSidebarOpen(false)
  }
  await nextTick()
  composerRef.value?.focus()
}

async function deleteChat(id: string) {
  if (sessionId.value === id) {
    abandonStream()
    sessionLoadSeq += 1
    loadingSession.value = false
  }
  try {
    await aiService.deleteSession(id)
  } catch {
    return
  }
  sessions.value = sessions.value.filter((item) => item.id !== id)
  if (sessionId.value !== id) return
  persistSessionId("")
  entries.value = []
  if (sessions.value[0]) {
    await loadSession(sessions.value[0].id)
  }
}

function attachMovies(assistantId: string, movies: AIAgentMovieCardDTO[]) {
  const current = entries.value.find((entry) => entry.id === assistantId)
  if (current?.kind === "assistant") {
    current.movies = movies
  }
}

function findProcessFor(assistantId: string) {
  const index = entries.value.findIndex((entry) => entry.id === assistantId)
  for (let i = index - 1; i >= 0; i -= 1) {
    const entry = entries.value[i]
    if (entry?.kind === "process") return entry
    if (entry?.kind === "user") break
  }
  return null
}

function collapseProcess(assistantId: string) {
  const process = findProcessFor(assistantId)
  if (!process) return
  process.thinkingActive = false
  process.open = false
  if (!process.thinking.trim() && process.tools.length === 0) {
    const index = entries.value.findIndex((entry) => entry.id === process.id)
    if (index >= 0) removeEntryAt(index)
  }
}

function chatContextFrom(text: string, active: AgentMention[], omitted: readonly string[] = [], selected?: AIEntityCandidateDTO): AIChatContextDTO | undefined {
  const page: AIChatContextDTO = { ...(agentPageContext(route) ?? {}) }
  const omittedKeys = new Set(omitted)
  if (omittedKeys.has("route")) delete page.route
  if (omittedKeys.has("movie")) delete page.movieId
  if (omittedKeys.has("actor")) delete page.actorName
  if (page.activeFilters) {
    const filters = { ...page.activeFilters }
    if (omittedKeys.has("filter:query")) {
      delete filters.query
      delete page.query
    }
    if (omittedKeys.has("filter:tag")) delete filters.tag
    if (omittedKeys.has("filter:actor")) delete filters.actor
    if (omittedKeys.has("filter:playState")) delete filters.playState
    if (omittedKeys.has("filter:runtime")) delete filters.runtime
    page.activeFilters = Object.keys(filters).length > 0 ? filters : undefined
  }
  const kept = mentionsStillInText(active, text).slice(0, 8)
  if (kept.length > 0) {
    page.mentions = kept.map((item) => ({
      kind: item.kind,
      id: item.id,
      label: item.label,
    }))
    const movieIds = kept.filter((item) => item.kind === "movie").map((item) => item.id)
    const actors = kept.filter((item) => item.kind === "actor").map((item) => item.id || item.label)
    if (movieIds.length > 0) page.selectedMovieIds = [...new Set(movieIds)]
    if (actors.length > 0) page.selectedActors = [...new Set(actors)]
    if (page.selectedMovieIds?.length || page.selectedActors?.length) page.contextVersion = 1
  }
  if (selected?.kind === "movie" && selected.movieId) {
    page.selectedMovieIds = [...new Set([...(page.selectedMovieIds ?? []), selected.movieId])]
    page.contextVersion = 1
  }
  if (selected?.kind === "actor" && selected.actorName) {
    page.selectedActors = [...new Set([...(page.selectedActors ?? []), selected.actorName])]
    page.contextVersion = 1
  }
  if (!page.activeFilters && !page.selectedMovieIds?.length && !page.selectedActors?.length) {
    delete page.contextVersion
  }
  return Object.keys(page).length > 0 ? page : undefined
}

type AgentContextChip = { key: string; label: string }

const contextChips = computed<AgentContextChip[]>(() => {
  const page = chatContextFrom(draft.value, mentions.value, omittedContext.value)
  if (!page) return []
  const chips: AgentContextChip[] = []
  if (page.route) chips.push({ key: "route", label: t("agentWindow.contextRoute", { route: page.route }) })
  if (page.movieId) chips.push({ key: "movie", label: t("agentWindow.contextMovie", { id: page.movieId }) })
  if (page.actorName) chips.push({ key: "actor", label: t("agentWindow.contextActor", { name: page.actorName }) })
  if (page.activeFilters?.query) chips.push({ key: "filter:query", label: t("agentWindow.contextQuery", { value: page.activeFilters.query }) })
  if (page.activeFilters?.tag) chips.push({ key: "filter:tag", label: t("agentWindow.contextTag", { value: page.activeFilters.tag }) })
  if (page.activeFilters?.actor) chips.push({ key: "filter:actor", label: t("agentWindow.contextFilterActor", { value: page.activeFilters.actor }) })
  if (page.activeFilters?.playState) chips.push({ key: "filter:playState", label: t("agentWindow.contextPlayState", { value: page.activeFilters.playState }) })
  if (page.activeFilters?.runtime) chips.push({ key: "filter:runtime", label: t("agentWindow.contextRuntime", { value: page.activeFilters.runtime }) })
  return chips
})

function omitContext(key: string) {
  if (!omittedContext.value.includes(key)) omittedContext.value = [...omittedContext.value, key]
}

function removeAssistantTurn(assistantId: string) {
  const process = findProcessFor(assistantId)
  const index = entries.value.findIndex((entry) => entry.id === assistantId)
  if (index >= 0) removeEntryAt(index)
  if (process && !process.tools.length && !process.thinking) {
    const processIndex = entries.value.findIndex((entry) => entry.id === process.id)
    if (processIndex >= 0) removeEntryAt(processIndex)
  }
}

function openMovieDetail(movieId: string) {
  const id = movieId.trim()
  if (!id) return
  void router.push({ name: "detail", params: { id } })
}

async function applyConfirm(entryId: string) {
  const entry = entries.value.find((item) => item.id === entryId)
  if (!entry || entry.kind !== "confirm" || entry.status !== "pending") return
  entry.status = "applying"
  entry.error = undefined
  try {
    await aiService.confirmTool({
      sessionId: entry.sessionId,
      name: entry.name,
      arguments: entry.arguments,
      confirmToken: entry.confirmToken,
    })
    entry.status = "applied"
    if (entry.name === "create_saved_view") {
      try {
        await libraryService.refreshSavedViews()
      } catch {
        // Write already succeeded; the next library visit can catch up.
      }
    }
  } catch (err) {
    entry.status = "pending"
    entry.error = err instanceof AIServiceError ? err.message : (err as Error).message
  }
}

function discardConfirm(entryId: string) {
  const entry = entries.value.find((item) => item.id === entryId)
  if (!entry || entry.kind !== "confirm" || entry.status === "applied") return
  entry.status = "discarded"
}

function removeEntryAt(index: number) {
  entries.value.splice(index, 1)
}

function stop() {
  if (!streaming.value) return
  abortController?.abort()
  entries.value.push({
    id: nextEntryId("outcome"),
    kind: "outcome",
    outcome: { status: "cancelled", reason: t("agentWindow.cancelledReason") },
  })
}

function abandonStream() {
  streamSeq += 1
  abortController?.abort()
  abortController = null
  streaming.value = false
}

async function send(selected?: AIEntityCandidateDTO) {
  const content = draft.value.trim()
  if (!content || streaming.value || loadingSession.value || loadingOlder.value) return
  draft.value = ""
  const activeMentions = mentions.value
  mentions.value = []
  const activeContext = chatContextFrom(content, activeMentions, omittedContext.value, selected)
  omittedContext.value = []
  entries.value.push({ id: nextEntryId("user"), kind: "user", content })

  // The server owns session history. Sending it again eventually exceeds the
  // request limit and cannot restore tool evidence or trusted references.
  const history: AIChatMessageDTO[] = [{ role: "user", content }]
  const process: AgentChatEntry = {
    id: nextEntryId("process"),
    kind: "process",
    thinking: "",
    thinkingActive: true,
    tools: [],
    open: false,
  }
  const assistant: AgentChatEntry = { id: nextEntryId("assistant"), kind: "assistant", content: "", movies: [] }
  entries.value.push(process, assistant)
  const assistantId = assistant.id

  streaming.value = true
  providerUnconfigured.value = false
  errorMessage.value = ""
  const controller = new AbortController()
  abortController = controller
  const seq = ++streamSeq
  void scrollListToEnd()

  try {
    await aiService.streamChat(
      {
        messages: history,
        sessionId: sessionId.value || undefined,
        context: activeContext,
        locale: locale.value,
      },
      {
        signal: controller.signal,
        onSession(id) {
          if (seq !== streamSeq || controller.signal.aborted) return
          persistSessionId(id)
        },
        onThinking(delta) {
          if (seq !== streamSeq || controller.signal.aborted) return
          const current = findProcessFor(assistantId)
          if (!current || !delta) return
          // Auto-open once; phase changes and later chunks must preserve the
          // user's disclosure choice until the entire turn finishes.
          if (!current.thinking) current.open = true
          current.thinking += delta
          current.thinkingActive = true
          void scrollListToEnd()
        },
        onDelta(delta) {
          if (seq !== streamSeq || controller.signal.aborted) return
          const process = findProcessFor(assistantId)
          if (process) process.thinkingActive = false
          const current = entries.value.find((entry) => entry.id === assistantId)
          if (current?.kind === "assistant") {
            current.content += delta
          }
          void scrollListToEnd()
        },
        onToolStart(event) {
          if (seq !== streamSeq || controller.signal.aborted) return
          const current = findProcessFor(assistantId)
          if (!current || !isAgentProcessTool(event.name)) return
          current.thinkingActive = false
          current.tools.push({
            toolCallId: event.toolCallId,
            name: event.name,
            pending: true,
          })
          void scrollListToEnd()
        },
        onToolResult(event) {
          if (seq !== streamSeq || controller.signal.aborted) return
          const current = findProcessFor(assistantId)
          const card = current?.tools.find((item) => item.toolCallId === event.toolCallId)
          if (card) {
            card.pending = false
            card.ok = event.ok
            card.evidence = event.evidence
            card.providerRows = event.providerRows
          }
          if (event.movies?.length) {
            attachMovies(assistantId, event.movies)
          }
          if (event.resolution && event.resolution.status !== "matched") {
            entries.value.push({ id: nextEntryId("resolution"), kind: "resolution", resolution: event.resolution })
          }
          void scrollListToEnd()
        },
        onMovieCards(movies) {
          if (seq !== streamSeq || controller.signal.aborted) return
          attachMovies(assistantId, movies)
          void scrollListToEnd()
        },
        onConfirmRequired(event) {
          if (seq !== streamSeq || controller.signal.aborted) return
          entries.value.push({
            id: nextEntryId("confirm"),
            kind: "confirm",
            name: event.name,
            confirmToken: event.confirmToken,
            expiresAt: event.expiresAt,
            changes: event.changes,
            arguments: event.arguments,
            sessionId: event.sessionId || sessionId.value,
            status: "pending",
          })
          void scrollListToEnd()
        },
        onOutcome(outcome) {
          if (seq !== streamSeq || controller.signal.aborted) return
          entries.value.push({ id: nextEntryId("outcome"), kind: "outcome", outcome })
          void scrollListToEnd()
        },
      },
    )
    if (seq === streamSeq && !controller.signal.aborted) void refreshSessions()
  } catch (err) {
    if (seq === streamSeq && !controller.signal.aborted) {
      const aiErr = err instanceof AIServiceError ? err : null
      if (aiErr?.code === AI_PROVIDER_UNAVAILABLE_CODE) {
        providerUnconfigured.value = true
      } else {
        errorMessage.value = aiErr?.message ?? (err as Error).message ?? t("agentWindow.errorFallback")
      }
      if (!draft.value.trim()) draft.value = content
      entries.value.push({ id: nextEntryId("outcome"), kind: "outcome",
        outcome: { status: "failed", reason: errorMessage.value || t("agentWindow.errorFallback"), retryable: true } })
    }
  } finally {
    if (seq === streamSeq) {
      streaming.value = false
      // A transport error/abort can arrive before a tool result event.
      for (const tool of findProcessFor(assistantId)?.tools ?? []) tool.pending = false
      collapseProcess(assistantId)
      const current = entries.value.find((entry) => entry.id === assistantId)
      if (current?.kind === "assistant" && !current.content && !current.movies?.length) {
        removeAssistantTurn(assistantId)
      }
      void scrollListToEnd()
    }
  }
}

function selectEntity(entryId: string, candidate: AIEntityCandidateDTO) {
  if (streaming.value) return
  const entry = entries.value.find((item) => item.id === entryId)
  if (!entry || entry.kind !== "resolution" || entry.selected) return
  entry.selected = true
  draft.value = `选择「${candidate.kind === "movie" ? candidate.title || candidate.code : candidate.actorName}」，请继续处理我刚才的请求。`
  void send(candidate)
}

const dragging = ref(false)
const resizing = ref(false)

function onHeaderPointerdown(e: PointerEvent) {
  if (isMobileViewport.value || e.button !== 0) return
  const target = e.target as HTMLElement
  if (target.closest("button, a, select")) return
  dragging.value = true
  const originX = e.clientX
  const originY = e.clientY
  const baseX = position.value.x
  const baseY = position.value.y

  const onMove = (ev: PointerEvent) => {
    moveTo(baseX + (ev.clientX - originX), baseY + (ev.clientY - originY))
  }
  const onUp = () => {
    dragging.value = false
    window.removeEventListener("pointermove", onMove)
    window.removeEventListener("pointerup", onUp)
  }
  window.addEventListener("pointermove", onMove)
  window.addEventListener("pointerup", onUp)
}

type ResizeEdge = "e" | "s" | "se"

function onResizePointerdown(edge: ResizeEdge, e: PointerEvent) {
  if (isMobileViewport.value || e.button !== 0) return
  e.preventDefault()
  e.stopPropagation()
  resizing.value = true
  const originX = e.clientX
  const originY = e.clientY
  const baseWidth = size.value.width
  const baseHeight = size.value.height

  const onMove = (ev: PointerEvent) => {
    const nextWidth = edge === "s" ? baseWidth : baseWidth + (ev.clientX - originX)
    const nextHeight = edge === "e" ? baseHeight : baseHeight + (ev.clientY - originY)
    resizeTo(nextWidth, nextHeight)
  }
  const onUp = () => {
    resizing.value = false
    window.removeEventListener("pointermove", onMove)
    window.removeEventListener("pointerup", onUp)
  }
  window.addEventListener("pointermove", onMove)
  window.addEventListener("pointerup", onUp)
}

const windowStyle = ref<Record<string, string>>({})
watch(
  [position, size, isMobileViewport],
  () => {
    windowStyle.value = isMobileViewport.value
      ? {}
      : {
          left: `${position.value.x}px`,
          top: `${position.value.y}px`,
          width: `${size.value.width}px`,
          height: `${size.value.height}px`,
          maxHeight: "calc(100dvh - 24px)",
        }
  },
  { immediate: true },
)
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed z-[120] flex flex-col overflow-hidden rounded-2xl border border-border bg-background text-foreground shadow-lg max-md:inset-2"
      :class="dragging || resizing ? 'select-none' : ''"
      :style="windowStyle"
      role="dialog"
      aria-label="Curated Agent"
      data-agent-window
    >
      <div class="relative flex min-h-0 flex-1">
        <AgentChatSidebar
          v-if="sidebarOpen && !sidebarOverlays"
          :sessions="sessions"
          :active-id="sessionId"
          @create="startNewChat"
          @select="selectSession"
          @delete="deleteChat"
          @title-pointerdown="onHeaderPointerdown"
        />
        <div
          class="relative flex min-h-0 min-w-0 flex-1 flex-col"
          :data-agent-chat-wide="chatWide ? 'true' : 'false'"
        >
          <div
            class="flex min-h-11 shrink-0 cursor-grab items-center gap-1 border-b border-border px-1.5 py-1 text-foreground active:cursor-grabbing md:min-h-10"
            :class="isMobileViewport ? '' : 'touch-none'"
            data-agent-window-header
            @pointerdown="onHeaderPointerdown"
          >
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="size-11 shrink-0 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground md:size-8"
              :aria-label="t('agentWindow.toggleSidebar')"
              :aria-pressed="sidebarOpen"
              data-agent-window-sidebar-toggle
              @click="setSidebarOpen(!sidebarOpen)"
            >
              <PanelLeft class="size-4" />
            </Button>
            <p class="min-w-0 flex-1 truncate px-1 text-[13px] font-medium">{{ headerTitle }}</p>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="size-11 shrink-0 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground md:size-8"
              :aria-label="t('agentWindow.newChat')"
              data-agent-window-header-new
              @click="startNewChat"
            >
              <Plus class="size-4" />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="size-11 shrink-0 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground md:size-8"
              :aria-label="t('agentWindow.close')"
              @click="close"
            >
              <X class="size-4" />
            </Button>
          </div>
          <div class="relative flex min-h-0 flex-1 flex-col">
            <AgentChatThread
              :has-older="Boolean(historyCursor)"
              :loading-older="loadingOlder"
              :history-disabled="streaming || loadingSession"
              @load-older="loadOlder"
              ref="threadRef"
              :entries="entries"
              :provider-unconfigured="providerUnconfigured"
              :error-message="errorMessage"
              :wide="chatWide"
              @close="close"
              @open-movie="openMovieDetail"
              @apply-confirm="applyConfirm"
              @discard-confirm="discardConfirm"
              @select-entity="selectEntity"
            />
            <div
              class="mx-auto w-full shrink-0"
              :class="chatWide ? 'max-w-[52rem] px-6' : 'px-4'"
            >
              <div v-if="contextChips.length" class="flex flex-wrap gap-2 border-t border-border/60 py-2" data-agent-context-chips>
                <Button
                  v-for="chip in contextChips"
                  :key="chip.key"
                  type="button"
                  variant="secondary"
                  class="h-auto min-h-11 max-w-full gap-1 rounded-full px-3 py-1 text-xs text-muted-foreground hover:text-foreground"
                  :aria-label="t('agentWindow.removeContext', { context: chip.label })"
                  :data-agent-context-chip="chip.key"
                  @click="omitContext(chip.key)"
                >
                  <span class="truncate">{{ chip.label }}</span>
                  <X class="size-3.5 shrink-0" aria-hidden="true" />
                </Button>
              </div>
              <AgentChatComposer
                ref="composerRef"
                v-model="draft"
                v-model:mentions="mentions"
                :streaming="streaming"
                :disabled="loadingSession || loadingOlder"
                @send="send"
                @stop="stop"
              />
            </div>
            <template v-if="sidebarOpen && sidebarOverlays">
              <button
                type="button"
                class="absolute inset-0 z-20 bg-background/70"
                :aria-label="t('agentWindow.toggleSidebar')"
                data-agent-window-sidebar-backdrop
                @click="setSidebarOpen(false)"
              />
              <AgentChatSidebar
                class="absolute inset-y-0 left-0 z-40 shadow-md"
                :sessions="sessions"
                :active-id="sessionId"
                @create="startNewChat"
                @select="selectSession"
                @delete="deleteChat"
                @title-pointerdown="onHeaderPointerdown"
              />
            </template>
          </div>
        </div>
      </div>

      <template v-if="!isMobileViewport">
        <div
          class="absolute top-0 right-0 z-20 h-11 w-2 cursor-ew-resize touch-none"
          aria-hidden="true"
          data-agent-window-resize="e"
          @pointerdown="onResizePointerdown('e', $event)"
        />
        <div
          class="absolute inset-x-0 bottom-0 z-20 h-2 cursor-ns-resize touch-none"
          aria-hidden="true"
          data-agent-window-resize="s"
          @pointerdown="onResizePointerdown('s', $event)"
        />
        <button
          type="button"
          class="absolute right-0 bottom-0 z-50 size-4 cursor-nwse-resize touch-none rounded-br-2xl"
          :aria-label="t('agentWindow.resize')"
          data-agent-window-resize="se"
          @pointerdown="onResizePointerdown('se', $event)"
        >
          <span
            class="pointer-events-none absolute right-1.5 bottom-1.5 size-2 border-r-2 border-b-2 border-muted-foreground/55"
            aria-hidden="true"
          />
        </button>
      </template>
    </div>
  </Teleport>
</template>

