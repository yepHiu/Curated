<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { RefreshCw } from "lucide-vue-next"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"

defineProps<{
  enabled: boolean
  saving: boolean
  error: string
}>()

const emit = defineEmits<{
  change: [value: boolean]
}>()

const { t } = useI18n()
</script>

<template>
  <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
        aria-hidden="true"
      >
        <RefreshCw class="size-4" />
      </span>
      <CardTitle class="min-w-0 text-lg tracking-tight">
        {{ t("settings.autoDownloadUpdatesTitle") }}
      </CardTitle>
    </CardHeader>
    <CardContent class="flex flex-col gap-3 pt-0">
      <div
        class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
        :aria-busy="saving"
      >
        <div class="min-w-0 flex flex-col gap-1">
          <p class="text-sm font-semibold text-foreground">
            {{ t("settings.autoDownloadUpdatesSwitch") }}
          </p>
          <p v-if="saving" role="status" class="text-xs text-muted-foreground motion-safe:animate-pulse">
            {{ t("settings.autoDownloadUpdatesSyncing") }}
          </p>
        </div>
        <Switch
          class="motion-safe:transition-colors motion-safe:duration-200"
          :model-value="enabled"
          :aria-label="t('settings.autoDownloadUpdatesSwitch')"
          @update:model-value="emit('change', $event)"
        />
      </div>
      <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
    </CardContent>
  </Card>
</template>
