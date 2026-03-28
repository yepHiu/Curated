<script setup lang="ts">
import {
  inject,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  type Ref,
} from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { SETTINGS_SCROLL_EL_KEY } from "@/composables/use-settings-scroll-preserve"
import {
  type SettingsSectionSlug,
  SETTINGS_NAV_ITEMS,
  isSettingsSectionSlug,
  settingsSectionDomId,
} from "@/lib/settings-nav"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

const props = defineProps<{
  /** 与 `SETTINGS_NAV_ITEMS` 对齐；默认使用全局列表 */
  items?: typeof SETTINGS_NAV_ITEMS
}>()

const navItems = props.items ?? SETTINGS_NAV_ITEMS

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const scrollElRef = inject<Ref<HTMLElement | null>>(SETTINGS_SCROLL_EL_KEY, ref(null))

const activeSlug = ref<SettingsSectionSlug>(navItems[0]?.slug ?? "overview")

let programmaticLock = false
let lockTimer: ReturnType<typeof setTimeout> | null = null
let queryDebounce: ReturnType<typeof setTimeout> | null = null
let scrollListenerAttached: HTMLElement | null = null

function clearLockTimer() {
  if (lockTimer) {
    clearTimeout(lockTimer)
    lockTimer = null
  }
}

function setProgrammaticLock(ms = 520) {
  programmaticLock = true
  clearLockTimer()
  lockTimer = setTimeout(() => {
    programmaticLock = false
    lockTimer = null
  }, ms)
}

function computeActiveSlugFromScroll(): SettingsSectionSlug {
  const root = scrollElRef.value
  if (!root) return navItems[0]?.slug ?? "overview"

  const rootRect = root.getBoundingClientRect()
  const lineY = rootRect.top + Math.min(120, rootRect.height * 0.12)

  let chosen = navItems[0]?.slug ?? "overview"
  for (const item of navItems) {
    const el = document.getElementById(settingsSectionDomId(item.slug))
    if (!el) continue
    const top = el.getBoundingClientRect().top
    if (top <= lineY) {
      chosen = item.slug
    }
  }
  return chosen
}

function syncActiveFromScroll() {
  if (programmaticLock) return
  const next = computeActiveSlugFromScroll()
  if (next !== activeSlug.value) {
    activeSlug.value = next
  }
  if (queryDebounce) clearTimeout(queryDebounce)
  queryDebounce = setTimeout(() => {
    queryDebounce = null
    const slug = activeSlug.value
    if (route.query.section !== slug) {
      router.replace({ query: { ...route.query, section: slug } }).catch(() => {})
    }
  }, 320)
}

function attachScrollListener() {
  const root = scrollElRef.value
  if (!root || scrollListenerAttached === root) return
  detachScrollListener()
  scrollListenerAttached = root
  root.addEventListener("scroll", syncActiveFromScroll, { passive: true })
}

function detachScrollListener() {
  if (scrollListenerAttached) {
    scrollListenerAttached.removeEventListener("scroll", syncActiveFromScroll)
    scrollListenerAttached = null
  }
}

/**
 * 仅在设置页滚动根上滚动，避免 `scrollIntoView` 连锁滚动 window/document 导致「整页」上浮。
 */
function scrollSectionIntoSettingsRoot(slug: SettingsSectionSlug, smooth: boolean) {
  const root = scrollElRef.value
  const id = settingsSectionDomId(slug)
  const el = document.getElementById(id)
  if (!root || !el) return
  if (!root.contains(el)) return

  const rootRect = root.getBoundingClientRect()
  const elRect = el.getBoundingClientRect()
  const scrollMarginTop = Number.parseFloat(getComputedStyle(el).scrollMarginTop) || 0
  const targetTop = elRect.top - rootRect.top + root.scrollTop - scrollMarginTop

  if (smooth) {
    root.scrollTo({ top: Math.max(0, targetTop), behavior: "smooth" })
  } else {
    root.scrollTop = Math.max(0, targetTop)
  }
}

function scrollToSlug(slug: SettingsSectionSlug, smooth: boolean) {
  if (!document.getElementById(settingsSectionDomId(slug))) return
  setProgrammaticLock()
  scrollSectionIntoSettingsRoot(slug, smooth)
  activeSlug.value = slug
  const nextQuery = { ...route.query, section: slug }
  if (route.query.section !== slug) {
    router.replace({ query: nextQuery }).catch(() => {})
  }
}

function applyInitialSectionFromQuery() {
  const raw = route.query.section
  const s = typeof raw === "string" ? raw : Array.isArray(raw) ? raw[0] : undefined
  if (s && isSettingsSectionSlug(s)) {
    nextTick(() => {
      requestAnimationFrame(() => scrollToSlug(s, false))
    })
  }
}

onMounted(() => {
  nextTick(() => {
    attachScrollListener()
    applyInitialSectionFromQuery()
    syncActiveFromScroll()
  })
})

watch(
  () => scrollElRef.value,
  () => {
    nextTick(() => {
      detachScrollListener()
      attachScrollListener()
      syncActiveFromScroll()
    })
  },
)

watch(
  () => route.query.section,
  (raw) => {
    const s = typeof raw === "string" ? raw : undefined
    if (!s || !isSettingsSectionSlug(s)) return
    if (s === activeSlug.value) return
    scrollToSlug(s, false)
  },
)

onBeforeUnmount(() => {
  clearLockTimer()
  if (queryDebounce) clearTimeout(queryDebounce)
  detachScrollListener()
})
</script>

<template>
  <nav
    class="w-full shrink-0 lg:sticky lg:top-3 lg:w-52 lg:self-start"
    :aria-label="t('settings.navAriaLabel')"
  >
    <!-- 窄屏：下拉跳转（减少横向 Chip 拥挤） -->
    <div class="flex flex-col gap-2 lg:hidden">
      <label class="sr-only" for="settings-nav-jump">{{ t("settings.navJumpTo") }}</label>
      <Select
        id="settings-nav-jump"
        :model-value="activeSlug"
        @update:model-value="
          (v) => v != null && scrollToSlug(v as SettingsSectionSlug, true)
        "
      >
        <SelectTrigger
          class="h-10 w-full rounded-xl border-border/70 bg-card/80"
          :aria-label="t('settings.navJumpTo')"
        >
          <SelectValue :placeholder="t('settings.navJumpTo')" />
        </SelectTrigger>
        <SelectContent class="max-h-[min(24rem,70vh)] rounded-xl border-border/70">
          <SelectItem
            v-for="item in navItems"
            :key="`jump-${item.slug}`"
            class="rounded-lg"
            :value="item.slug"
          >
            {{ t(item.labelKey) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- 宽屏：纵向 -->
    <div
      class="hidden flex-col gap-1 border-border/60 text-muted-foreground/90 lg:flex lg:border-r lg:pr-4"
    >
      <button
        v-for="item in navItems"
        :key="`desktop-${item.slug}`"
        type="button"
        class="rounded-xl px-3 py-2 text-left text-sm font-medium transition-colors"
        :class="
          cn(
            activeSlug === item.slug
              ? 'bg-primary/12 text-foreground'
              : 'text-muted-foreground hover:bg-muted/40 hover:text-foreground',
          )
        "
        @click="scrollToSlug(item.slug, true)"
      >
        {{ t(item.labelKey) }}
      </button>
    </div>
  </nav>
</template>
