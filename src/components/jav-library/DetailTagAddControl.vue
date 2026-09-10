<script setup lang="ts">
import { onClickOutside, useFocusWithin } from "@vueuse/core"
import { computed, nextTick, ref, useId } from "vue"
import { useI18n } from "vue-i18n"
import { Plus, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { useUserTagSuggestKeyboard } from "@/composables/use-user-tag-suggest-keyboard"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"

const props = withDefaults(defineProps<{
  tags: readonly string[]
  suggestions?: readonly string[]
  disabled?: boolean
  saveErrorMessage: string
}>(), { suggestions: () => [] })

// Persistence stays with each library; done keeps the draft available on failure.
const emit = defineEmits<{
  add: [tag: string, done: (error?: unknown) => void]
}>()

const { t } = useI18n()
const draft = ref("")
const error = ref("")
const open = ref(false)
const saving = ref(false)
const inputRef = ref<HTMLInputElement | null>(null)
const inlineZoneRef = ref<HTMLElement | null>(null)
const suggestRootRef = ref<HTMLElement | null>(null)
const suggestListRef = ref<HTMLElement | null>(null)
const domId = useId()
const { focused } = useFocusWithin(suggestRootRef)
const filteredSuggestions = computed(() =>
  filterUserTagSuggestions(props.suggestions, draft.value, new Set(props.tags), { limit: 10 }),
)
const showSuggestions = computed(() =>
  open.value && focused.value && !saving.value && draft.value.trim() !== "" && filteredSuggestions.value.length > 0,
)

function cancel() {
  if (saving.value) return
  open.value = false
  draft.value = ""
  error.value = ""
}

function add(raw: string) {
  if (saving.value || props.disabled) return
  error.value = ""
  const tag = raw.trim()
  if (!tag) return
  if ([...tag].length > 64) {
    error.value = t("curated.tagMaxRunes", { n: 64 })
    return
  }
  if (props.tags.includes(tag)) {
    draft.value = ""
    return
  }
  if (props.tags.length >= 64) {
    error.value = t("curated.tagMaxCount", { n: 64 })
    return
  }
  saving.value = true
  emit("add", tag, (failure?: unknown) => {
    saving.value = false
    if (failure) {
      error.value = failure instanceof Error && failure.message.trim()
        ? failure.message : props.saveErrorMessage
      return
    }
    draft.value = ""
  })
}

async function onAddClick() {
  if (saving.value || props.disabled) return
  error.value = ""
  if (!open.value) {
    open.value = true
    await nextTick()
    inputRef.value?.focus()
    return
  }
  add(draft.value)
}

const { highlightIndex, onTagSuggestKeydown } = useUserTagSuggestKeyboard({
  showSuggestions,
  suggestions: filteredSuggestions,
  listRootRef: suggestListRef,
  commitTag: add,
  commitDraft: () => add(draft.value),
})

function onKeydown(event: KeyboardEvent) {
  if (event.isComposing || saving.value || props.disabled) return
  if (event.key === "Escape") {
    event.preventDefault()
    cancel()
    return
  }
  onTagSuggestKeydown(event)
}

onClickOutside(inlineZoneRef, cancel)
</script>

<template>
  <div ref="inlineZoneRef" class="flex max-w-full flex-wrap items-center gap-2" :aria-busy="saving">
    <Button
      type="button"
      variant="secondary"
      class="h-[29px] shrink-0 rounded-2xl px-3 py-0 text-xs leading-none"
      data-detail-add-tag
      :disabled="disabled || saving"
      @click="onAddClick"
    >
      <Plus class="size-3.5 shrink-0" data-icon="inline-start" />
      {{ t("common.add") }}
    </Button>
    <div v-if="open" ref="suggestRootRef" class="relative max-w-full min-w-[min(100%,12rem)]">
      <div class="flex h-9 w-full items-center gap-0.5 rounded-2xl border border-border/80 bg-background/80 pl-3 pr-0.5 shadow-sm">
        <input
          ref="inputRef"
          v-model="draft"
          data-detail-new-tag-input
          type="text"
          autocomplete="off"
          role="combobox"
          :disabled="disabled || saving"
          :aria-expanded="showSuggestions"
          :aria-activedescendant="highlightIndex >= 0 ? `${domId}-opt-${highlightIndex}` : undefined"
          aria-autocomplete="list"
          :aria-controls="showSuggestions ? `${domId}-list` : undefined"
          :aria-invalid="Boolean(error)"
          :aria-describedby="error ? `${domId}-error` : undefined"
          :aria-label="t('detailPanel.newTagPlaceholder')"
          :placeholder="t('detailPanel.newTagPlaceholder')"
          class="placeholder:text-muted-foreground h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0"
          @keydown="onKeydown"
        >
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="size-8 shrink-0 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
          :disabled="saving"
          :aria-label="t('detailPanel.ariaCancelTagInput')"
          @click="cancel"
        >
          <X class="size-4" />
        </Button>
      </div>
      <ul
        v-if="showSuggestions"
        :id="`${domId}-list`"
        ref="suggestListRef"
        class="absolute top-full left-0 z-50 mt-1 max-h-60 w-full min-w-[min(100%,12rem)] overflow-y-auto rounded-2xl border border-border/80 bg-popover/98 py-1 text-popover-foreground shadow-lg backdrop-blur-sm"
        role="listbox"
        :aria-label="t('detailPanel.tagSuggestAria')"
      >
        <li v-for="(suggestion, index) in filteredSuggestions" :key="suggestion">
          <button
            :id="`${domId}-opt-${index}`"
            type="button"
            role="option"
            :data-tag-suggest-idx="index"
            class="w-full truncate px-3 py-2 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
            :class="highlightIndex === index ? 'bg-muted' : ''"
            :aria-selected="highlightIndex === index"
            @mousedown.prevent="add(suggestion)"
          >
            {{ suggestion }}
          </button>
        </li>
      </ul>
    </div>
    <p v-if="error" :id="`${domId}-error`" role="alert" class="basis-full text-sm text-destructive">{{ error }}</p>
  </div>
</template>
