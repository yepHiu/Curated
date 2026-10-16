<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { Captions, ChevronRight, RefreshCw, Sparkles } from "lucide-vue-next"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import MediaEmptyState from "@/components/jav-library/MediaEmptyState.vue"
import TaskStatusBadge from "@/components/jav-library/TaskStatusBadge.vue"
import { useTaskCenterService } from "@/services/task-center-service"
import { useTaskCenter } from "@/composables/use-task-center"
import type { CenterTask, TaskKind, TaskFilter } from "@/services/contracts/task-center-service"
import { taskActive, taskAttention, taskPercent, taskRoute } from "@/lib/task-center"
import { taskDate, taskStage, taskSubject, taskTitle } from "@/lib/task-center-text"

const { t } = useI18n(), route = useRoute(), router = useRouter(), service = useTaskCenterService()
const center = useTaskCenter()
const items = ref<CenterTask[]>([]), cursor = ref("")
const scrollEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const pageCount = computed(() => Math.max(1, Math.min(100, Number(route.query.pages) || 1)))
const loading = ref(true), failed = ref(false), moreBusy = ref(false)
const state = computed<TaskFilter>(() => ["active", "attention", "history"].includes(String(route.query.state)) ? route.query.state as TaskFilter : "all")
const kind = computed<TaskKind | "all">(() => ["subtitle", "organize"].includes(String(route.query.kind)) ? route.query.kind as TaskKind : "all")
const groups = computed(() => [
  { key: "active", items: items.value.filter(taskActive) },
  { key: "attention", items: items.value.filter(taskAttention) },
  { key: "finished", items: items.value.filter(item => !taskActive(item) && !taskAttention(item)) },
].filter(group => group.items.length))
let generation = 0
async function load(more = false, quiet = false) {
  if (moreBusy.value) return
  const seq = ++generation
  if (more) moreBusy.value = true
  else if (!quiet) loading.value = true
  try {
    let page = await service.list({ kind: kind.value, state: state.value, cursor: more ? cursor.value : undefined, limit: 30 })
    if (!more) {
      const all = [...page.items]
      for (let i = 1; i < pageCount.value && page.nextCursor; i++) {
        if (seq !== generation) return
        page = await service.list({ kind: kind.value, state: state.value, cursor: page.nextCursor, limit: 30 })
        all.push(...page.items)
      }
      page = { ...page, items: all }
    }
    if (seq !== generation) return
    items.value = more ? [...items.value, ...page.items.filter(item => !items.value.some(old => old.key === item.key))] : page.items
    cursor.value = page.nextCursor; failed.value = false
    if (more) void router.replace({ query: { ...route.query, pages: String(pageCount.value + 1) } })
    if (!quiet && !more) { loading.value = false; await nextTick(); if (seq === generation && scrollEl.value) scrollEl.value.scrollTop = Number(route.query.scroll) || 0 }
  } catch { if (seq === generation) failed.value = true }
  finally { if (seq === generation) { loading.value = false; moreBusy.value = false } }
}
function filter(field: "state" | "kind", value: unknown) { void router.replace({ query: { ...route.query, [field]: String(value) === "all" ? undefined : String(value), pages: undefined, scroll: undefined } }) }
watch([state, kind], () => { generation++; moreBusy.value = false; items.value = []; void load() }, { immediate: true })
// Refresh all loaded pages without collapsing history or moving its scroll position.
watch(center.revision, () => { if (!loading.value && !moreBusy.value) void load(false, true) })
onBeforeUnmount(() => generation++)
</script>
<template>
  <div ref="scrollEl" class="h-full min-h-0 overflow-y-auto" data-task-center @scroll="scrollTop = scrollEl?.scrollTop ?? 0">
    <div class="mx-auto flex max-w-6xl flex-col gap-6 px-1 py-3 sm:px-4 sm:py-5">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('tasks.title') }}</h1>
      <Alert v-if="failed || !center.connected.value"><AlertDescription class="flex flex-wrap items-center justify-between gap-3"><span>{{ items.length ? t('tasks.stale', { time: taskDate(center.updatedAt.value) }) : t('tasks.loadFailed') }}</span><Button variant="outline" size="sm" @click="load()"><RefreshCw data-icon="inline-start" />{{ t('tasks.reload') }}</Button></AlertDescription></Alert>
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-border pb-4">
        <Tabs :model-value="state" @update:model-value="filter('state', $event)"><TabsList :aria-label="t('tasks.statusFilter')"><TabsTrigger v-for="tab in ['all','active','attention','history']" :key="tab" :value="tab">{{ t(`tasks.filters.${tab}`) }}</TabsTrigger></TabsList></Tabs>
        <Select :model-value="kind" @update:model-value="filter('kind', $event)"><SelectTrigger class="min-h-11 w-40 rounded-full sm:min-h-9" :aria-label="t('tasks.typeFilter')"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="type in ['all','subtitle','organize']" :key="type" :value="type">{{ t(`tasks.kinds.${type}`) }}</SelectItem></SelectGroup></SelectContent></Select>
      </div>
      <div v-if="loading" class="flex flex-col gap-3" role="status" :aria-label="t('tasks.loading')"><Skeleton v-for="i in 3" :key="i" class="h-24 w-full rounded-xl" /></div>
      <MediaEmptyState v-else-if="!items.length && !failed" :title="t('tasks.empty')" :description="t(state === 'all' && kind === 'all' ? 'tasks.emptyDescription' : 'tasks.noMatch')"><Button v-if="state !== 'all' || kind !== 'all'" variant="outline" @click="router.replace({ query: {} })">{{ t('tasks.clearFilters') }}</Button><Button v-else as-child variant="outline"><RouterLink to="/library">{{ t('tasks.library') }}</RouterLink></Button></MediaEmptyState>
      <section v-for="group in groups" v-else :key="group.key" class="flex flex-col gap-3">
        <h2 class="text-xs font-medium text-muted-foreground">{{ t(`tasks.groups.${group.key}`) }} · {{ group.items.length }}</h2>
        <div class="divide-y divide-border overflow-hidden rounded-2xl border border-border bg-card">
          <RouterLink v-for="task in group.items" :key="task.key" :to="{ path: taskRoute(task.key), query: { ...route.query, scroll: String(scrollTop) } }" class="grid min-w-0 grid-cols-[2.5rem_minmax(0,1fr)_1rem] items-center gap-3 px-4 py-5 transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring lg:grid-cols-[2.5rem_minmax(0,1fr)_minmax(12rem,30%)_1rem]" :aria-label="`${taskTitle(task)} · ${taskSubject(task)}`">
            <span class="flex size-10 items-center justify-center rounded-xl bg-secondary text-primary"><Captions v-if="task.kind === 'subtitle'" class="size-5" /><Sparkles v-else class="size-5" /></span>
            <div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><h3 class="text-sm font-medium">{{ taskTitle(task) }}</h3><TaskStatusBadge :state="task.state" /></div><p class="mt-1.5 break-words text-xs text-muted-foreground">{{ taskSubject(task) }}</p></div>
            <div class="col-start-2 min-w-0 text-xs lg:col-auto"><div class="flex flex-wrap justify-between gap-2"><span>{{ taskStage(task) }}</span><span v-if="taskActive(task) && task.total > 0" class="tabular-nums">{{ task.completed }} / {{ task.total }} {{ t(`tasks.units.${task.unit}`) }}</span></div><Progress v-if="taskActive(task) && task.total > 0" :model-value="taskPercent(task)" class="my-2 h-1" :aria-label="taskStage(task)" /><p class="mt-1 text-xs text-muted-foreground">{{ taskDate(task.updatedAt) }}</p></div>
            <ChevronRight class="col-start-3 row-start-1 size-4 text-muted-foreground lg:col-auto lg:row-auto" aria-hidden="true" />
          </RouterLink>
        </div>
      </section>
      <Button v-if="cursor && !loading" variant="outline" class="self-center rounded-full" :disabled="moreBusy" @click="load(true)">{{ t(moreBusy ? 'tasks.loading' : 'tasks.loadMore') }}</Button>
    </div>
  </div>
</template>
