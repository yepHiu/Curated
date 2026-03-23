<script setup lang="ts">
import type { Component } from "vue"
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  ChevronLeft,
  ChevronRight,
  Clock3,
  Clapperboard,
  History,
  LibraryBig,
  RefreshCw,
  Settings2,
  Sparkles,
  Tags,
  Users,
} from "lucide-vue-next"
import { RouterLink, useRoute } from "vue-router"
import type { AppPage, LibraryMode } from "@/domain/library/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { buildBrowseRouteTarget } from "@/lib/library-query"
import { countCuratedFrames } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import {
  listSortedByUpdatedDesc,
  playbackProgressRevision,
} from "@/lib/playback-progress-storage"
import {
  countUniqueTags,
  formatSidebarCount,
  isMovieRecentlyAdded,
} from "@/lib/library-stats"
import { useBackendHealth } from "@/composables/use-backend-health"
import { useLibraryService } from "@/services/library-service"

const props = withDefaults(
  defineProps<{
    /** 桌面宽屏：仅显示图标窄条 */
    compact?: boolean
    /** 是否显示「收起为窄条 / 展开」按钮（仅桌面栅格内使用） */
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

const { t, locale } = useI18n()
const route = useRoute()
const libraryService = useLibraryService()
const { useWebApi: backendUseWebApi, status: backendStatus, probing: backendProbing, checkNow: checkBackendHealth } =
  useBackendHealth()

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

const browseItems = computed((): NavigationItem[] => {
  void locale.value
  void playbackProgressRevision.value
  void curatedFramesRevision.value
  const movies = libraryService.movies.value
  const total = movies.length
  const recent = movies.filter((m) => isMovieRecentlyAdded(m.addedAt)).length
  const tagCount = countUniqueTags(movies)
  const actorSet = new Set<string>()
  for (const m of movies) {
    for (const raw of m.actors) {
      const a = raw.trim()
      if (a) actorSet.add(a)
    }
  }
  const actorCount = actorSet.size

  const historyTotal = listSortedByUpdatedDesc().length
  const framesTotal = curatedFrameCount.value

  return [
    { label: t("nav.library"), page: "library", icon: LibraryBig, hint: formatSidebarCount(total) },
    { label: t("nav.recent"), page: "recent", icon: Clock3, hint: formatSidebarCount(recent) },
    { label: t("nav.tags"), page: "tags", icon: Tags, hint: formatSidebarCount(tagCount) },
    {
      label: t("nav.actors"),
      page: "actors",
      icon: Users,
      hint: actorCount > 0 ? formatSidebarCount(actorCount) : undefined,
    },
    {
      label: t("nav.history"),
      page: "history",
      icon: History,
      hint: historyTotal > 0 ? formatSidebarCount(historyTotal) : undefined,
    },
    {
      label: t("nav.curatedFrames"),
      page: "curated-frames",
      icon: Clapperboard,
      hint: framesTotal > 0 ? formatSidebarCount(framesTotal) : undefined,
    },
  ]
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
    class="flex h-full min-h-0 w-full min-w-0 flex-col rounded-3xl border border-border/70 bg-sidebar/95 text-sidebar-foreground shadow-2xl shadow-black/20 backdrop-blur"
    :class="props.compact ? 'items-center px-2 py-3' : 'p-4'"
  >
    <!-- 桌面完整：标题 + 可选收起 -->
    <div
      v-if="!props.compact"
      class="flex items-center justify-between gap-2 px-2 py-3"
    >
      <div class="flex min-w-0 flex-col gap-2">
        <div
          class="font-curated inline-flex w-fit max-w-full items-center gap-2 rounded-full bg-[#2d1b2d] px-3.5 py-2.5 text-lg font-semibold tracking-wide text-[#FF6B9B] shadow-inner shadow-black/20 sm:text-xl"
          title="Curated"
        >
          <Sparkles class="size-5 shrink-0 text-[#FF6B9B] sm:size-[1.35rem]" aria-hidden="true" />
          <span class="truncate">Curated</span>
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-1 self-start pt-0.5">
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

    <!-- 桌面窄条：展开按钮 + 品牌图标 -->
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
        class="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-[#2d1b2d] text-[#FF6B9B]"
        title="Curated"
      >
        <Sparkles class="size-5 text-[#FF6B9B]" />
      </div>
    </div>

    <Separator
      class="my-3 shrink-0 bg-sidebar-border/80"
      :class="props.compact ? 'w-10' : ''"
    />

    <!-- 折叠时不用 ScrollArea，避免右侧滚动条占位导致图标视觉偏移 -->
    <ScrollArea v-if="!props.compact" class="min-h-0 w-full min-w-0 flex-1">
      <div class="flex flex-col gap-6 pr-3">
        <section class="flex flex-col gap-2">
          <span
            class="px-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground"
          >
            {{ t("nav.browse") }}
          </span>
          <Button
            v-for="item in browseItems"
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

    <!-- 折叠导航：不用 Button as-child，避免 Reka 合并样式后主区域高度/子节点异常；自占满列宽以便居中 -->
    <div
      v-if="props.compact"
      class="flex min-h-0 w-full min-w-0 flex-1 flex-col self-stretch overflow-y-auto"
    >
      <nav
        class="flex flex-col items-center gap-2 py-1"
        :aria-label="t('nav.browse')"
      >
        <RouterLink
          v-for="item in browseItems"
          :key="item.page"
          :to="getNavigationTarget(item.page)"
          :title="item.label"
          class="flex size-10 shrink-0 items-center justify-center rounded-2xl text-sidebar-foreground transition-colors outline-none hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
          :class="
            isActive(item.page)
              ? 'bg-sidebar-accent text-sidebar-accent-foreground'
              : ''
          "
        >
          <component :is="item.icon" class="size-5 shrink-0" />
        </RouterLink>
      </nav>
    </div>

    <Separator
      class="my-3 shrink-0 bg-sidebar-border/80"
      :class="props.compact ? 'w-10' : ''"
    />

    <!-- 后端在线检测（紧贴设置项上方） -->
    <section
      v-if="!props.compact"
      class="mb-2 flex min-w-0 flex-col gap-2"
    >
      <div
        class="flex min-w-0 items-center gap-2 rounded-2xl border border-border/50 bg-background/35 px-3 py-2"
        role="status"
        :aria-label="`${t('nav.backendLabel')}：${backendStatusText}`"
        :aria-live="backendUseWebApi ? 'polite' : 'off'"
      >
        <span
          class="size-2 shrink-0 rounded-full"
          :class="backendDotClass"
          aria-hidden="true"
        />
        <span class="min-w-0 flex-1 truncate text-xs text-muted-foreground">
          {{ backendStatusText }}
        </span>
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
          <RefreshCw class="size-4" :class="{ 'animate-spin': backendProbing }" />
        </Button>
      </div>
    </section>
    <div
      v-else
      class="mb-2 flex w-full shrink-0 items-center justify-center gap-1"
      role="status"
      :aria-label="`${t('nav.backendLabel')}：${backendStatusText}`"
      :aria-live="backendUseWebApi ? 'polite' : 'off'"
    >
      <span
        class="size-2.5 shrink-0 rounded-full"
        :class="backendDotClass"
        :title="backendStatusText"
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
        <RefreshCw class="size-3.5" :class="{ 'animate-spin': backendProbing }" />
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
      :class="
        isActive('settings')
          ? 'bg-sidebar-accent text-sidebar-accent-foreground'
          : ''
      "
    >
      <Settings2 class="size-5 shrink-0" />
    </RouterLink>
  </aside>
</template>
