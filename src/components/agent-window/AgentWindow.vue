<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { onKeyStroke } from "@vueuse/core"
import { Bot, Loader2, Send, Settings, X } from "lucide-vue-next"
import { RouterLink } from "vue-router"
import { Button } from "@/components/ui/button"
import type { AIChatMessageDTO } from "@/api/types"
import { useAgentWindow, AGENT_WINDOW_WIDTH, AGENT_WINDOW_HEIGHT } from "@/composables/use-agent-window"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

interface ChatEntry {
  role: "user" | "assistant"
  content: string
}

const AI_PROVIDER_UNAVAILABLE_CODE = "AI_PROVIDER_UNAVAILABLE"

const { t } = useI18n()
const aiService = useAIService()
const { open, position, isMobileViewport, closeWindow, moveTo } = useAgentWindow()

const entries = ref<ChatEntry[]>([])
const draft = ref("")
const streaming = ref(false)
const providerUnconfigured = ref(false)
const errorMessage = ref("")
const listRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLTextAreaElement | null>(null)

let abortController: AbortController | null = null
let streamSeq = 0

onKeyStroke("Escape", (e) => {
  if (open.value) {
    e.preventDefault()
    close()
  }
})

onBeforeUnmount(() => {
  abortController?.abort()
})

watch(open, async (isOpen) => {
  if (isOpen) {
    await nextTick()
    inputRef.value?.focus()
  }
})

function close() {
  // 关闭窗口时中止进行中的流式请求（P-07 降级：不留僵尸请求）
  abortController?.abort()
  closeWindow()
}

async function scrollListToEnd() {
  await nextTick()
  if (listRef.value) {
    listRef.value.scrollTop = listRef.value.scrollHeight
  }
}

function removeEntryAt(index: number) {
  entries.value.splice(index, 1)
}

async function send() {
  const content = draft.value.trim()
  if (!content || streaming.value) return
  draft.value = ""
  entries.value.push({ role: "user", content })

  const history: AIChatMessageDTO[] = entries.value.map((entry) => ({
    role: entry.role,
    content: entry.content,
  }))
  entries.value.push({ role: "assistant", content: "" })
  const assistantIndex = entries.value.length - 1

  streaming.value = true
  providerUnconfigured.value = false
  errorMessage.value = ""
  abortController = new AbortController()
  const seq = ++streamSeq
  void scrollListToEnd()

  try {
    await aiService.streamChat(history, {
      signal: abortController.signal,
      onDelta(delta) {
        if (seq !== streamSeq) return
        entries.value[assistantIndex]!.content += delta
        void scrollListToEnd()
      },
    })
  } catch (err) {
    if (!abortController.signal.aborted) {
      const aiErr = err instanceof AIServiceError ? err : null
      if (aiErr?.code === AI_PROVIDER_UNAVAILABLE_CODE) {
        providerUnconfigured.value = true
        removeEntryAt(assistantIndex)
      } else {
        errorMessage.value = aiErr?.message ?? (err as Error).message ?? t("agentWindow.errorFallback")
        removeEntryAt(assistantIndex)
      }
    }
  } finally {
    if (seq === streamSeq) {
      streaming.value = false
      // 流被中止且占位气泡仍为空时移除，避免残留空气泡
      if (abortController.signal.aborted && !entries.value[assistantIndex]?.content) {
        removeEntryAt(assistantIndex)
      }
      void scrollListToEnd()
    }
  }
}

function onComposerKeydown(e: KeyboardEvent) {
  if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    void send()
  }
}

// —— 拖拽（标题栏；移动端全屏态不拖拽）——
const dragging = ref(false)

