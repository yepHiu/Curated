<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { Heart, HeartOff, Tag, Trash2, X } from "lucide-vue-next"
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

defineProps<{
  selectedCount: number
  operationBusy: boolean
}>()

const emit = defineEmits<{
  exit: []
  clearSelection: []
  selectAllVisible: []
  addFavorite: []
  removeFavorite: []
  addTag: [tag: string]
  deleteComics: []
}>()

const { t } = useI18n()

const tagDialogOpen = ref(false)
const deleteConfirmOpen = ref(false)
const tagDraft = ref("")
const tagError = ref("")

function openTagDialog() {
  tagDraft.value = ""
  tagError.value = ""
  tagDialogOpen.value = true
}

function submitTagDialog() {
  const tag = tagDraft.value.trim()
  if (!tag) {
    tagError.value = t("comics.batchTagRequired")
    return
  }
  tagDialogOpen.value = false
  emit("addTag", tag)
}

function confirmDelete() {
  deleteConfirmOpen.value = false
  emit("deleteComics")
}
</script>

<template>
  <div
    role="toolbar"
    :aria-label="t('comics.batchToolbarAria')"
    class="w-full shrink-0 overflow-hidden border-t border-border/70 bg-card/95 px-3 py-3 shadow-[0_-8px_30px_rgba(0,0,0,0.12)] backdrop-blur-md sm:px-4"
    style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))"
  >
    <div class="flex w-full flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-muted-foreground">
        <span class="font-medium text-foreground">
          {{ t("comics.batchSelected", { n: selectedCount }) }}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          data-comic-batch-clear-selection
          :disabled="selectedCount === 0 || operationBusy"
          @click="emit('clearSelection')"
        >
          {{ t("comics.batchClearSelection") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          data-comic-batch-select-visible
          :disabled="operationBusy"
          @click="emit('selectAllVisible')"
        >
          {{ t("comics.batchSelectVisible") }}
        </Button>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          data-comic-batch-add-favorite
          :disabled="selectedCount === 0 || operationBusy"
          @click="emit('addFavorite')"
        >
          <Heart class="size-4" />
          {{ t("comics.batchAddFavorite") }}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          data-comic-batch-remove-favorite
          :disabled="selectedCount === 0 || operationBusy"
          @click="emit('removeFavorite')"
        >
          <HeartOff class="size-4" />
          {{ t("comics.batchRemoveFavorite") }}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          data-comic-batch-open-tag
          :disabled="selectedCount === 0 || operationBusy"
          @click="openTagDialog"
        >
          <Tag class="size-4" />
          {{ t("comics.batchAddTag") }}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl text-destructive hover:bg-destructive/10"
          data-comic-batch-open-delete
          :disabled="selectedCount === 0 || operationBusy"
          @click="deleteConfirmOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t("comics.batchDelete") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground disabled:opacity-40"
          data-comic-batch-exit
          :disabled="operationBusy"
          @click="emit('exit')"
        >
          <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
          {{ t("comics.batchExit") }}
        </Button>
      </div>
    </div>

    <Dialog v-model:open="tagDialogOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ t("comics.batchTagDialogTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("comics.batchTagDialogDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <div class="grid gap-2 py-2">
          <p
            v-if="tagError"
            class="text-sm text-destructive"
          >
            {{ tagError }}
          </p>
          <Input
            v-model="tagDraft"
            class="rounded-xl"
            data-comic-batch-tag-input
            :placeholder="t('comics.batchTagPlaceholder')"
            autocomplete="off"
          />
        </div>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            class="rounded-2xl"
            data-comic-batch-submit-tag
            @click="submitTagDialog"
          >
            {{ t("comics.batchTagSubmit") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="deleteConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle v-once>{{ t("comics.batchDeleteConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("comics.batchDeleteConfirmDesc", { n: selectedCount }) }}
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
            data-comic-batch-confirm-delete
            @click="confirmDelete"
          >
            {{ t("comics.batchDeleteConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
