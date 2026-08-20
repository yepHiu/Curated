<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { onKeyStroke } from "@vueuse/core"
import { PanelLeft, Plus, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import type { AIAgentMovieCardDTO, AIChatContextDTO, AIChatMessageDTO, AIChatSessionDTO } from "@/api/types"
import { agentPageContext } from "@/lib/agent-page-context"
import { parsePresentMoviesContent } from "@/lib/agent-movie-cards"
import { isAgentProcessTool } from "@/lib/agent-tool-labels"
import { mentionsStillInText, type AgentMention } from "@/lib/agent-mentions"
import {
  AGENT_WINDOW_CHAT_WIDE_MIN,
  AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH,
  useAgentWindow,
} from "@/composables/use-agent-window"
import { useAIService } from "@/services/ai-service"
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
const streaming = ref(false)
const providerUnconfigured = ref(false)
const errorMessage = ref("")
const threadRef = ref<{ scrollToEnd: () => void } | null>(null)
const composerRef = ref<{ focus: () => void; mentionOpen?: boolean; closeMentions?: () => void } | null>(null)

let abortController: AbortController | null = null
let streamSeq = 0
let entrySeq = 0

const sidebarOverlays = computed(
  () => isMobileViewport.value || size.value.width < AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH,
)

const chatWide = computed(
  () => !isMobileViewport.value && size.value.width >= AGENT_WINDOW_CHAT_WIDE_MIN,
)

const headerTitle = computed(() => {
  const title = sessions.value.find((item) => item.id === sessionId.value)?.title?.trim()
  return title || t("agentWindow.title")
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
  abortController?.abort()
})

watch(open, async (isOpen) => {
  if (isOpen) {
    await refreshSessions()
    await nextTick()
    composerRef.value?.focus()
  }
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
  sessionId.value = id
  if (id) {
    localStorage.setItem(SESSION_STORAGE_KEY, id)
  } else {
    localStorage.removeItem(SESSION_STORAGE_KEY)
  }
}

async function refreshSessions() {
  try {
    sessions.value = await aiService.listSessions()
  } catch {
    sessions.value = []
  }
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
  persistSessionId(id)
  try {
    const detail = await aiService.getSession(id)
    let pendingMovies: AIAgentMovieCardDTO[] = []
    let processTools: Extract<AgentChatEntry, { kind: "process" }>["tools"] = []
    const next: AgentChatEntry[] = []
    const flushProcess = () => {
      if (processTools.length === 0) return
      next.push({
        id: nextEntryId("process"),
        kind: "process",
        thinking: "",
        thinkingActive: false,
        tools: processTools,
        open: false,
      })
      processTools = []
    }
    for (const message of detail.messages) {
      if (message.role === "user") {
        flushProcess()
        next.push({ id: message.id, kind: "user", content: message.content })
        continue
      }
      if (message.role === "assistant") {
        flushProcess()
        next.push({
          id: message.id,
          kind: "assistant",
          content: message.content,
          movies: pendingMovies,
        })
        pendingMovies = []
        continue
      }
      if (message.role === "tool") {
        const movies = message.toolName === "present_movies" ? parsePresentMoviesContent(message.content) : []
        if (movies.length) pendingMovies = movies
        if (isAgentProcessTool(message.toolName || "")) {
          processTools.push({
            toolCallId: message.toolCallId || message.id,
            name: message.toolName || "tool",
            pending: false,
            ok: true,
          })
        }
      }
    }
    flushProcess()
    if (pendingMovies.length) {
      next.push({
        id: nextEntryId("assistant"),
        kind: "assistant",
        content: "",
        movies: pendingMovies,
      })
    }
    entries.value = next
  } catch {
    entries.value = []
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
  abortController?.abort()
  try {
    const created = await aiService.createSession()
    sessions.value = [created, ...sessions.value.filter((item) => item.id !== created.id)]
    persistSessionId(created.id)
  } catch {
    persistSessionId("")
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
    abortController?.abort()
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

function chatContextFrom(text: string, active: AgentMention[]): AIChatContextDTO | undefined {
  const page: AIChatContextDTO = { ...(agentPageContext(route) ?? {}) }
  const kept = mentionsStillInText(active, text).slice(0, 8)
  if (kept.length > 0) {
    page.mentions = kept.map((item) => ({
      kind: item.kind,
      id: item.id,
      label: item.label,
    }))
  }
  return Object.keys(page).length > 0 ? page : undefined
}

function removeAssistantTurn(assistantId: string) {
  const process = findProcessFor(assistantId)
  const index = entries.value.findIndex((entry) => entry.id === assistantId)
  if (index >= 0) removeEntryAt(index)
  if (process) {
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
  abortController?.abort()
}

async function send() {
  const content = draft.value.trim()
  if (!content || streaming.value) return
  draft.value = ""
  const activeMentions = mentions.value
  mentions.value = []
  entries.value.push({ id: nextEntryId("user"), kind: "user", content })

  const history: AIChatMessageDTO[] = entries.value.flatMap((entry) => {
    if (entry.kind === "user" || entry.kind === "assistant") {
      return [{ role: entry.kind, content: entry.content }]
    }
    return []
  })
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
  abortController = new AbortController()
  const seq = ++streamSeq
  void scrollListToEnd()

  try {
    await aiService.streamChat(
      {
        messages: history,
        sessionId: sessionId.value || undefined,
        context: chatContextFrom(content, activeMentions),
        locale: locale.value,
      },
      {
        signal: abortController.signal,
        onSession(id) {
          if (seq !== streamSeq) return
          persistSessionId(id)
        },
        onThinking(delta) {
          if (seq !== streamSeq) return
          const current = findProcessFor(assistantId)
          if (!current) return
          current.thinking += delta
          current.thinkingActive = true
          void scrollListToEnd()
        },
        onDelta(delta) {
          if (seq !== streamSeq) return
          collapseProcess(assistantId)
          const current = entries.value.find((entry) => entry.id === assistantId)
          if (current?.kind === "assistant") {
            current.content += delta
          }
          void scrollListToEnd()
        },
        onToolStart(event) {
          if (seq !== streamSeq) return
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
          if (seq !== streamSeq) return
          const current = findProcessFor(assistantId)
          const card = current?.tools.find((item) => item.toolCallId === event.toolCallId)
          if (card) {
            card.pending = false
            card.ok = event.ok
          }
          if (event.movies?.length) {
            attachMovies(assistantId, event.movies)
          }
          void scrollListToEnd()
        },
        onMovieCards(movies) {
          if (seq !== streamSeq) return
          attachMovies(assistantId, movies)
          void scrollListToEnd()
        },
        onConfirmRequired(event) {
          if (seq !== streamSeq) return
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
      },
    )
    void refreshSessions()
  } catch (err) {
    if (!abortController.signal.aborted) {
      const aiErr = err instanceof AIServiceError ? err : null
      if (aiErr?.code === AI_PROVIDER_UNAVAILABLE_CODE) {
        providerUnconfigured.value = true
        removeAssistantTurn(assistantId)
      } else {
        errorMessage.value = aiErr?.message ?? (err as Error).message ?? t("agentWindow.errorFallback")
        removeAssistantTurn(assistantId)
      }
    }
  } finally {
    if (seq === streamSeq) {
      streaming.value = false
      collapseProcess(assistantId)
      const current = entries.value.find((entry) => entry.id === assistantId)
      if (abortController.signal.aborted && current?.kind === "assistant" && !current.content && !current.movies?.length) {
        removeAssistantTurn(assistantId)
      }
      void scrollListToEnd()
    }
  }
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
      <div
        class="flex min-h-11 cursor-grab items-center gap-1 border-b border-border px-1.5 py-1 text-foreground active:cursor-grabbing md:min-h-10"
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

      <div class="relative flex min-h-0 flex-1">
        <AgentChatSidebar
          v-if="sidebarOpen && !sidebarOverlays"
          :sessions="sessions"
          :active-id="sessionId"
          @create="startNewChat"
          @select="selectSession"
          @delete="deleteChat"
        />
        <div
          class="flex min-h-0 min-w-0 flex-1 flex-col"
          :data-agent-chat-wide="chatWide ? 'true' : 'false'"
        >
          <AgentChatThread
            ref="threadRef"
            :entries="entries"
            :provider-unconfigured="providerUnconfigured"
            :error-message="errorMessage"
            :wide="chatWide"
            @close="close"
            @open-movie="openMovieDetail"
            @apply-confirm="applyConfirm"
            @discard-confirm="discardConfirm"
          />
          <div
            class="mx-auto w-full shrink-0"
            :class="chatWide ? 'max-w-[52rem] px-6' : 'px-4'"
          >
            <AgentChatComposer
              ref="composerRef"
              v-model="draft"
              v-model:mentions="mentions"
              :streaming="streaming"
              @send="send"
              @stop="stop"
            />
          </div>
        </div>
        <template v-if="sidebarOpen && sidebarOverlays">
          <button
            type="button"
            class="absolute inset-0 z-30 bg-background/70"
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
          />
        </template>
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
