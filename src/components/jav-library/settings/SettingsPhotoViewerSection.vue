<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import { useI18n } from "vue-i18n"
import { PanelsTopLeft } from "lucide-vue-next"
import type { PhotoViewerSettings } from "@/domain/photo/types"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const props = defineProps<{
  viewer: PhotoViewerSettings
  saving: boolean
  error: string
}>()

const emit = defineEmits<{
  patchViewer: [patch: Partial<PhotoViewerSettings>]
}>()

const { t } = useI18n()

function onMode(value: unknown) {
  if (value === "page" || value === "scroll") {
    emit("patchViewer", { mode: value })
  }
}

function onFit(value: unknown) {
  if (value === "contain" || value === "width") {
    emit("patchViewer", { fit: value })
  }
}

function onDirection(value: unknown) {
  if (value === "ltr" || value === "rtl") {
    emit("patchViewer", { direction: value })
  }
}
</script>

<template>
  <div
    data-photo-viewer
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
    :aria-busy="saving"
  >
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <PanelsTopLeft class="size-4 text-primary" aria-hidden="true" />
        <SettingsHint :text="t('settings.photoViewerDesc')">
          <p class="text-sm font-semibold text-foreground">
            {{ t("settings.photoViewerTitle") }}
          </p>
        </SettingsHint>
      </div>
    </div>

    <div data-photo-viewer-settings-list class="flex flex-col gap-2">
      <label
        data-photo-viewer-setting-row="mode"
        class="grid grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)] items-center gap-3"
      >
        <span class="min-w-0 text-sm font-medium text-muted-foreground">
          {{ t("settings.photoViewerMode") }}
        </span>
        <Select :model-value="props.viewer.mode" :disabled="saving" @update:model-value="onMode">
          <SelectTrigger size="sm" class="h-9 w-full min-w-0 rounded-xl border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent class="rounded-xl border-border/50">
            <SelectItem value="page">{{ t("settings.photoViewerModePage") }}</SelectItem>
            <SelectItem value="scroll">{{ t("settings.photoViewerModeScroll") }}</SelectItem>
          </SelectContent>
        </Select>
      </label>

      <label
        data-photo-viewer-setting-row="fit"
        class="grid grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)] items-center gap-3"
      >
        <span class="min-w-0 text-sm font-medium text-muted-foreground">
          {{ t("settings.photoViewerFit") }}
        </span>
        <Select :model-value="props.viewer.fit" :disabled="saving" @update:model-value="onFit">
          <SelectTrigger size="sm" class="h-9 w-full min-w-0 rounded-xl border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent class="rounded-xl border-border/50">
            <SelectItem value="contain">{{ t("settings.photoViewerFitContain") }}</SelectItem>
            <SelectItem value="width">{{ t("settings.photoViewerFitWidth") }}</SelectItem>
          </SelectContent>
        </Select>
      </label>

      <label
        data-photo-viewer-setting-row="direction"
        class="grid grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)] items-center gap-3"
      >
        <span class="min-w-0 text-sm font-medium text-muted-foreground">
          {{ t("settings.photoViewerDirection") }}
        </span>
        <Select
          :model-value="props.viewer.direction"
          :disabled="saving"
          @update:model-value="onDirection"
        >
          <SelectTrigger size="sm" class="h-9 w-full min-w-0 rounded-xl border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent class="rounded-xl border-border/50">
            <SelectItem value="ltr">{{ t("settings.photoViewerDirectionLtr") }}</SelectItem>
            <SelectItem value="rtl">{{ t("settings.photoViewerDirectionRtl") }}</SelectItem>
          </SelectContent>
        </Select>
      </label>
    </div>

    <p v-if="saving" class="text-xs text-muted-foreground motion-safe:animate-pulse">
      {{ t("settings.photoViewerSyncing") }}
    </p>
    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
  </div>
</template>
