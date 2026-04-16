<script setup lang="ts">
import type { Component } from "vue"
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  Clapperboard,
  History,
  House,
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
import { statusDotClass } from "@/lib/ui/status-tone"
import { useLibraryService } from "@/services/library-service"

const props = withDefaults(
  defineProps<{
    compact?: boolean
  }>(),
  {
    compact: false,
  },
)

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

interface SidebarNavSection {
  key: "browse" | "yours"
  title: string
  items: NavigationItem[]
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
      return statusDotClass("success")
    case "offline":
      return statusDotClass("danger")
    case "mock":
      return statusDotClass("warning")
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
      { label: t("nav.home"), page: "home", icon: House },
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

const sidebarSections = computed((): SidebarNavSection[] => [
  {
    key: "browse",
    title: t("nav.browse"),
    items: sidebarNavGroups.value.browse,
  },
  {
    key: "yours",
    title: t("nav.yours"),
    items: sidebarNavGroups.value.yours,
  },
])

const isActive = (page: AppPage) => route.name === page

const getNavigationTarget = (page: AppPage) => {
  if (page === "home") {
    return { name: "home" }
  }
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
    class="flex h-full min-h-0 w-full min-w-0 flex-col overflow-x-hidden bg-sidebar text-sidebar-foreground motion-reduce:transition-none"
    :class="props.compact ? 'px-2 pb-3 pt-0' : 'px-3.5 pb-3.5 pt-0'"
  >
    <div
      class="flex min-h-[4.5rem] shrink-0 items-center border-b border-sidebar-border/80"
      :class="props.compact ? 'justify-center py-3.5 lg:py-4' : 'justify-between gap-2 px-2 py-3.5 sm:px-2 lg:px-2 lg:py-4'"
    >
      <div
        class="font-curated inline-flex w-fit max-w-full items-center gap-2 px-1 py-1 font-semibold tracking-wide text-primary"
        :class="props.compact ? 'justify-center text-base' : 'text-lg sm:text-xl'"
        title="Curated"
      >
        <Sparkles class="size-5 shrink-0 text-primary sm:size-[1.35rem]" aria-hidden="true" />
        <span
          class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
          :class="props.compact ? 'max-w-0 opacity-0' : 'max-w-[10rem] opacity-100'"
          :aria-hidden="props.compact"
        >
          Curated
        </span>
      </div>
    </div>

    <ScrollArea class="min-h-0 w-full min-w-0 flex-1">
      <div class="flex flex-col pt-3.5" :class="props.compact ? 'gap-3 pb-1' : 'gap-5 pr-2.5'">
        <section
          v-for="section in sidebarSections"
          :key="section.key"
          class="flex flex-col gap-2"
          :aria-label="section.title"
        >
          <span
            class="overflow-hidden text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground transition-[opacity,max-height,padding] duration-200 motion-reduce:transition-none"
            :class="props.compact ? 'max-h-0 px-0 opacity-0' : 'max-h-8 px-2 opacity-100'"
            :aria-hidden="props.compact"
          >
            {{ section.title }}
          </span>

          <RouterLink
            v-for="item in section.items"
            :key="item.page"
            :to="getNavigationTarget(item.page)"
            data-sidebar-nav-link
            :title="props.compact ? item.label : undefined"
            class="group flex min-h-10 w-full min-w-0 items-center rounded-2xl text-sidebar-foreground outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/60"
            :class="[
              isActive(item.page)
                ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                : 'hover:bg-sidebar-accent/60',
              props.compact ? 'justify-center px-2' : 'justify-between px-3',
            ]"
          >
            <span
              class="flex min-w-0 items-center overflow-hidden"
              :class="props.compact ? 'justify-center' : 'gap-2 truncate'"
            >
              <component :is="item.icon" class="size-5 shrink-0" data-icon="inline-start" />
              <span
                class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
                :class="props.compact ? 'max-w-0 opacity-0' : 'max-w-[10rem] opacity-100'"
                :aria-hidden="props.compact"
              >
                {{ item.label }}
              </span>
            </span>

            <span
              v-if="item.hint"
              class="shrink-0 transition-[opacity,width] duration-150 motion-reduce:transition-none"
              :class="props.compact ? 'w-0 overflow-hidden opacity-0' : 'w-auto opacity-100'"
              :aria-hidden="props.compact"
            >
              <Badge
                variant="secondary"
                class="rounded-full border border-border/60 bg-background/60"
              >
                {{ item.hint }}
              </Badge>
            </span>
          </RouterLink>
        </section>
      </div>
    </ScrollArea>

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
      class="mb-2 flex w-full shrink-0 items-center justify-center"
      role="status"
      :aria-label="backendAriaLabel"
      :aria-live="backendUseWebApi ? 'polite' : 'off'"
    >
      <span
        class="size-2.5 shrink-0 rounded-full"
        :class="backendDotClass"
        :title="backendCompactTitle"
      />
    </div>

    <Button
      as-child
      :variant="isActive('settings') ? 'secondary' : 'ghost'"
      class="w-full rounded-2xl"
    >
      <RouterLink
      data-sidebar-nav-link
      :to="getNavigationTarget('settings')"
        :title="props.compact ? t('nav.settings') : undefined"
        class="flex min-h-10 w-full min-w-0 items-center rounded-2xl text-sidebar-foreground outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/60"
        :class="[
          isActive('settings')
            ? 'bg-sidebar-accent text-sidebar-accent-foreground'
            : 'hover:bg-sidebar-accent/60',
          props.compact ? 'justify-center px-2' : 'justify-start px-3',
        ]"
      >
        <Settings2 class="size-5 shrink-0" data-icon="inline-start" />
        <span
          class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
          :class="props.compact ? 'max-w-0 opacity-0' : 'max-w-[10rem] pl-2 opacity-100'"
          :aria-hidden="props.compact"
        >
          {{ t("nav.settings") }}
        </span>
      </RouterLink>
    </Button>
  </aside>
</template>
