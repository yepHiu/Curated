<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Database } from "lucide-vue-next"
import type { ComicCacheStatusDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const GB = 1024 * 1024 * 1024

const CACHE_PRESETS = [
  { maxBytes: 1 * GB, labelKey: "settings.comicCachePreset1gb" },
  { maxBytes: 2 * GB, labelKey: "settings.comicCachePreset2gb" },
  { maxBytes: 5 * GB, labelKey: "settings.comicCachePreset5gb" },
  { maxBytes: 10 * GB, labelKey: "settings.comicCachePreset10gb" },
  { maxBytes: -1, labelKey: "settings.comicCachePresetUnlimited" },
]

const props = defineProps<{
  maxBytes: number
  status: ComicCacheStatusDTO | null
  saving: boolean
  cleanupBusy: boolean
  error: string
}>()

const emit = defineEmits<{
  changeMaxBytes: [maxBytes: number]
  cleanup: []
}>()

const { t } = useI18n()

const modelValue = computed(() => String(props.maxBytes))

function onPreset(value: unknown) {
  if (typeof value !== "string") return
  const next = Number(value)
  if (Number.isFinite(next)) {
    emit("changeMaxBytes", next)
  }
}
</script>

<template>
  <div
    data-comic-cache
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
    :aria-busy="saving || cleanupBusy"
  >
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <Database class="size-4 text-primary" aria-hidden="true" />
        <SettingsHint :text="t('settings.comicCacheDesc')">
          <p class="text-sm font-semibold text-foreground">
            {{ t("settings.comicCacheTitle") }}
          </p>
        </SettingsHint>
      </div>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-col gap-1">
        <SettingsHint :text="t('settings.comicCacheLimitDesc')">
          <p class="text-sm font-medium text-foreground">
            {{ t("settings.comicCacheLimit") }}
          </p>
        </SettingsHint>
      </div>
      <Select :model-value="modelValue" :disabled="saving" @update:model-value="onPreset">
        <SelectTrigger
          size="sm"
          class="h-9 w-full min-w-[12rem] rounded-xl border-border/50 sm:w-52"
          :aria-label="t('settings.comicCacheLimit')"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent align="end" class="rounded-xl border-border/50">
          <SelectItem
            v-for="preset in CACHE_PRESETS"
            :key="preset.maxBytes"
            :value="String(preset.maxBytes)"
          >
            {{ t(preset.labelKey) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{
          status
            ? t("settings.comicCacheUsage", {
                used: status.usedBytes,
                entries: status.entryCount,
              })
            : t("settings.comicCacheUsageUnknown")
        }}
      </p>
      <Button
        type="button"
        size="sm"
        variant="outline"
        class="h-8 shrink-0"
        :disabled="cleanupBusy"
        @click="emit('cleanup')"
      >
        {{ cleanupBusy ? t("settings.comicCacheCleaning") : t("settings.comicCacheCleanup") }}
      </Button>
    </div>

    <p v-if="saving" class="text-xs text-muted-foreground motion-safe:animate-pulse">
      {{ t("settings.comicCacheSyncing") }}
    </p>
    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
  </div>
</template>
