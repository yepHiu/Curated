<script setup lang="ts">
import { useI18n } from "vue-i18n"
defineProps<{ pageCount: number; addedAt: string }>()
const { t } = useI18n()
function dateLabel(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : date.toISOString().slice(0, 10) }
</script>
<template>
  <dl class="grid min-w-0 grid-cols-2 gap-4 rounded-2xl border border-border/60 bg-muted/20 p-4 text-sm">
    <div class="flex flex-col gap-1"><dt class="text-xs text-muted-foreground">{{ t('bookBrowser.pages') }}</dt><dd class="font-medium tabular-nums">{{ pageCount }}</dd></div>
    <div class="flex flex-col gap-1"><dt class="text-xs text-muted-foreground">{{ t('bookBrowser.addedAt') }}</dt><dd>{{ dateLabel(addedAt) }}</dd></div>
  </dl>
</template>
