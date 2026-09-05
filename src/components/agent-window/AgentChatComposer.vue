<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { AtSign, Send, Square } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  applyMention,
  mentionPickerLimit,
  mentionQueryAtCursor,
  mentionsStillInText,
  searchActorMentions,
  searchMovieMentions,
  searchTagMentions,
  type AgentMention,
  type AgentMentionKind,
} from "@/lib/agent-mentions"
import { useLibraryService } from "@/services/library-service"

defineOptions({ name: "AgentChatComposer" })

const props = defineProps<{
  streaming: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  send: []
  stop: []
}>()

const draft = defineModel<string>({ required: true })
const mentions = defineModel<AgentMention[]>("mentions", { default: () => [] })
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const mentionOpen = computed(() => {
  if (pickerDismissed.value) return false
  return Boolean(
    mentionQueryAtCursor(draft.value, cursor.value) ??
      mentionQueryAtCursor(draft.value, draft.value.length),
  )
})
const pickerDismissed = ref(false)
const cursor = ref(0)
const highlight = ref(0)
const actorHits = ref<AgentMention[]>([])
const syncingMention = ref(false)
const { t } = useI18n()
const library = useLibraryService()

const catalog = computed(() => {
  const movies = library.movies
  if (Array.isArray(movies)) return movies
  const value = movies && "value" in movies ? movies.value : []
  return Array.isArray(value) ? value : []
})
const mentionQuery = computed(() => mentionQueryAtCursor(draft.value, cursor.value))
const pickerLimit = computed(() => mentionPickerLimit(mentionQuery.value?.query ?? ""))
const movieHits = computed(() =>
  searchMovieMentions(catalog.value, mentionQuery.value?.query ?? "", pickerLimit.value),
)
const tagHits = computed(() =>
  searchTagMentions(catalog.value, mentionQuery.value?.query ?? "", pickerLimit.value),
)

const groups = computed(() => {
  const next: { kind: AgentMentionKind; items: AgentMention[] }[] = []
  if (movieHits.value.length) next.push({ kind: "movie", items: movieHits.value })
  if (actorHits.value.length) next.push({ kind: "actor", items: actorHits.value })
  if (tagHits.value.length) next.push({ kind: "tag", items: tagHits.value })
  return next
})

const flatItems = computed(() => groups.value.flatMap((group) => group.items))

watch(
  () => [mentionOpen.value, mentionQuery.value?.query ?? ""] as const,
  async ([open, query]) => {
    if (!open) {
      actorHits.value = []
      return
    }
    try {
      const limit = mentionPickerLimit(query)
      const listed = await library.listActors({ q: query, limit })
      actorHits.value = searchActorMentions(listed.actors ?? [], query, limit)
    } catch {
      actorHits.value = []
    }
  },
)

watch(flatItems, (items) => {
  if (highlight.value >= items.length) {
    highlight.value = Math.max(0, items.length - 1)
  }
})

watch(
  () => draft.value,
  () => {
    if (syncingMention.value) return
    syncFromTextarea()
  },
)

defineExpose({
  mentionOpen,
  closeMentions,
  focus() {
    textareaRef.value?.focus()
  },
})

function syncFromTextarea(event?: Event) {
  const live = event?.target instanceof HTMLTextAreaElement ? event.target.value : draft.value
  const el = event?.target instanceof HTMLTextAreaElement ? event.target : textareaRef.value
  const caret = el?.selectionStart
  const fromCaret = typeof caret === "number" ? mentionQueryAtCursor(live, caret) : null
  const fromEnd = mentionQueryAtCursor(live, live.length)
  const found = fromCaret ?? fromEnd
  if (typeof caret === "number" && caret > 0) {
    cursor.value = caret
  } else if (found) {
    cursor.value = found.start + 1 + found.query.length
  }
  mentions.value = mentionsStillInText(mentions.value, live)
  if (!found) {
    pickerDismissed.value = false
  }
}

function closeMentions() {
  pickerDismissed.value = true
}

function choose(item: AgentMention) {
  const next = applyMention(draft.value, cursor.value, item)
  syncingMention.value = true
  draft.value = next.text
  mentions.value = mentionsStillInText([...mentions.value, item], next.text)
  pickerDismissed.value = true
  void nextTick(() => {
    const el = textareaRef.value
    if (el) {
      el.focus()
      el.setSelectionRange(next.cursor, next.cursor)
    }
    cursor.value = next.cursor
    syncingMention.value = false
  })
}

