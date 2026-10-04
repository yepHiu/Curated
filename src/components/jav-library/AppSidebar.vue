<script setup lang="ts">
import type { Component } from "vue"
import { computed, onMounted } from "vue"
import { useI18n } from "vue-i18n"
import {
  BookOpen,
  Clapperboard,
  ChartNoAxesColumnIncreasing,
  History,
  House,
  Images,
  LibraryBig,
  RefreshCw,
  Settings2,
  Trash2,
  Users,
} from "lucide-vue-next"
import { RouterLink, useRoute } from "vue-router"
import type { AppPage, LibraryMode } from "@/domain/library/types"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import SidebarPlaybackEntry from "@/components/jav-library/SidebarPlaybackEntry.vue"
import SidebarTagOrganizationEntry from "@/components/jav-library/SidebarTagOrganizationEntry.vue"
import { useBackendHealth } from "@/composables/use-backend-health"
import { buildBrowseRouteTarget } from "@/lib/library-query"
import { statusDotClass } from "@/lib/ui/status-tone"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"
import curatedMark from "@/icon/curated-mark.png"

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
const comicService = useComicLibraryService()
const photoService = usePhotoLibraryService()
const {
  useWebApi: backendUseWebApi,
  status: backendStatus,
  probing: backendProbing,
  versionDisplay: backendVersionDisplay,
  checkNow: checkBackendHealth,
} = useBackendHealth()

onMounted(() => {
  void Promise.resolve(comicService.refreshSettings()).catch((error) => {
    console.warn("[sidebar] comic settings refresh failed", error)
  })
  void Promise.resolve(photoService.refreshSettings()).catch((error) => {
    console.warn("[sidebar] photo settings refresh failed", error)
  })
})

const comicLibraryEnabled = computed(() => comicService.comicLibraryEnabled.value)
const photoLibraryEnabled = computed(() => photoService.photoLibraryEnabled.value)

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

const sidebarNavGroups = computed(/* 使用同一导航结构展示影片和 FC2 入口。 */ (): SidebarNavGroups => {
  void locale.value
  const browse: NavigationItem[] = [
    { label: t("nav.home"), page: "home", icon: House },
    { label: t("nav.library"), page: "library", icon: LibraryBig },
    { label: t("nav.fc2"), page: "fc2", icon: Clapperboard },
  ]

  if (comicLibraryEnabled.value) {
    browse.push({ label: t("nav.comics"), page: "comics", icon: BookOpen })
  }
  if (photoLibraryEnabled.value) {
    browse.push({ label: t("nav.photos"), page: "photos", icon: Images })
  }

  browse.push(
    { label: t("nav.actors"), page: "actors", icon: Users },
    { label: t("nav.trash"), page: "trash", icon: Trash2 },
  )

  return {
    browse,
    yours: [
      { label: t("wishlist.title"), page: "wishlist", icon: LibraryBig },
      { label: t("nav.insights"), page: "insights", icon: ChartNoAxesColumnIncreasing },
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

const isActive = (page: AppPage) => {
  if (page === "wishlist") return String(route.name).startsWith("wishlist")
  if (page === "actors") {
    return route.name === "actors" || route.name === "actor-detail"
  }
  if (page === "comics") {
    return ["comics", "comic-detail", "comic-reader"].includes(String(route.name ?? ""))
  }
  if (page === "photos") {
    return ["photos", "photo-detail", "photo-viewer"].includes(String(route.name ?? ""))
  }
  return route.name === page
}

const brandHomeTarget = computed(() => ({ name: "home" as const }))
const brandName = typeof window !== "undefined" && window.javLibrary
  ? "Curated Desktop"
  : "Curated Web"

const getNavigationTarget = (page: AppPage) => {
  if (page === "wishlist") return { name: "wishlist" }
  if (page === "home") {
    return { name: "home" }
  }
  if (page === "settings") {
    return { name: page }
  }
  if (page === "history") {
    return { name: "history" }
  }
  if (page === "insights") {
    return { name: "insights" }
  }
  if (page === "curated-frames") {
    return { name: "curated-frames" }
  }
  if (page === "actors") {
    return { name: "actors" }
  }
  if (page === "comics") {
    return { name: "comics" }
  }
  if (page === "photos") {
    return { name: "photos" }
  }
  return buildBrowseRouteTarget(page as LibraryMode, route.query)
}
</script>

<template>
  <aside
    :data-sidebar-compact="props.compact"
    class="flex h-full min-h-0 w-full min-w-0 flex-col overflow-x-hidden bg-sidebar text-sidebar-foreground motion-reduce:transition-none"
    :class="props.compact ? 'px-2 pb-3 pt-0' : 'px-3.5 pb-3.5 pt-0'"
  >
    <div
      data-sidebar-header
      class="flex min-h-[var(--app-header-min-height)] shrink-0 items-center"
      :class="props.compact ? 'justify-center py-[var(--app-header-py)] lg:py-[var(--app-header-py-lg)]' : 'justify-between gap-2 px-2 py-[var(--app-header-py)] sm:px-2 lg:px-2 lg:py-[var(--app-header-py-lg)]'"
    >
      <div
        class="flex min-w-0 items-center"
        :class="props.compact ? 'w-full justify-center' : 'w-full min-w-0 justify-start gap-2'"
      >
        <RouterLink
          :to="brandHomeTarget"
          data-sidebar-brand-link
          class="font-curated inline-flex items-center px-1 py-1 font-semibold tracking-wide text-primary"
          :class="
            props.compact
              ? 'w-full max-w-full justify-center gap-0 text-base'
              : 'min-w-0 w-fit max-w-full flex-1 gap-2 text-lg'
          "
          :title="`${brandName} · ${t('nav.home')}`"
          :aria-label="`${brandName} · ${t('nav.home')}`"
        >
          <img :src="curatedMark" class="size-7 shrink-0 object-contain" alt="" aria-hidden="true" />
          <span
            class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
            :class="props.compact ? 'max-w-0 opacity-0' : 'max-w-full opacity-100'"
            :aria-hidden="props.compact"
          >
            {{ brandName }}
          </span>
        </RouterLink>
      </div>
    </div>

    <div class="app-sidebar-scroll min-h-0 w-full min-w-0 flex-1 overflow-x-hidden overflow-y-auto overscroll-contain">
      <div class="flex flex-col pt-1.5" :class="props.compact ? 'gap-3 pb-1' : 'gap-5'">
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
    </div>

    <Separator
      class="my-2.5 shrink-0 bg-sidebar-border/80"
      :class="props.compact ? 'mx-auto w-10' : ''"
    />

    <SidebarPlaybackEntry :compact="props.compact" />
    <SidebarTagOrganizationEntry :compact="props.compact" />

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

<style scoped>
.app-sidebar-scroll {
  scrollbar-width: none;
}

.app-sidebar-scroll::-webkit-scrollbar {
  display: none;
}
</style>
