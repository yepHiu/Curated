<script setup lang="ts">
import type { Component } from "vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import {
  Clapperboard,
  History,
  House,
  LibraryBig,
  Play,
  RefreshCw,
  Settings2,
  Sparkles,
  Tags,
  Trash2,
  Users,
  X,
} from "lucide-vue-next"
import { RouterLink, useRoute } from "vue-router"
import type { AppPage, LibraryMode } from "@/domain/library/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { useAppUpdate } from "@/composables/use-app-update"
import { useActivePlaybackSession } from "@/composables/use-active-playback-session"
import { useBackendHealth } from "@/composables/use-backend-health"
import { buildBrowseRouteTarget } from "@/lib/library-query"
import { statusDotClass } from "@/lib/ui/status-tone"

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
const {
  useWebApi: backendUseWebApi,
  status: backendStatus,
  probing: backendProbing,
  versionDisplay: backendVersionDisplay,
  checkNow: checkBackendHealth,
} = useBackendHealth()
const { hasUpdateBadge, summary: appUpdateSummary } = useAppUpdate()
const {
  activePlaybackSession,
  dismissActivePlaybackSession,
} = useActivePlaybackSession()

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

const sidebarNavGroups = computed((): SidebarNavGroups => {
  void locale.value

  return {
    browse: [
      { label: t("nav.home"), page: "home", icon: House },
      { label: t("nav.library"), page: "library", icon: LibraryBig },
      { label: t("nav.actors"), page: "actors", icon: Users },
      { label: t("nav.tags"), page: "tags", icon: Tags },
      { label: t("nav.trash"), page: "trash", icon: Trash2 },
    ],
    yours: [
      { label: t("nav.curatedFrames"), page: "curated-frames", icon: Clapperboard },
      { label: t("nav.history"), page: "history", icon: History },
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

const brandHomeTarget = computed(() => ({ name: "home" as const }))

const brandAboutUpdateTarget = computed(() => ({
  name: "settings" as const,
  query: { ...route.query, section: "about" },
}))

const brandUpdateTitle = computed(() => {
  if (!hasUpdateBadge.value) {
    return "Curated"
  }
  const latest = appUpdateSummary.value?.latestVersion?.trim()
  return latest
    ? `Curated\nNew version ${latest}`
    : "Curated\nNew version available"
})

function formatSidebarPlaybackClock(seconds: number): string {
  const total = Math.max(0, Math.floor(Number.isFinite(seconds) ? seconds : 0))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(secs).padStart(2, "0")}`
  }
  return `${minutes}:${String(secs).padStart(2, "0")}`
}

const isSameActivePlayerRoute = computed(() => {
  const active = activePlaybackSession.value
  if (!active || route.name !== "player") return false
  return typeof route.params.id === "string" && route.params.id === active.movieId
})

const showActivePlaybackEntry = computed(() =>
  Boolean(activePlaybackSession.value && !isSameActivePlayerRoute.value),
)

const activePlaybackTimeLabel = computed(() => {
  const active = activePlaybackSession.value
  if (!active) return ""
  return formatSidebarPlaybackClock(active.positionSec)
})

const activePlaybackProgressValue = computed(() =>
  Math.max(0, Math.min(100, activePlaybackSession.value?.progressPercent ?? 0)),
)

const activePlaybackAriaLabel = computed(() => {
  const active = activePlaybackSession.value
  if (!active) return t("nav.continuePlayback")
  return t("nav.continuePlaybackAria", {
    title: active.title,
    time: activePlaybackTimeLabel.value,
  })
})

const activePlaybackCompactTitle = computed(() => {
  const active = activePlaybackSession.value
  if (!active) return t("nav.continuePlayback")
  return `${t("nav.continuePlayback")}: ${active.title} · ${activePlaybackTimeLabel.value}`
})

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
      class="flex min-h-[var(--app-header-min-height)] shrink-0 items-center border-b border-sidebar-border/80"
      :class="props.compact ? 'justify-center py-[var(--app-header-py)] lg:py-[var(--app-header-py-lg)]' : 'justify-between gap-2 px-2 py-[var(--app-header-py)] sm:px-2 lg:px-2 lg:py-[var(--app-header-py-lg)]'"
    >
      <div
        class="flex min-w-0 items-center"
        :class="props.compact ? 'w-full justify-center' : 'w-full min-w-0 justify-start gap-2'"
      >
        <RouterLink
          :to="brandHomeTarget"
          data-sidebar-brand-link
          class="font-curated inline-flex items-center gap-2 px-1 py-1 font-semibold tracking-wide text-primary"
          :class="
            props.compact
              ? 'w-full max-w-full justify-center text-base'
              : 'min-w-0 w-fit min-w-0 max-w-full flex-1 text-lg sm:text-xl'
          "
          :title="t('nav.home')"
        >
          <span class="relative inline-flex items-center">
            <Sparkles class="size-5 shrink-0 text-primary sm:size-[1.35rem]" aria-hidden="true" />
            <span
              v-if="props.compact && hasUpdateBadge"
              data-update-dot
              class="absolute -right-1 -top-1 size-2.5 rounded-full border border-sidebar bg-primary shadow-[0_0_0_3px_rgba(254,98,142,0.16)]"
              aria-hidden="true"
            />
          </span>
          <span
            class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
            :class="props.compact ? 'max-w-0 opacity-0' : 'max-w-[10rem] opacity-100'"
            :aria-hidden="props.compact"
          >
            Curated
          </span>
        </RouterLink>
        <RouterLink
          v-if="!props.compact && hasUpdateBadge"
          :to="brandAboutUpdateTarget"
          data-update-badge-link
          :title="brandUpdateTitle"
          :aria-label="brandUpdateTitle"
          class="inline-flex shrink-0"
        >
          <Badge
            data-update-badge
            variant="secondary"
            class="pointer-events-none rounded-full border border-primary/25 bg-primary/12 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-normal text-primary"
          >
            New
          </Badge>
        </RouterLink>
      </div>
    </div>

    <ScrollArea class="min-h-0 w-full min-w-0 flex-1">
      <div class="flex flex-col pt-3.5" :class="props.compact ? 'gap-3 pb-1' : 'gap-5'">
        <template v-for="(section, sectionIndex) in sidebarSections" :key="section.key">
          <section
            class="flex flex-col gap-2"
            :class="{ 'pr-2.5': !props.compact }"
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
              class="group flex min-w-0 items-center text-sidebar-foreground outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/60"
              :class="[
                isActive(item.page)
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                  : 'hover:bg-sidebar-accent/60',
                props.compact
                  ? 'mx-auto size-10 shrink-0 justify-center rounded-lg p-0'
                  : 'min-h-10 w-full justify-between rounded-2xl px-3',
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
            </RouterLink>
          </section>

          <Separator
            v-if="sectionIndex === 0"
            class="my-2.5 shrink-0 bg-sidebar-border/80"
            :class="props.compact ? 'mx-auto w-10' : ''"
          />
        </template>
      </div>
    </ScrollArea>

    <Separator
      class="my-2.5 shrink-0 bg-sidebar-border/80"
      :class="props.compact ? 'mx-auto w-10' : ''"
    />

    <section
      v-if="showActivePlaybackEntry && activePlaybackSession"
      class="mb-2 min-w-0"
      :class="props.compact ? 'flex justify-center' : 'flex flex-col'"
    >
      <div
        v-if="!props.compact"
        class="relative min-w-0 rounded-lg border border-border/60 bg-background/45"
      >
        <RouterLink
          data-active-playback-card
          :to="activePlaybackSession.resumeRouteTarget"
          class="group flex min-w-0 flex-col gap-2 rounded-lg px-3 py-2.5 pr-10 text-sidebar-foreground outline-none transition-colors hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
          :aria-label="activePlaybackAriaLabel"
        >
          <span class="flex min-w-0 items-center justify-between gap-2">
            <span class="inline-flex min-w-0 items-center gap-2 text-xs font-medium text-primary">
              <span class="size-2 shrink-0 rounded-full bg-primary shadow-[0_0_0_4px_hsl(var(--primary)/0.16)]" aria-hidden="true" />
              <span class="truncate">{{ t("nav.continuePlayback") }}</span>
            </span>
            <span class="inline-flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Play class="size-3.5 fill-current" aria-hidden="true" />
            </span>
          </span>
          <span class="line-clamp-2 min-w-0 text-sm font-medium leading-snug">
            {{ activePlaybackSession.title }}
          </span>
          <span class="flex min-w-0 items-center justify-between gap-2 text-xs text-muted-foreground">
            <span class="truncate">
              {{ t("nav.continuePlaybackAt", { time: activePlaybackTimeLabel }) }}
            </span>
            <span class="shrink-0 tabular-nums">{{ Math.round(activePlaybackProgressValue) }}%</span>
          </span>
          <Progress
            :model-value="activePlaybackProgressValue"
            class="h-1 bg-primary/15"
            aria-hidden="true"
          />
        </RouterLink>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          data-active-playback-dismiss
          class="absolute right-1.5 top-1.5 size-7 rounded-lg text-muted-foreground hover:text-foreground"
          :aria-label="t('nav.dismissContinuePlayback')"
          @click="dismissActivePlaybackSession(activePlaybackSession.movieId)"
        >
          <X class="size-3.5" aria-hidden="true" />
        </Button>
      </div>

      <RouterLink
        v-else
        data-active-playback-compact
        :to="activePlaybackSession.resumeRouteTarget"
        class="relative mx-auto inline-flex size-10 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/45 text-primary outline-none transition-colors hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
        :title="activePlaybackCompactTitle"
        :aria-label="activePlaybackAriaLabel"
      >
        <Play class="size-4 fill-current" aria-hidden="true" />
        <span class="absolute inset-x-1.5 bottom-1.5 h-0.5 overflow-hidden rounded-full bg-primary/15" aria-hidden="true">
          <span
            class="block h-full rounded-full bg-primary"
            :style="{ width: `${activePlaybackProgressValue}%` }"
          />
        </span>
      </RouterLink>
    </section>

    <section
      v-if="!props.compact"
      class="mb-2 flex min-w-0 flex-col gap-2"
    >
      <div
        class="flex min-w-0 items-center gap-2 rounded-lg border border-border/60 bg-background/45 px-3 py-2"
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

    <RouterLink
      data-sidebar-nav-link
      :to="getNavigationTarget('settings')"
      :title="props.compact ? t('nav.settings') : undefined"
      class="group flex min-w-0 items-center text-sidebar-foreground outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/60"
      :class="[
        isActive('settings')
          ? 'bg-sidebar-accent text-sidebar-accent-foreground'
          : 'hover:bg-sidebar-accent/60',
        props.compact
          ? 'mx-auto size-10 shrink-0 justify-center rounded-lg p-0'
          : 'min-h-10 w-full justify-between rounded-2xl px-3',
      ]"
    >
      <span
        class="flex min-w-0 items-center overflow-hidden"
        :class="props.compact ? 'justify-center' : 'gap-2 truncate'"
      >
        <Settings2 class="size-5 shrink-0" data-icon="inline-start" />
        <span
          class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
          :class="props.compact ? 'max-w-0 opacity-0' : 'max-w-[10rem] opacity-100'"
          :aria-hidden="props.compact"
        >
          {{ t("nav.settings") }}
        </span>
      </span>
    </RouterLink>
  </aside>
</template>
