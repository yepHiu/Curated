<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { RefreshCw } from "lucide-vue-next"
import { Button } from "@/components/ui/button"

defineProps<{
  busy: boolean
  success: string
  error: string
}>()

const emit = defineEmits<{
  run: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex flex-col gap-3">
    <div
      class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-start sm:justify-between"
    >
      <div class="min-w-0 flex flex-col gap-3 text-left">
        <p class="text-sm font-semibold text-foreground">
          {{ t("settings.triggerScrape") }}
        </p>
        <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
          {{ t("settings.triggerScrapeHint") }}
        </p>
      </div>
      <Button
        type="button"
        variant="default"
        class="h-11 shrink-0 rounded-2xl px-5 font-medium"
        :disabled="busy"
        data-trigger-scrape-run
        @click="emit('run')"
      >
        <RefreshCw
          data-icon="inline-start"
          class="size-4"
          :class="{ 'motion-safe:animate-spin': busy }"
        />
        {{ busy ? t("settings.triggerScrapeRunning") : t("settings.triggerScrapeRunButton") }}
      </Button>
    </div>
    <p
      v-if="success"
      class="text-sm text-primary"
    >
      {{ success }}
    </p>
    <p
      v-if="error"
      class="text-sm text-destructive"
      role="alert"
    >
      {{ error }}
    </p>
  </div>
</template>
