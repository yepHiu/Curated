<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Download, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  selectedCount: number
  exportBusy: boolean
  /** false when「按演员」tab：与资料库一致不提供全选可见 */
  showSelectVisible: boolean
  useWebApi: boolean
  exportError: string
}>()

const emit = defineEmits<{
  exit: []
  clearSelection: []
  selectAllVisible: []
  exportWebp: []
  exportPng: []
}>()

const { t } = useI18n()
</script>

<template>
  <div
    role="toolbar"
    :aria-label="t('curated.batchToolbarAria')"
    class="w-full shrink-0 overflow-hidden border-t border-border/70 bg-card/95 px-3 py-3 shadow-[0_-8px_30px_rgba(0,0,0,0.12)] backdrop-blur-md sm:px-4 rounded-b-[calc(2rem-1rem)] lg:rounded-b-[calc(2rem-1.25rem)] xl:rounded-b-[calc(2rem-1.5rem)]"
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
          :disabled="selectedCount === 0 || exportBusy"
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
          :disabled="exportBusy"
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
          :disabled="selectedCount === 0 || exportBusy || !props.useWebApi"
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
          :disabled="selectedCount === 0 || exportBusy || !props.useWebApi"
          :title="!props.useWebApi ? t('curated.exportRequiresApi') : undefined"
          @click="emit('exportPng')"
        >
          <Download class="size-4" />
          {{ exportBusy ? t("curated.exportWorking") : t("curated.exportPng") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground disabled:opacity-40"
          :disabled="exportBusy"
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