function onHeaderPointerdown(e: PointerEvent) {
  if (isMobileViewport.value || e.button !== 0) return
  const target = e.target as HTMLElement
  if (target.closest("button, a")) return
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

const windowStyle = ref<Record<string, string>>({})
watch(
  [position, isMobileViewport],
  () => {
    windowStyle.value = isMobileViewport.value
      ? {}
      : {
          left: `${position.value.x}px`,
          top: `${position.value.y}px`,
          width: `${AGENT_WINDOW_WIDTH}px`,
          height: `min(${AGENT_WINDOW_HEIGHT}px, calc(100dvh - 24px))`,
        }
  },
  { immediate: true },
)
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed z-[120] flex flex-col overflow-hidden rounded-2xl border border-border/80 bg-popover/98 text-popover-foreground shadow-2xl shadow-black/40 backdrop-blur-md max-md:inset-2"
      :class="dragging ? 'select-none' : ''"
      :style="windowStyle"
      role="dialog"
      aria-label="Curated Agent"
      data-agent-window
    >
      <div
        class="flex min-h-11 cursor-grab items-center gap-2 border-b border-border/60 px-3 py-2 active:cursor-grabbing"
        :class="isMobileViewport ? '' : 'touch-none'"
        data-agent-window-header
        @pointerdown="onHeaderPointerdown"
      >
        <span
          class="flex size-7 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <Bot class="size-4" />
        </span>
        <p class="min-w-0 flex-1 truncate text-sm font-semibold">{{ t("agentWindow.title") }}</p>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="size-9 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
          :aria-label="t('agentWindow.close')"
          @click="close"
        >
          <X class="size-4" />
        </Button>
      </div>

      <div
        ref="listRef"
        class="min-h-0 flex-1 space-y-2.5 overflow-y-auto overscroll-contain px-3 py-3"
        data-agent-window-messages
        aria-live="polite"
      >
        <p
          v-if="entries.length === 0"
          class="px-1 py-6 text-center text-xs leading-relaxed text-muted-foreground"
        >
          {{ t("agentWindow.emptyHint") }}
        </p>
        <div
          v-for="(entry, i) in entries"
          :key="`entry-${i}`"
          class="flex"
          :class="entry.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <p
            class="max-w-[85%] rounded-2xl px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap break-words"
            :class="
              entry.role === 'user'
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted/70 text-foreground'
            "
            :data-agent-entry="entry.role"
          >
            <template v-if="entry.content">{{ entry.content }}</template>
            <Loader2
              v-else
              class="size-4 motion-safe:animate-spin text-muted-foreground"
              aria-hidden="true"
            />
          </p>
        </div>

        <div
          v-if="providerUnconfigured"
          class="flex flex-col gap-2 rounded-2xl border border-border/60 bg-muted/10 px-3 py-2.5 text-sm"
          data-agent-window-unconfigured
        >
          <p class="leading-relaxed text-muted-foreground">{{ t("agentWindow.unconfigured") }}</p>
          <Button as-child variant="outline" size="sm" class="w-fit rounded-full">
            <RouterLink :to="{ name: 'settings', query: { section: 'experimental' } }" @click="close">
              <Settings data-icon="inline-start" class="size-4" />
              {{ t("agentWindow.openSettings") }}
            </RouterLink>
          </Button>
        </div>
        <p
          v-else-if="errorMessage"
          class="rounded-2xl border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm leading-relaxed text-destructive"
          data-agent-window-error
        >
          {{ t("agentWindow.errorPrefix") }}{{ errorMessage }}
        </p>
      </div>

      <div class="border-t border-border/60 p-2.5">
        <div class="flex items-end gap-2">
          <textarea
            ref="inputRef"
            v-model="draft"
            rows="2"
            class="min-h-11 flex-1 resize-none rounded-xl border border-border/60 bg-background/70 px-3 py-2 text-sm leading-relaxed outline-none transition-colors placeholder:text-muted-foreground focus:border-primary/60 focus:ring-1 focus:ring-primary/30 disabled:opacity-60"
            :placeholder="t('agentWindow.inputPlaceholder')"
            :disabled="streaming"
            data-agent-window-input
            @keydown="onComposerKeydown"
          />
          <Button
            type="button"
            size="icon"
            class="size-11 shrink-0 rounded-xl"
            :disabled="streaming || !draft.trim()"
            :aria-label="t('agentWindow.send')"
            data-agent-window-send
            @click="send"
          >
            <Loader2 v-if="streaming" class="size-4 motion-safe:animate-spin" aria-hidden="true" />
            <Send v-else class="size-4" />
          </Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
