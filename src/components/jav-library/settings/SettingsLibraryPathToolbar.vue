<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, Sparkles, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"

defineProps<{
  batchMode: boolean
  libraryPathsCount: number
  hasMetadataPathSelection: boolean
  metadataRefreshBusy: boolean
}>()

const emit = defineEmits<{
  enterBatchMode: []
  selectAll: []
  clearSelection: []
  refreshMetadata: []
  exitBatchMode: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div class="flex min-w-0 flex-1 flex-col gap-3">
      <p class="font-medium">{{ t("settings.libraryPaths") }}</p>
    </div>
    <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
      <template v-if="!batchMode">
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-full"
          data-library-paths-enter-batch
          @click="emit('enterBatchMode')"
        >
          <ListChecks class="size-4 opacity-80" aria-hidden="true" />
          {{ t("library.batchManage") }}
        </Button>
      </template>
      <template v-else>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-full"
          :disabled="libraryPathsCount === 0"
          data-library-paths-select-all
          @click="emit('selectAll')"
        >
          <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
          {{ t("settings.selectAllPaths") }}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-full"
          data-library-paths-clear-selection
          @click="emit('clearSelection')"
        >
          {{ t("settings.clearSelection") }}
        </Button>
        <Button
          type="button"
          size="sm"
          class="gap-1.5 rounded-full"
          :disabled="!hasMetadataPathSelection || metadataRefreshBusy"
          data-library-paths-refresh-metadata
          @click="emit('refreshMetadata')"
        >
          <Sparkles class="size-4 shrink-0" aria-hidden="true" />
          {{ metadataRefreshBusy ? t("settings.submitting") : t("settings.refreshMetadata") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="gap-1.5 rounded-full text-muted-foreground hover:bg-muted/80 hover:text-foreground"
          data-library-paths-exit-batch
          @click="emit('exitBatchMode')"
        >
          <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
          {{ t("library.batchExitToolbar") }}
        </Button>
      </template>
    </div>
  </div>
</template>
