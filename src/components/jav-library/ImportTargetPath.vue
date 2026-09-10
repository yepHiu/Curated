<script setup lang="ts">
import { useI18n } from "vue-i18n"

defineProps<{
  path?: string
  emptyMessage: string
}>()

const { t } = useI18n()
</script>

<template>
  <div data-import-target-path class="min-w-0 rounded-xl border border-border/70 bg-muted/25 px-3 py-2.5 text-sm">
    <div class="flex min-w-0 flex-wrap items-center justify-between gap-2">
      <span class="shrink-0 text-muted-foreground">{{ t("import.targetPath") }}</span>
      <span v-if="path" class="max-w-full truncate font-mono text-xs text-foreground" :title="path">
        {{ path }}
      </span>
      <span v-else class="text-xs text-destructive">{{ emptyMessage }}</span>
    </div>
  </div>
</template>