function insertMentionTrigger() {
  if (mentionOpen.value) {
    closeMentions()
    return
  }
  const el = textareaRef.value
  const at = el?.selectionStart ?? draft.value.length
  const found = mentionQueryAtCursor(draft.value, at)
  if (found) {
    pickerDismissed.value = false
    cursor.value = at
    return
  }
  const before = draft.value.slice(0, at)
  const token = before.length > 0 && !/\s$/.test(before) ? " @" : "@"
  const next = `${before}${token}${draft.value.slice(at)}`
  const nextCursor = at + token.length
  syncingMention.value = true
  draft.value = next
  cursor.value = nextCursor
  pickerDismissed.value = false
  void nextTick(() => {
    const field = textareaRef.value
    if (field) {
      field.focus()
      field.setSelectionRange(nextCursor, nextCursor)
    }
    cursor.value = nextCursor
    syncingMention.value = false
  })
}

function onKeydown(e: KeyboardEvent) {
  if (mentionOpen.value) {
    if (e.key === "Escape") {
      e.preventDefault()
      e.stopPropagation()
      closeMentions()
      return
    }
    if (flatItems.value.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        highlight.value = (highlight.value + 1) % flatItems.value.length
        return
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        highlight.value = (highlight.value - 1 + flatItems.value.length) % flatItems.value.length
        return
      }
      if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
        e.preventDefault()
        const item = flatItems.value[highlight.value]
        if (item) choose(item)
        return
      }
    }
  }
  if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    if (!props.streaming && !props.disabled && draft.value.trim()) {
      emit("send")
    }
  }
}
</script>

<template>
  <div class="px-0 pb-3 pt-1" :data-agent-mention-open="mentionOpen ? 'true' : 'false'">
    <div
      class="relative rounded-2xl border border-border/60 bg-muted/40 p-2 text-foreground shadow-sm outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-ring/50 focus-within:ring-[3px]"
      data-agent-window-composer
    >
      <div
        v-if="mentionOpen"
        class="absolute inset-x-2 bottom-full z-20 mb-1 max-h-44 overflow-y-auto rounded-xl border border-border bg-popover py-1 text-popover-foreground shadow-md"
        data-agent-mention-picker
      >
        <div v-if="flatItems.length === 0" class="px-3 py-2 text-xs text-muted-foreground">
          {{ t("agentWindow.mention.empty") }}
        </div>
        <template v-else>
          <button
            v-for="item in flatItems"
            :key="`${item.kind}:${item.id}`"
            type="button"
            class="flex h-8 w-full items-center gap-2 px-2.5 text-left text-[13px] text-popover-foreground hover:bg-accent hover:text-accent-foreground"
            :class="flatItems[highlight]?.id === item.id && flatItems[highlight]?.kind === item.kind ? 'bg-accent text-accent-foreground' : ''"
            :data-agent-mention-item="`${item.kind}:${item.id}`"
            @mousedown.prevent="choose(item)"
          >
            <span class="w-8 shrink-0 text-[10px] text-muted-foreground">
              {{ t(`agentWindow.mention.${item.kind}`) }}
            </span>
            <span class="min-w-0 truncate">{{ item.label }}</span>
          </button>
        </template>
      </div>
      <textarea
        ref="textareaRef"
        v-model="draft"
        rows="2"
        class="min-h-11 w-full resize-none border-0 bg-transparent px-2 py-1.5 text-sm leading-relaxed text-foreground outline-none placeholder:text-muted-foreground disabled:opacity-60"
        :placeholder="t('agentWindow.inputPlaceholder')"
        :disabled="streaming || disabled"
        data-agent-window-input
        @keydown="onKeydown"
        @input="syncFromTextarea"
        @click="syncFromTextarea"
        @keyup="syncFromTextarea"
      />
      <div class="flex items-center justify-between px-1 pb-0.5">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="size-11 rounded-lg text-muted-foreground hover:text-foreground md:size-8"
          :disabled="streaming || disabled"
          :aria-label="t('agentWindow.mention.trigger')"
          data-agent-mention-trigger
          @click="insertMentionTrigger"
        >
          <AtSign class="size-4" />
        </Button>
        <Button
          type="button"
          size="icon"
          class="size-11 rounded-lg md:size-8"
          :variant="streaming ? 'secondary' : 'default'"
          :disabled="disabled || (!streaming && !draft.trim())"
          :aria-label="streaming ? t('agentWindow.stop') : t('agentWindow.send')"
          data-agent-window-send
          @click="streaming ? emit('stop') : emit('send')"
        >
          <Square v-if="streaming" class="size-3 fill-current" />
          <Send v-else class="size-4" />
        </Button>
      </div>
    </div>
  </div>
</template>
