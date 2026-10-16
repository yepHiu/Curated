<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { RouterLink } from "vue-router"
import { ListTodo } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { useTaskCenter } from "@/composables/use-task-center"
defineProps<{ compact?: boolean }>()
const { t } = useI18n()
const { summary, connected, loaded } = useTaskCenter()
</script>
<template>
  <section class="mb-2 min-w-0" data-sidebar-task-center>
    <Button v-if="compact" as-child variant="ghost" size="icon" class="mx-auto flex size-11 rounded-lg"><RouterLink to="/tasks" :title="t('tasks.title')" :aria-label="t('tasks.title')"><ListTodo /></RouterLink></Button>
    <RouterLink v-else to="/tasks" class="flex min-h-11 flex-col gap-2 rounded-lg border border-border/60 bg-background/45 px-3 py-2.5 text-sidebar-foreground transition-colors hover:bg-sidebar-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60">
      <span class="flex items-center justify-between gap-2 text-xs font-medium"><span class="flex items-center gap-2"><ListTodo class="size-4 text-muted-foreground" aria-hidden="true" />{{ t('tasks.title') }}</span><span v-if="loaded" class="text-muted-foreground">{{ t('tasks.runningCount', { count: summary.running }) }}</span></span>
      <span class="text-xs text-muted-foreground">{{ !connected ? t('tasks.disconnected') : !loaded ? t('tasks.loading') : t('tasks.queueCount', { count: summary.queued + summary.waiting }) }}</span>
      <span v-if="loaded && summary.attention" class="text-xs text-warning">{{ t('tasks.attentionCount', { count: summary.attention }) }}</span>
    </RouterLink>
  </section>
</template>
