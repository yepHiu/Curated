<script setup lang="ts">
import { useFocusWithin } from "@vueuse/core"
import { computed, nextTick, ref, useId, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Heart, HeartOff, RefreshCw, RotateCcw, Tag, Trash2, X } from "lucide-vue-next"
import type { LibraryMode } from "@/domain/library/types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { useUserTagSuggestKeyboard } from "@/composables/use-user-tag-suggest-keyboard"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"

const props = withDefaults(
  defineProps<{
    mode: LibraryMode
    selectedCount: number
    useWebApi: boolean
    scrapeProgress: { current: number; total: number } | null
    scrapeBusy: boolean
    operationBusy: boolean
    userTagSuggestions?: readonly string[]
  }>(),
  {
    userTagSuggestions: () => [],
  },
)

const emit = defineEmits<{
  exit: []
  clearSelection: []
  selectAllVisible: []
  addFavorite: []
  removeFavorite: []
  addUserTag: [tag: string]
  refreshMetadata: []
  moveToTrash: []
  restore: []
  permanentDelete: []
}>()

const { t } = useI18n()

const isTrash = () => props.mode === "trash"

const refreshConfirmOpen = ref(false)
const trashConfirmOpen = ref(false)
const permanentConfirmOpen = ref(false)
const restoreConfirmOpen = ref(false)
const tagDialogOpen = ref(false)
const tagDraft = ref("")
const tagError = ref("")
const tagInputRef = ref<{ focus?: () => void; $el?: { focus?: () => void } } | null>(null)
const tagSuggestRootRef = ref<HTMLElement | null>(null)
const tagSuggestListRef = ref<HTMLElement | null>(null)
const tagSuggestDomId = useId()
const { focused: tagSuggestFocused } = useFocusWithin(tagSuggestRootRef)

const filteredTagSuggestions = computed(() =>
  filterUserTagSuggestions(props.userTagSuggestions, tagDraft.value, [], { limit: 10 }),
)

const showTagSuggestions = computed(
  () =>
    tagDialogOpen.value &&
    tagSuggestFocused.value &&
    tagDraft.value.trim() !== "" &&
    filteredTagSuggestions.value.length > 0,
)

function focusTagInput() {
  const target = tagInputRef.value
  if (typeof target?.focus === "function") {
    target.focus()
    return
  }
  if (typeof target?.$el?.focus === "function") {
    target.$el.focus()
  }
}

watch(tagDialogOpen, (open) => {
  if (!open) {
    tagDraft.value = ""
    tagError.value = ""
    return
  }
  void nextTick(() => focusTagInput())
})

function openTagDialog() {
  tagError.value = ""
  tagDraft.value = ""
  tagDialogOpen.value = true
}

function submitTagDialog(raw?: unknown) {
  const tag = typeof raw === "string" ? raw.trim() : tagDraft.value.trim()
  if (!tag) {
    tagError.value = t("library.batchTagRequired")
    return
  }
  tagDialogOpen.value = false
  emit("addUserTag", tag)
}

function pickTagSuggestion(tag: string) {
  submitTagDialog(tag)
}

const { highlightIndex, onTagSuggestKeydown } = useUserTagSuggestKeyboard({
  showSuggestions: showTagSuggestions,
  suggestions: filteredTagSuggestions,
  listRootRef: tagSuggestListRef,
  commitTag: (tag) => submitTagDialog(tag),
  commitDraft: () => submitTagDialog(),
})
</script>

