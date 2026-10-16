<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
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
  <Card
    data-photo-viewer
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
      <SettingsHint :text="t('settings.photoViewerDesc')">
        <CardTitle class="flex min-w-0 flex-wrap items-center gap-2 text-lg tracking-tight">
          <span>{{ t("settings.photoViewerTitle") }}</span>
          <SettingsScopeBadge scope="server" />
        </CardTitle>
      </SettingsHint>
    </CardHeader>
    <CardContent class="flex flex-col gap-3 pt-0">
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
    </CardContent>
  </Card>
</template>
