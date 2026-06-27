<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { PanelsTopLeft } from "lucide-vue-next"
import type { ComicReaderSettings } from "@/domain/comic/types"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const props = defineProps<{
  reader: ComicReaderSettings
  saving: boolean
  error: string
}>()

const emit = defineEmits<{
  patchReader: [patch: Partial<ComicReaderSettings>]
}>()

const { t } = useI18n()

function onMode(value: unknown) {
  if (value === "page" || value === "scroll") {
    emit("patchReader", { mode: value })
  }
}

function onFit(value: unknown) {
  if (value === "contain" || value === "width") {
    emit("patchReader", { fit: value })
  }
}

function onDirection(value: unknown) {
  if (value === "ltr" || value === "rtl") {
    emit("patchReader", { direction: value })
  }
}
</script>

<template>
  <div
    data-comic-reader
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
    :aria-busy="saving"
  >
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <PanelsTopLeft class="size-4 text-primary" aria-hidden="true" />
        <p class="text-sm font-semibold text-foreground">
          {{ t("settings.comicReaderTitle") }}
        </p>
      </div>
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{ t("settings.comicReaderDesc") }}
      </p>
    </div>

    <div class="grid gap-3 md:grid-cols-3">
      <label class="flex flex-col gap-1.5">
        <span class="text-xs font-medium text-muted-foreground">
          {{ t("settings.comicReaderMode") }}
        </span>
        <Select :model-value="props.reader.mode" :disabled="saving" @update:model-value="onMode">
          <SelectTrigger size="sm" class="h-9 w-full rounded-xl border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent class="rounded-xl border-border/50">
            <SelectItem value="page">{{ t("settings.comicReaderModePage") }}</SelectItem>
            <SelectItem value="scroll">{{ t("settings.comicReaderModeScroll") }}</SelectItem>
          </SelectContent>
        </Select>
      </label>

      <label class="flex flex-col gap-1.5">
        <span class="text-xs font-medium text-muted-foreground">
          {{ t("settings.comicReaderFit") }}
        </span>
        <Select :model-value="props.reader.fit" :disabled="saving" @update:model-value="onFit">
          <SelectTrigger size="sm" class="h-9 w-full rounded-xl border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent class="rounded-xl border-border/50">
            <SelectItem value="contain">{{ t("settings.comicReaderFitContain") }}</SelectItem>
            <SelectItem value="width">{{ t("settings.comicReaderFitWidth") }}</SelectItem>
          </SelectContent>
        </Select>
      </label>

      <label class="flex flex-col gap-1.5">
        <span class="text-xs font-medium text-muted-foreground">
          {{ t("settings.comicReaderDirection") }}
        </span>
        <Select
          :model-value="props.reader.direction"
          :disabled="saving"
          @update:model-value="onDirection"
        >
          <SelectTrigger size="sm" class="h-9 w-full rounded-xl border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent class="rounded-xl border-border/50">
            <SelectItem value="rtl">{{ t("settings.comicReaderDirectionRtl") }}</SelectItem>
            <SelectItem value="ltr">{{ t("settings.comicReaderDirectionLtr") }}</SelectItem>
          </SelectContent>
        </Select>
      </label>
    </div>

    <p v-if="saving" class="text-xs text-muted-foreground motion-safe:animate-pulse">
      {{ t("settings.comicReaderSyncing") }}
    </p>
    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
  </div>
</template>
