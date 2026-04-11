<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Download, Trash2, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  selectedCount: number
  exportBusy: boolean
  deleteBusy: boolean
  /** false when「按演员」tab：与资料库一致不提供全选可见 */
  showSelectVisible: boolean
  useWebApi: boolean
  exportError: string
}>()

const emit = defineEmits<{
  exit: []
  clearSelection: []
  selectAllVisible: []
  deleteSelected: []
  exportWebp: []
  exportPng: []
}>()

const { t } = useI18n()
</script>

<template>
  <div
    role="toolbar"
    :aria-label="t('curated.batchToolbarAria')"
    class="relative z-30 w-full shrink-0 overflow-hidden border-t border-border bg-card px-3 py-3 shadow-[0_-12px_32px_rgba(0,0,0,0.14)] sm:px-4 dark:shadow-[0_-12px_36px_rgba(0,0,0,0.45)]"
    style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))"
  >
    <div class="flex w-full flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-muted-foreground">
        <span class="font-medium text-foreground">
          {{
            t("curated.exportSelectedCount", {
              n: selectedCount,
              max: 20,
            })
          }}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          :disabled="selectedCount === 0 || exportBusy || deleteBusy"
          @click="emit('clearSelection')"
        >
          {{ t("curated.clearSelection") }}
        </Button>
        <Button
          v-if="showSelectVisible"
          type="button"
          variant="ghost"
          size="sm"
          class="h-8 rounded-lg px-2"
          :disabled="exportBusy || deleteBusy"
          @click="emit('selectAllVisible')"
        >
          {{ t("library.batchSelectVisible") }}
        </Button>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          :disabled="selectedCount === 0 || exportBusy || deleteBusy || !props.useWebApi"
          :title="!props.useWebApi ? t('curated.exportRequiresApi') : undefined"
          @click="emit('exportWebp')"
        >
          <Download class="size-4" />
          {{ exportBusy ? t("curated.exportWorking") : t("curated.exportWebp") }}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          :disabled="selectedCount === 0 || exportBusy || deleteBusy || !props.useWebApi"
          :title="!props.useWebApi ? t('curated.exportRequiresApi') : undefined"
          @click="emit('exportPng')"
        >
          <Download class="size-4" />
          {{ exportBusy ? t("curated.exportWorking") : t("curated.exportPng") }}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl border-destructive/50 text-destructive hover:bg-destructive/10 hover:text-destructive"
          :disabled="selectedCount === 0 || exportBusy || deleteBusy"
          @click="emit('deleteSelected')"
        >
          <Trash2 class="size-4" />
          {{ deleteBusy ? t("curated.deleteWorking") : t("curated.deleteSelectedFrames") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground disabled:opacity-40"
          :disabled="exportBusy || deleteBusy"
          @click="emit('exit')"
        >
          <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
          {{ t("library.batchExit") }}
        </Button>
      </div>
    </div>
    <p v-if="exportError" class="mt-2 text-sm text-destructive" role="alert">
      {{ exportError }}
    </p>
  </div>
</template>
