<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Loader2, Check, AlertTriangle, X } from 'lucide-vue-next'
import type { CaptureJob } from '@/composables/use-curated-capture-queue'
const props = defineProps<{ job: CaptureJob; pending: number }>()
defineEmits<{ retry: []; retryExport: []; undo: []; view: []; dismiss: [] }>()
const { t } = useI18n()
const busy = computed(() => ['capturing', 'queued', 'saving'].includes(props.job.phase))
const time = computed(() => {
  const seconds = Math.max(0, Math.floor(props.job.positionSec))
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, seconds % 60].map(n => String(n).padStart(2, '0')).join(':')
})
</script>
<template>
  <div class="absolute bottom-24 left-3 z-20 flex max-w-[calc(100%-1.5rem)] items-center gap-3 rounded-lg border border-border bg-background/95 p-2 text-foreground shadow-lg sm:left-5" @click.stop @pointerdown.stop @keydown.stop>
    <button v-if="job.preview" type="button" class="shrink-0 rounded focus-visible:ring-2 focus-visible:ring-ring" :aria-label="t('curated.captureView')" @click="$emit('view')">
      <img :src="job.preview" :alt="job.movie.code" class="h-[54px] w-24 rounded object-contain" />
    </button>
    <div class="flex min-w-0 flex-col gap-1">
      <div class="flex items-center gap-2 text-sm">
        <Loader2 v-if="busy" class="size-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
        <AlertTriangle v-else-if="job.error" class="size-4 text-destructive" aria-hidden="true" />
        <Check v-else class="size-4" aria-hidden="true" />
        <span class="font-mono tabular-nums">{{ time }}</span>
        <span>{{ t(busy ? 'curated.captureSaving' : job.committed ? 'curated.captureSaved' : 'curated.captureFailed') }}</span>
      </div>
      <span v-if="pending > 1" class="text-xs text-muted-foreground">{{ t('curated.capturePending', { n: pending }) }}</span>
      <span v-if="job.error" class="max-w-64 text-xs text-destructive">{{ job.error }}</span>
      <div class="flex flex-wrap gap-1">
        <Button v-if="job.phase === 'error' && job.candidate" size="sm" variant="outline" @click="$emit('retry')">{{ t('curated.captureRetry') }}</Button>
        <Button v-if="job.phase === 'export-error'" size="sm" variant="outline" :disabled="job.exporting" @click="$emit('retryExport')">{{ t('curated.captureRetryExport') }}</Button>
        <Button v-if="job.committed" size="sm" variant="ghost" :disabled="job.exporting" @click="$emit('undo')">{{ t('curated.captureUndo') }}</Button>
      </div>
    </div>
    <Button v-if="!busy" size="icon-sm" variant="ghost" :aria-label="t('common.close')" :disabled="job.exporting" @click="$emit('dismiss')"><X /></Button>
  </div>
</template>
