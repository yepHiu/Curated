<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
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
  <Card
    data-comic-reader
    class="gap-2 rounded-xl border border-border bg-card shadow-sm"
    :aria-busy="saving"
  >
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
        aria-hidden="true"
      >
        <PanelsTopLeft class="size-4" />
      </span>
      <SettingsHint :text="t('settings.comicReaderDesc')">
        <CardTitle class="flex min-w-0 flex-wrap items-center gap-2 text-lg tracking-tight">
          <span>{{ t("settings.comicReaderTitle") }}</span>
          <SettingsScopeBadge scope="server" />
        </CardTitle>
      </SettingsHint>
    </CardHeader>
    <CardContent class="flex flex-col gap-3 pt-0">
      <div data-comic-reader-settings-list class="flex flex-col gap-2">
        <label
          data-comic-reader-setting-row="mode"
          class="grid grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)] items-center gap-3"
        >
          <span
            data-comic-reader-setting-label
            class="min-w-0 text-sm font-medium text-muted-foreground"
          >
            {{ t("settings.comicReaderMode") }}
          </span>
          <Select :model-value="props.reader.mode" :disabled="saving" @update:model-value="onMode">
            <SelectTrigger
              data-comic-reader-setting-trigger
              size="sm"
              class="h-9 w-full min-w-0 rounded-xl border-border/50"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent class="rounded-xl border-border/50">
              <SelectItem value="page">{{ t("settings.comicReaderModePage") }}</SelectItem>
              <SelectItem value="scroll">{{ t("settings.comicReaderModeScroll") }}</SelectItem>
            </SelectContent>
          </Select>
        </label>

        <label
          data-comic-reader-setting-row="fit"
          class="grid grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)] items-center gap-3"
        >
          <span
            data-comic-reader-setting-label
            class="min-w-0 text-sm font-medium text-muted-foreground"
          >
            {{ t("settings.comicReaderFit") }}
          </span>
          <Select :model-value="props.reader.fit" :disabled="saving" @update:model-value="onFit">
            <SelectTrigger
              data-comic-reader-setting-trigger
              size="sm"
              class="h-9 w-full min-w-0 rounded-xl border-border/50"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent class="rounded-xl border-border/50">
              <SelectItem value="contain">{{ t("settings.comicReaderFitContain") }}</SelectItem>
              <SelectItem value="width">{{ t("settings.comicReaderFitWidth") }}</SelectItem>
            </SelectContent>
          </Select>
        </label>

        <label
          data-comic-reader-setting-row="direction"
          class="grid grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)] items-center gap-3"
        >
          <span
            data-comic-reader-setting-label
            class="min-w-0 text-sm font-medium text-muted-foreground"
          >
            {{ t("settings.comicReaderDirection") }}
          </span>
          <Select
            :model-value="props.reader.direction"
            :disabled="saving"
            @update:model-value="onDirection"
          >
            <SelectTrigger
              data-comic-reader-setting-trigger
              size="sm"
              class="h-9 w-full min-w-0 rounded-xl border-border/50"
            >
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
    </CardContent>
  </Card>
</template>