<template>
  <div
    role="toolbar"
    :aria-label="t('library.batchToolbarAria')"
    class="w-full shrink-0 overflow-hidden border-t border-border/70 bg-card/95 px-3 py-3 shadow-[0_-8px_30px_rgba(0,0,0,0.12)] backdrop-blur-md sm:px-4"
    style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))"
  >
    <div class="flex w-full flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-muted-foreground">
        <span class="font-medium text-foreground">
          {{ t("library.batchSelected", { n: selectedCount }) }}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
          @click="emit('clearSelection')"
        >
          {{ t("library.batchClearSelection") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          :disabled="operationBusy || scrapeBusy"
          @click="emit('selectAllVisible')"
        >
          {{ t("library.batchSelectVisible") }}
        </Button>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <template v-if="isTrash()">
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
            @click="restoreConfirmOpen = true"
          >
            <RotateCcw class="size-4" />
            {{ t("library.batchRestore") }}
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            class="gap-1.5 rounded-xl"
            :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
            @click="permanentConfirmOpen = true"
          >
            <Trash2 class="size-4" />
            {{ t("library.batchDeletePermanently") }}
          </Button>
        </template>
        <template v-else>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
            @click="emit('addFavorite')"
          >
            <Heart class="size-4" />
            {{ t("library.batchAddFavorite") }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
            @click="emit('removeFavorite')"
          >
            <HeartOff class="size-4" />
            {{ t("library.batchRemoveFavorite") }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
            @click="openTagDialog"
          >
            <Tag class="size-4" />
            {{ t("library.batchAddTag") }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            :disabled="selectedCount === 0 || !useWebApi || operationBusy || scrapeBusy"
            :title="!useWebApi ? t('detail.refreshMockMode') : undefined"
            @click="refreshConfirmOpen = true"
          >
            <RefreshCw
              class="size-4"
              :class="scrapeBusy ? 'animate-spin' : ''"
            />
            {{
              scrapeBusy && scrapeProgress
                ? t("library.batchScrapeProgress", {
                    current: scrapeProgress.current,
                    total: scrapeProgress.total,
                  })
                : t("library.batchRefreshMetadata")
            }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl text-destructive hover:bg-destructive/10"
            :disabled="selectedCount === 0 || operationBusy || scrapeBusy"
            @click="trashConfirmOpen = true"
          >
            <Trash2 class="size-4" />
            {{ t("library.batchMoveToTrash") }}
          </Button>
        </template>

        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground disabled:opacity-40"
          :disabled="scrapeBusy"
          @click="emit('exit')"
        >
          <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
          {{ t("library.batchExit") }}
        </Button>
      </div>
    </div>

    <Dialog v-model:open="refreshConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("library.batchRefreshConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("library.batchRefreshConfirmDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            class="rounded-2xl"
            @click="
              refreshConfirmOpen = false;
              emit('refreshMetadata');
            "
          >
            {{ t("library.batchRefreshConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="trashConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("library.batchTrashConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("library.batchTrashConfirmDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            class="rounded-2xl"
            @click="
              trashConfirmOpen = false;
              emit('moveToTrash');
            "
          >
            {{ t("library.batchTrashConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="permanentConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("library.batchPermanentConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("library.batchPermanentConfirmDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            class="rounded-2xl"
            @click="
              permanentConfirmOpen = false;
              emit('permanentDelete');
            "
          >
            {{ t("library.batchPermanentConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="restoreConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("library.batchRestoreConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("library.batchRestoreConfirmDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            class="rounded-2xl"
            @click="
              restoreConfirmOpen = false;
              emit('restore');
            "
          >
            {{ t("library.batchRestoreConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="tagDialogOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ t("library.batchTagDialogTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("library.batchTagDialogDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-2 py-2">
          <p
            v-if="tagError"
            class="text-sm text-destructive"
          >
            {{ tagError }}
          </p>
          <div ref="tagSuggestRootRef" class="relative">
            <Input
              :id="`${tagSuggestDomId}-input`"
              ref="tagInputRef"
              v-model="tagDraft"
              class="rounded-xl"
              :placeholder="t('library.batchTagPlaceholder')"
              autocomplete="off"
              role="combobox"
              :aria-expanded="showTagSuggestions"
              :aria-activedescendant="
                highlightIndex >= 0 ? `${tagSuggestDomId}-opt-${highlightIndex}` : undefined
              "
              aria-autocomplete="list"
              :aria-controls="showTagSuggestions ? `${tagSuggestDomId}-list` : undefined"
              @keydown="onTagSuggestKeydown"
            />
            <ul
              v-if="showTagSuggestions"
              :id="`${tagSuggestDomId}-list`"
              ref="tagSuggestListRef"
              class="absolute top-full left-0 z-50 mt-1 max-h-60 w-full overflow-y-auto rounded-2xl border border-border/80 bg-popover/98 py-1 text-popover-foreground shadow-lg backdrop-blur-sm"
              role="listbox"
              :aria-label="t('detailPanel.tagSuggestAria')"
            >
              <li v-for="(tag, index) in filteredTagSuggestions" :key="tag">
                <button
                  :id="`${tagSuggestDomId}-opt-${index}`"
                  type="button"
                  role="option"
                  :data-tag-suggest-idx="index"
                  class="w-full truncate px-3 py-2 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
                  :class="highlightIndex === index ? 'bg-muted' : ''"
                  :aria-selected="highlightIndex === index"
                  @mousedown.prevent="pickTagSuggestion(tag)"
                >
                  {{ tag }}
                </button>
              </li>
            </ul>
          </div>
        </div>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button type="button" class="rounded-2xl" @click="submitTagDialog()">
            {{ t("library.batchTagSubmit") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
