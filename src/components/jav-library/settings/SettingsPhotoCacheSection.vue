<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Database } from "lucide-vue-next"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const GB = 1024 * 1024 * 1024

const CACHE_PRESETS = [
  { maxBytes: 5 * GB, labelKey: "settings.photoCachePreset5gb" },
  { maxBytes: 10 * GB, labelKey: "settings.photoCachePreset10gb" },
  { maxBytes: 20 * GB, labelKey: "settings.photoCachePreset20gb" },
  { maxBytes: -1, labelKey: "settings.photoCachePresetUnlimited" },
]

const props = defineProps<{
  maxBytes: number
  saving: boolean
  error: string
}>()

const emit = defineEmits<{
  changeMaxBytes: [maxBytes: number]
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
    data-photo-cache
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
    :aria-busy="saving"
  >
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <Database class="size-4 text-primary" aria-hidden="true" />
        <p class="text-sm font-semibold text-foreground">
          {{ t("settings.photoCacheTitle") }}
        </p>
      </div>
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{ t("settings.photoCacheDesc") }}
      </p>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-col gap-1">
        <p class="text-sm font-medium text-foreground">
          {{ t("settings.photoCacheLimit") }}
        </p>
        <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
          {{ t("settings.photoCacheLimitDesc") }}
        </p>
      </div>
      <Select :model-value="modelValue" :disabled="saving" @update:model-value="onPreset">
        <SelectTrigger
          size="sm"
          class="h-9 w-full min-w-[12rem] rounded-xl border-border/50 sm:w-52"
          :aria-label="t('settings.photoCacheLimit')"
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

    <p v-if="saving" class="text-xs text-muted-foreground motion-safe:animate-pulse">
      {{ t("settings.photoCacheSyncing") }}
    </p>
    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
  </div>
</template>
