<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
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
  <Card
    data-comic-cache
    class="gap-2 rounded-xl border border-border bg-card shadow-sm"
    :aria-busy="saving || cleanupBusy"
  >
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
        aria-hidden="true"
      >
        <Database class="size-4" />
      </span>
      <SettingsHint :text="t('settings.comicCacheDesc')">
        <CardTitle class="flex min-w-0 flex-wrap items-center gap-2 text-lg tracking-tight">
          <span>{{ t("settings.comicCacheTitle") }}</span>
          <SettingsScopeBadge scope="server" />
        </CardTitle>
      </SettingsHint>
    </CardHeader>
    <CardContent class="flex flex-col gap-3 pt-0">
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
          class="h-8 self-end shrink-0"
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
    </CardContent>
  </Card>
</template>
