<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { ActorListItemDTO } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{ sourceName: string }>()
const emit = defineEmits<{ select: [actor: ActorListItemDTO] }>()
const { t } = useI18n()
const service = useLibraryService()
const query = ref("")
const actors = ref<ActorListItemDTO[]>([])
const loading = ref(false)
const failed = ref(false)
const total = ref(0)
let request = 0
let timer: ReturnType<typeof setTimeout> | undefined

async function search() {
  clearTimeout(timer)
  const id = ++request
  loading.value = true
  failed.value = false
  actors.value = []
  try {
    const result = await service.listActors({ q: query.value.trim(), sort: "name", limit: 20 })
    if (id !== request) return
    actors.value = result.actors.filter((actor) => actor.name !== props.sourceName)
    total.value = result.total
  } catch {
    if (id === request) failed.value = true
  } finally {
    if (id === request) loading.value = false
  }
}

watch(query, () => {
  ++request
  clearTimeout(timer)
  actors.value = []
  failed.value = false
  loading.value = true
  timer = setTimeout(() => void search(), 250)
})
watch(() => props.sourceName, () => void search(), { immediate: true })
onBeforeUnmount(() => {
  ++request
  clearTimeout(timer)
})
</script>

<template>
  <section class="flex min-w-0 flex-col gap-3" :aria-busy="loading">
    <label for="actor-merge-target" class="text-sm font-medium">{{ t("actors.merge.targetLabel") }}</label>
    <Input id="actor-merge-target" v-model="query" autocomplete="off" :placeholder="t('actors.merge.targetPlaceholder')" @keydown.enter.prevent="search" />
    <p v-if="loading" role="status" class="text-sm text-muted-foreground">{{ t("actors.merge.searching") }}</p>
    <div v-else-if="failed" role="alert" class="flex items-center justify-between gap-3">
      <p class="text-sm text-destructive">{{ t("actors.merge.searchFailed") }}</p>
      <Button variant="outline" class="min-h-11 sm:min-h-9" @click="search">{{ t("actors.merge.retry") }}</Button>
    </div>
    <p v-else-if="actors.length === 0" role="status" class="text-sm text-muted-foreground">{{ t("actors.merge.noResults") }}</p>
    <ul v-else class="max-h-56 overflow-y-auto rounded-xl border border-border/70" :aria-label="t('actors.merge.candidates')">
      <li v-for="actor in actors" :key="actor.name">
        <button type="button" class="flex min-h-14 w-full cursor-pointer items-center gap-3 p-3 text-left hover:bg-muted focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring" @click="emit('select', actor)">
          <Avatar class="size-10 shrink-0 rounded-lg">
            <AvatarImage v-if="actor.avatarUrl" :src="actor.avatarUrl" :alt="actor.name" class="object-cover" />
            <AvatarFallback class="rounded-lg">{{ actor.name.slice(0, 2) }}</AvatarFallback>
          </Avatar>
          <span class="min-w-0 flex-1 break-words text-sm font-medium">{{ actor.name }}</span>
          <span class="shrink-0 text-xs text-muted-foreground">{{ t("actors.movieCount", { n: actor.movieCount }) }}</span>
        </button>
      </li>
    </ul>
    <p v-if="!loading && !failed && total > 20" class="text-xs text-muted-foreground">{{ t("actors.merge.refineSearch") }}</p>
  </section>
</template>
