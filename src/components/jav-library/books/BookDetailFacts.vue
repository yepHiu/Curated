<script setup lang="ts">
import { useI18n } from "vue-i18n"
defineProps<{ pageCount: number; fileName: string; addedAt: string; location: string }>()
const { t } = useI18n()
function dateLabel(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : date.toISOString().slice(0, 10) }
</script>
<template>
  <dl class="grid min-w-0 grid-cols-2 gap-4 rounded-2xl border border-border/60 bg-muted/20 p-4 text-sm">
    <div class="flex flex-col gap-1"><dt class="text-xs text-muted-foreground">{{ t('bookBrowser.pages') }}</dt><dd class="font-medium tabular-nums">{{ pageCount }}</dd></div>
    <div class="flex flex-col gap-1"><dt class="text-xs text-muted-foreground">{{ t('bookBrowser.addedAt') }}</dt><dd>{{ dateLabel(addedAt) }}</dd></div>
    <div class="col-span-2 flex min-w-0 flex-col gap-1"><dt class="text-xs text-muted-foreground">{{ t('bookBrowser.sourceFile') }}</dt><dd class="break-all text-xs leading-relaxed">{{ fileName || '—' }}</dd></div>
    <div v-if="location" class="col-span-2 min-w-0"><dt class="sr-only">{{ t('bookBrowser.location') }}</dt><dd><details class="min-w-0"><summary class="min-h-11 cursor-pointer py-3 text-xs text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">{{ t('bookBrowser.location') }}</summary><p class="break-all text-xs leading-relaxed text-muted-foreground">{{ location }}</p></details></dd></div>
  </dl>
</template>
