<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { Heart, HeartOff, Tag, Trash2 } from "lucide-vue-next"
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

const props = defineProps<{
  kind: "photos" | "comics"
  selectedCount: number
  operationBusy: boolean
}>()

const emit = defineEmits<{
  clearSelection: []
  addFavorite: []
  removeFavorite: []
  addTag: [tag: string]
  deleteSelection: []
}>()

const { t } = useI18n()

function label(key: string, values?: Record<string, number>) {
  return values ? t(`${props.kind}.${key}`, values) : t(`${props.kind}.${key}`)
}

function testId(action: string) {
  return { [`data-${props.kind === "photos" ? "photo" : "comic"}-batch-${action}`]: "" }
}

const actions = [
  { id: "add-favorite", key: "batchAddFavorite", icon: Heart, run: () => emit("addFavorite") },
  { id: "remove-favorite", key: "batchRemoveFavorite", icon: HeartOff, run: () => emit("removeFavorite") },
  { id: "open-tag", key: "batchAddTag", icon: Tag, run: () => openTagDialog() },
  { id: "open-delete", key: "batchDelete", icon: Trash2, run: () => { dialog.value = "delete" } },
]

const dialog = ref<"tag" | "delete" | null>(null)
const tagDraft = ref("")
const tagError = ref("")

/** 打开批量追加标签对话框并清空上次草稿。 */
function openTagDialog() {
  tagDraft.value = ""
  tagError.value = ""
  dialog.value = "tag"
}

/** 校验标签后交给父级写入所选写真集。 */
function submitTagDialog() {
  const tag = tagDraft.value.trim()
  if (!tag) {
    tagError.value = label("batchTagRequired")
    return
  }
  dialog.value = null
  emit("addTag", tag)
}

/** 确认后删除所选写真集索引。 */
function confirmDelete() {
  dialog.value = null
  emit("deleteSelection")
}
</script>

<template>
  <div
    role="toolbar"
    :aria-label="label('batchToolbarAria')"
    class="w-full shrink-0 overflow-hidden border-t border-border/70 bg-card/95 px-3 py-3 shadow-[0_-8px_30px_rgba(0,0,0,0.12)] backdrop-blur-md sm:px-4"
    style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))"
  >
    <div class="flex w-full flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-muted-foreground">
        <span class="font-medium text-foreground">
          {{ label("batchSelected", { n: selectedCount }) }}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          v-bind="testId('clear-selection')"
          :disabled="selectedCount === 0 || operationBusy"
          @click="emit('clearSelection')"
        >
          {{ label("batchClearSelection") }}
        </Button>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          v-for="action in actions"
          :key="action.id"
          type="button"
          variant="outline"
          size="sm"
          :class="['gap-1.5 rounded-xl', action.id === 'open-delete' && 'text-destructive hover:bg-destructive/10']"
          v-bind="testId(action.id)"
          :disabled="selectedCount === 0 || operationBusy"
          @click="action.run"
        >
          <component :is="action.icon" class="size-4" />
          {{ label(action.key) }}
        </Button>
      </div>
    </div>

    <Dialog :open="dialog !== null" @update:open="dialog = $event ? dialog : null">
      <DialogContent :class="['rounded-3xl border-border/70', dialog === 'tag' ? 'sm:max-w-lg' : 'sm:max-w-md']">
        <DialogHeader>
          <DialogTitle>{{ label(dialog === "tag" ? "batchTagDialogTitle" : "batchDeleteConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ label(dialog === "tag" ? "batchTagDialogDesc" : "batchDeleteConfirmDesc", { n: selectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <div v-if="dialog === 'tag'" class="grid gap-2 py-2">
          <p
            v-if="tagError"
            class="text-sm text-destructive"
          >
            {{ tagError }}
          </p>
          <Input
            v-model="tagDraft"
            class="rounded-xl"
            v-bind="testId('tag-input')"
            :placeholder="label('batchTagPlaceholder')"
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
            :variant="dialog === 'tag' ? 'default' : 'destructive'"
            class="rounded-2xl"
            v-bind="testId(dialog === 'tag' ? 'submit-tag' : 'confirm-delete')"
            @click="dialog === 'tag' ? submitTagDialog() : confirmDelete()"
          >
            {{ label(dialog === "tag" ? "batchTagSubmit" : "batchDeleteConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
