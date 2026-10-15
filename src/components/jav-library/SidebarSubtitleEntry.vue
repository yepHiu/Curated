<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Captions } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { useSubtitleJobs, isSubtitleActive, subtitleProgress } from "@/composables/use-subtitle-jobs"
defineProps<{ compact?: boolean }>()
const { t } = useI18n()
const state = useSubtitleJobs()
const job = state.current
const label = computed(() => !state.connected.value ? t('topics.disconnected') : job.value ? `${job.value.fileName} · ${t(`subtitles.stages.${job.value.stage}`)} · ${t(`subtitles.states.${job.value.state}`)}` : '')
</script>
<template>
  <section v-if="job" class="mb-2 min-w-0" data-sidebar-subtitle-jobs>
    <Button v-if="compact" variant="ghost" size="icon" class="relative mx-auto flex size-11 rounded-lg" :title="label" :aria-label="`${t('subtitles.title')}: ${label}`" @click="state.openJob(job)">
      <Captions />
      <Progress v-if="isSubtitleActive(job)" :model-value="subtitleProgress(job)" class="absolute inset-x-1.5 bottom-1 h-0.5 w-auto" aria-hidden="true" />
    </Button>
    <button v-else type="button" class="flex min-h-11 w-full min-w-0 flex-col gap-2 rounded-lg border border-border/60 bg-background/45 px-3 py-2.5 text-left transition-colors hover:bg-sidebar-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60" :aria-label="`${t('subtitles.title')}: ${label}`" @click="state.openJob(job)">
      <span class="flex min-w-0 items-center justify-between gap-2 text-xs">
        <span class="flex min-w-0 items-center gap-2 font-medium"><Captions class="size-3.5 shrink-0 text-primary" aria-hidden="true" /><span class="truncate">{{ t('subtitles.title') }}</span></span>
        <span v-if="isSubtitleActive(job) && job.total > 0" class="shrink-0 tabular-nums text-muted-foreground">{{ Math.round(subtitleProgress(job)) }}%</span>
      </span>
      <span class="truncate text-xs text-muted-foreground" :title="label">{{ label }}</span>
      <Progress v-if="isSubtitleActive(job) && job.total > 0" :model-value="subtitleProgress(job)" class="h-1" :aria-label="label" />
      <span class="text-xs text-muted-foreground">{{ t('subtitles.historyCount', { count: state.jobs.value.length }) }}</span>
    </button>
    <p v-if="state.error.value" role="alert" class="break-words px-2 py-1 text-xs text-destructive">{{ state.error.value === 'subtitles.movieUnavailable' ? t(state.error.value) : state.error.value }}</p>
    <div v-if="state.active.value.length > 1" class="flex flex-col gap-1 pt-1">
      <Button v-for="other in state.active.value.filter(item => item.id !== job?.id)" :key="other.id" variant="ghost" :size="compact ? 'icon' : 'default'" class="min-h-11 min-w-0 rounded-lg" :title="other.fileName" :aria-label="`${t('subtitles.title')}: ${other.fileName}`" @click="state.openJob(other)"><Captions v-if="compact" /><span v-else class="truncate">{{ other.fileName }}</span></Button>
    </div>
  </section>
</template>
