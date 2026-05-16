<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { TabsList, TabsTrigger } from "@/components/ui/tabs"

defineProps<{
  shownCount: number
  totalRows: number
  batchMode: boolean
  showSelectVisible: boolean
  selectVisibleDisabled: boolean
}>()

const emit = defineEmits<{
  enterBatchMode: []
  selectVisible: []
  exitBatchMode: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex shrink-0 flex-wrap items-center justify-between gap-3 pt-4 sm:pt-6">
    <div class="flex flex-wrap items-center gap-3">
      <TabsList class="h-auto w-fit max-w-full flex-wrap rounded-2xl bg-muted/60 p-1">
        <TabsTrigger value="timeline" class="rounded-xl px-4 py-2">{{ t("curated.tabTimeline") }}</TabsTrigger>
        <TabsTrigger value="actors" class="rounded-xl px-4 py-2">{{ t("curated.tabActors") }}</TabsTrigger>
        <TabsTrigger value="movies" class="rounded-xl px-4 py-2">{{ t("curated.tabMovies") }}</TabsTrigger>
      </TabsList>
      <p class="text-xs text-muted-foreground">
        {{ t("curated.pageSummary", { shown: shownCount, total: totalRows }) }}
      </p>
    </div>
    <div class="flex shrink-0 flex-wrap items-center gap-2">
      <template v-if="!batchMode">
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          @click="emit('enterBatchMode')"
        >
          <ListChecks class="size-4 opacity-80" aria-hidden="true" />
          {{ t("library.batchManage") }}
        </Button>
      </template>
      <template v-else>
        <Button
          v-if="showSelectVisible"
          type="button"
          variant="outline"
          size="sm"
          class="gap-1.5 rounded-xl"
          :disabled="selectVisibleDisabled"
          @click="emit('selectVisible')"
        >
          <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
          {{ t("library.batchSelectVisible") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
          @click="emit('exitBatchMode')"
        >
          <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
          {{ t("library.batchExitToolbar") }}
        </Button>
      </template>
    </div>
  </div>
</template>
