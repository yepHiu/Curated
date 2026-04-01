<script setup lang="ts">
import type { Component } from "vue"
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  ChevronLeft,
  ChevronRight,
  Clapperboard,
  History,
  LibraryBig,
  RefreshCw,
  Settings2,
  Sparkles,
  Tags,
  Trash2,
  Users,
} from "lucide-vue-next"
import { RouterLink, useRoute } from "vue-router"
import type { AppPage, LibraryMode } from "@/domain/library/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { useBackendHealth } from "@/composables/use-backend-health"
import { countCuratedFrames } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { buildBrowseRouteTarget } from "@/lib/library-query"
import { countUniqueTags, formatSidebarCount } from "@/lib/library-stats"
import {
  listSortedByUpdatedDesc,
  playbackProgressRevision,
} from "@/lib/playback-progress-storage"
import { useLibraryService } from "@/services/library-service"

const props = withDefaults(
  defineProps<{
    compact?: boolean
    showCollapseToggle?: boolean
  }>(),
  {
    compact: false,
    showCollapseToggle: false,
  },
)

const emit = defineEmits<{
  toggleCompact: []
}>()

interface NavigationItem {
  label: string
  page: AppPage
  icon: Component
  hint?: string
}

interface SidebarNavGroups {
  browse: NavigationItem[]
  yours: NavigationItem[]
}

const { t, locale } = useI18n()
const route = useRoute()
const libraryService = useLibraryService()
const {
  useWebApi: backendUseWebApi,
  status: backendStatus,
  probing: backendProbing,
  versionDisplay: backendVersionDisplay,
  checkNow: checkBackendHealth,
} = useBackendHealth()

const backendStatusText = computed(() => {
  void locale.value
  switch (backendStatus.value) {
    case "mock":
      return t("nav.backendMock")
    case "checking":
      return t("nav.backendChecking")
    case "online":
      return t("nav.backendOnline")
    case "offline":
      return t("nav.backendOffline")
    default:
      return ""
  }
})

const backendDotClass = computed(() => {
  switch (backendStatus.value) {
    case "checking":
      return "bg-muted-foreground/55 animate-pulse"
    case "online":
      return "bg-emerald-500 shadow-[0_0_10px_rgb(52_211_153_/_0.35)]"
    case "offline":
      return "bg-destructive"
    case "mock":
      return "bg-amber-500/90"
    default:
      return "bg-muted-foreground/40"
  }
})

const backendMetaText = computed(() => {
  if (backendVersionDisplay.value) {
    return backendVersionDisplay.value
  }
  return backendUseWebApi ? null : "mock"
})

const backendAriaLabel = computed(() => {
  const suffix = backendMetaText.value ? `, ${backendMetaText.value}` : ""
  if (backendStatus.value === "online") {
    return backendMetaText.value
      ? `${t("nav.backendLabel")}: ${backendMetaText.value}`
      : t("nav.backendLabel")
  }
  return `${t("nav.backendLabel")}: ${backendStatusText.value}${suffix}`
})

const backendCompactTitle = computed(() => {
  if (backendStatus.value === "online") {
    return backendMetaText.value ?? t("nav.backendLabel")
  }
  return backendMetaText.value
    ? `${backendStatusText.value}\n${backendMetaText.value}`
    : backendStatusText.value
})

const curatedFrameCount = ref(0)

async function refreshCuratedFrameCount() {
  try {
    curatedFrameCount.value = await countCuratedFrames()
  } catch {
    curatedFrameCount.value = 0
  }
}

watch(curatedFramesRevision, () => {
  void refreshCuratedFrameCount()
})

onMounted(() => {
  void refreshCuratedFrameCount()
})

const sidebarNavGroups = computed((): SidebarNavGroups => {
  void locale.value
  void playbackProgressRevision.value
  void curatedFramesRevision.value
  const movies = libraryService.movies.value
  const trashTotal = libraryService.trashedMovies.value.length
  const total = movies.length
  const tagCount = countUniqueTags(movies)
  const actorSet = new Set<string>()
  for (const movie of movies) {
    for (const rawActor of movie.actors) {
      const actor = rawActor.trim()
      if (actor) actorSet.add(actor)
    }
  }
  const actorCount = actorSet.size
  const historyTotal = listSortedByUpdatedDesc().length
  const framesTotal = curatedFrameCount.value

  return {
    browse: [
      { label: t("nav.library"), page: "library", icon: LibraryBig, hint: formatSidebarCount(total) },
      {
        label: t("nav.actors"),
        page: "actors",
        icon: Users,
        hint: actorCount > 0 ? formatSidebarCount(actorCount) : undefined,
      },
      { label: t("nav.tags"), page: "tags", icon: Tags, hint: formatSidebarCount(tagCount) },
      {
        label: t("nav.trash"),
        page: "trash",
        icon: Trash2,
        hint: trashTotal > 0 ? formatSidebarCount(trashTotal) : undefined,
      },
    ],
    yours: [
      {
        label: t("nav.curatedFrames"),
        page: "curated-frames",
        icon: Clapperboard,
        hint: framesTotal > 0 ? formatSidebarCount(framesTotal) : undefined,
      },
      {
        label: t("nav.history"),
        page: "history",
        icon: History,
        hint: historyTotal > 0 ? formatSidebarCount(historyTotal) : undefined,
      },
    ],
  }
})

const isActive = (page: AppPage) => route.name === page

const getNavigationTarget = (page: AppPage) => {
  if (page === "settings") {
    return { name: page }
  }
  if (page === "history") {
    return { name: "history" }
  }
  if (page === "curated-frames") {
    return { name: "curated-frames" }
  }
  if (page === "actors") {
    return { name: "actors" }
  }
  return buildBrowseRouteTarget(page as LibraryMode, route.query)
}
</script>

<template>
  <aside
    class="flex h-full min-h-0 w-full min-w-0 flex-col overflow-x-hidden rounded-[1.75rem] border border-border/60 bg-sidebar/95 text-sidebar-foreground backdrop-blur transition-[padding] duration-300 ease-in-out motion-reduce:transition-none"
    :class="props.compact ? 'items-center px-2 py-3' : 'px-3.5 py-3.5'"
  >
    <div
      v-if="!props.compact"
      class="flex min-h-14 items-center justify-between gap-2 px-2 py-2.5"
    >
      <div class="flex min-w-0 items-center">
        <div
          class="font-curated inline-flex w-fit max-w-full items-center gap-2 px-1 py-1 text-lg font-semibold tracking-wide text-primary sm:text-xl"
          title="Curated"
        >
          <Sparkles class="size-5 shrink-0 text-primary sm:size-[1.35rem]" aria-hidden="true" />
          <span class="truncate">Curated</span>
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-1">
        <Button
          v-if="props.showCollapseToggle"
          type="button"
          variant="ghost"
          size="icon"
          class="rounded-xl"
          :title="t('nav.collapseSidebar')"
          :aria-label="t('nav.collapseSidebar')"
          @click="emit('toggleCompact')"
        >
          <ChevronLeft class="size-5" />
        </Button>
      </div>
    </div>

    <div
      v-else
      class="flex w-full flex-col items-center gap-2 py-1"
    >
      <Button
        v-if="props.showCollapseToggle"
        type="button"
        variant="ghost"
        size="icon-lg"
        class="shrink-0 rounded-xl"
        :title="t('nav.expandSidebar')"
        :aria-label="t('nav.expandSidebar')"
        @click="emit('toggleCompact')"
      >
        <ChevronRight class="size-5" />
      </Button>
      <div
        class="flex size-10 shrink-0 items-center justify-center rounded-2xl text-primary"
        title="Curated"
      >
        <Sparkles class="size-5 text-primary" />
      </div>
    </div>

    <Separator
      class="my-2.5 shrink-0 bg-sidebar-border/80"
      :class="props.compact ? 'w-10' : ''"
    />

    <ScrollArea v-if="!props.compact" class="min-h-0 w-full min-w-0 flex-1">
      <div class="flex flex-col gap-5 pr-2.5">
        <section class="flex flex-col gap-2">
          <span class="px-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground">
            {{ t("nav.browse") }}
          </span>
          <Button
            v-for="item in sidebarNavGroups.browse"
            :key="item.page"
            as-child
            :variant="isActive(item.page) ? 'secondary' : 'ghost'"
            class="w-full justify-between rounded-2xl px-3"
          >
            <RouterLink :to="getNavigationTarget(item.page)">
              <span class="flex min-w-0 items-center gap-2 truncate">
                <component :is="item.icon" data-icon="inline-start" />
                <span class="truncate">{{ item.label }}</span>
              </span>
              <Badge
                v-if="item.hint"
                variant="secondary"
                class="rounded-full border border-border/60 bg-background/60"
              >
                {{ item.hint }}
              </Badge>
            </RouterLink>
          </Button>
        </section>

        <section class="flex flex-col gap-2">
          <span class="px-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground">
            {{ t("nav.yours") }}
          </span>
          <Button
            v-for="item in sidebarNavGroups.yours"
            :key="item.page"
            as-child
            :variant="isActive(item.page) ? 'secondary' : 'ghost'"
            class="w-full justify-between rounded-2xl px-3"
          >
            <RouterLink :to="getNavigationTarget(item.page)">
              <span class="flex min-w-0 items-center gap-2 truncate">
                <component :is="item.icon" data-icon="inline-start" />
                <span class="truncate">{{ item.label }}</span>
              </span>
              <Badge
                v-if="item.hint"
                variant="secondary"
                class="rounded-full border border-border/60 bg-background/60"
              >
                {{ item.hint }}
              </Badge>
            </RouterLink>
          </Button>
        </section>
      </div>
    </ScrollArea>

    <div
      v-if="props.compact"
      class="flex min-h-0 w-full min-w-0 flex-1 flex-col self-stretch overflow-y-auto"
    >
      <div class="flex flex-col gap-3 py-1">
        <nav class="flex flex-col items-center gap-2" :aria-label="t('nav.browse')">
          <RouterLink
            v-for="item in sidebarNavGroups.browse"
            :key="item.page"
            :to="getNavigationTarget(item.page)"
            :title="item.label"
            class="flex size-10 shrink-0 items-center justify-center rounded-2xl text-sidebar-foreground transition-colors outline-none hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
            :class="isActive(item.page) ? 'bg-sidebar-accent text-sidebar-accent-foreground' : ''"
          >
            <component :is="item.icon" class="size-5 shrink-0" />
          </RouterLink>
        </nav>

        <div class="mx-auto h-px w-8 shrink-0 bg-sidebar-border/80" aria-hidden="true" />

        <nav class="flex flex-col items-center gap-2" :aria-label="t('nav.yours')">
          <RouterLink
            v-for="item in sidebarNavGroups.yours"
            :key="item.page"
            :to="getNavigationTarget(item.page)"
            :title="item.label"
            class="flex size-10 shrink-0 items-center justify-center rounded-2xl text-sidebar-foreground transition-colors outline-none hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
            :class="isActive(item.page) ? 'bg-sidebar-accent text-sidebar-accent-foreground' : ''"
          >
            <component :is="item.icon" class="size-5 shrink-0" />
          </RouterLink>
        </nav>
      </div>
    </div>

    <Separator
      class="my-2.5 shrink-0 bg-sidebar-border/80"
      :class="props.compact ? 'w-10' : ''"
    />

    <section
      v-if="!props.compact"
      class="mb-2 flex min-w-0 flex-col gap-2"
    >
      <div
        class="flex min-w-0 items-center gap-2 rounded-2xl border border-border/50 bg-background/35 px-3 py-2"
        role="status"
        :aria-label="backendAriaLabel"
        :aria-live="backendUseWebApi ? 'polite' : 'off'"
      >
        <span class="mt-0.5 size-2 shrink-0 rounded-full" :class="backendDotClass" aria-hidden="true" />
        <div class="min-w-0 flex-1">
          <div
            v-if="backendStatus !== 'online'"
            class="truncate text-xs text-muted-foreground"
          >
            {{ backendStatusText }}
          </div>
          <div
            v-if="backendMetaText"
            class="truncate text-muted-foreground/80"
            :class="backendStatus === 'online' ? 'text-xs' : 'text-[11px]'"
            :title="backendMetaText"
          >
            {{ backendMetaText }}
          </div>
        </div>
        <Button
          v-if="backendUseWebApi"
          type="button"
          variant="ghost"
          size="icon"
          class="size-8 shrink-0 rounded-xl text-muted-foreground hover:text-foreground"
          :title="t('nav.backendRecheck')"
          :aria-label="t('nav.backendRecheck')"
          :disabled="backendProbing"
          @click="checkBackendHealth"
        >
          <RefreshCw
            class="size-4"
            :class="{ 'motion-safe:animate-spin': backendProbing }"
          />
        </Button>
      </div>
    </section>

    <div
      v-else
      class="mb-2 flex w-full shrink-0 items-center justify-center gap-1"
      role="status"
      :aria-label="backendAriaLabel"
      :aria-live="backendUseWebApi ? 'polite' : 'off'"
    >
      <span
        class="size-2.5 shrink-0 rounded-full"
        :class="backendDotClass"
        :title="backendCompactTitle"
      />
      <Button
        v-if="backendUseWebApi"
        type="button"
        variant="ghost"
        size="icon"
        class="size-7 shrink-0 rounded-xl text-muted-foreground hover:text-foreground"
        :title="t('nav.backendRecheck')"
        :aria-label="t('nav.backendRecheck')"
        :disabled="backendProbing"
        @click="checkBackendHealth"
      >
        <RefreshCw
          class="size-3.5"
          :class="{ 'motion-safe:animate-spin': backendProbing }"
        />
      </Button>
    </div>

    <Button
      v-if="!props.compact"
      as-child
      :variant="isActive('settings') ? 'secondary' : 'ghost'"
      class="w-full justify-start rounded-2xl px-3"
    >
      <RouterLink
        :to="getNavigationTarget('settings')"
        class="flex w-full items-center gap-2"
      >
        <Settings2 data-icon="inline-start" />
        <span>{{ t("nav.settings") }}</span>
      </RouterLink>
    </Button>

    <RouterLink
      v-else
      :to="getNavigationTarget('settings')"
      :title="t('nav.settings')"
      class="flex size-10 shrink-0 items-center justify-center rounded-2xl text-sidebar-foreground transition-colors outline-none hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
      :class="isActive('settings') ? 'bg-sidebar-accent text-sidebar-accent-foreground' : ''"
    >
      <Settings2 class="size-5 shrink-0" />
    </RouterLink>
  </aside>
</template>
