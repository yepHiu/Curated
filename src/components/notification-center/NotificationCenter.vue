<script setup lang="ts">
import { ref, computed } from "vue"
import { useI18n } from "vue-i18n"
import { useRouter } from "vue-router"
import { ArrowLeft, ArrowRight, Bell, Check, ChevronDown, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useNotificationCenter, type AppNotification } from "@/composables/use-notification-center"

const { t } = useI18n()
const router = useRouter()
const {
  unreadNotifications,
  readNotifications,
  unreadCount,
  markAllRead,
  dismissOne,
  clearAll,
  setCenterOpen,
} = useNotificationCenter()

const popoverOpen = ref(false)
const showHistory = ref(false)
const readExpanded = ref(false)
const notificationFilter = ref<NotificationFilter>("all")

type NotificationFilter = "all" | "attention" | "tasks" | "system"

const notificationFilters: Array<{ value: NotificationFilter; labelKey: string }> = [
  { value: "all", labelKey: "notificationCenter.filters.all" },
  { value: "attention", labelKey: "notificationCenter.filters.attention" },
  { value: "tasks", labelKey: "notificationCenter.filters.tasks" },
  { value: "system", labelKey: "notificationCenter.filters.system" },
]

function onPopoverOpenChange(open: boolean) {
  popoverOpen.value = open
  setCenterOpen(open)
  if (open) {
    return
  }
  showHistory.value = false
  readExpanded.value = false
}

function enterHistory() {
  showHistory.value = true
}

function leaveHistory() {
  showHistory.value = false
}

function handleClearAll() {
  clearAll()
  showHistory.value = false
}

function activateNotification(notif: AppNotification) {
  const route = notif.source?.route?.trim()
  if (!route) {
    return
  }
  void router.push(route)
  popoverOpen.value = false
  setCenterOpen(false)
  showHistory.value = false
  readExpanded.value = false
}

function matchesFilter(notif: AppNotification): boolean {
  if (notificationFilter.value === "all") {
    return true
  }
  if (notificationFilter.value === "attention") {
    return notif.severity === "warning" || notif.severity === "error"
  }
  if (notificationFilter.value === "tasks") {
    return notif.type === "scan" || notif.type === "scrape"
  }
  return notif.type === "update" || notif.type === "system"
}

const filteredUnreadNotifications = computed(() => unreadNotifications.value.filter(matchesFilter))
const filteredReadNotifications = computed(() => readNotifications.value.filter(matchesFilter))
const recentReadNotifications = computed(() => filteredReadNotifications.value.slice(0, 20))
const readPreviewNotifications = computed(() => filteredReadNotifications.value.slice(0, 5))
const unreadBadgeLabel = computed(() => (unreadCount.value > 99 ? "99+" : String(unreadCount.value)))
const emptyNotificationLabel = computed(() =>
  notificationFilter.value === "all"
    ? t("notificationCenter.empty")
    : t("notificationCenter.emptyFiltered"),
)

function timeAgo(ts: number): string {
  const diff = Date.now() - ts
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return t("notificationCenter.justNow")
  if (mins < 60) return t("notificationCenter.minutesAgo", { n: mins })
  const hours = Math.floor(mins / 60)
  if (hours < 24) return t("notificationCenter.hoursAgo", { n: hours })
  return t("notificationCenter.daysAgo", { n: Math.floor(hours / 24) })
}

function dotClass(type: AppNotification["type"]) {
  return `notif-dot-${type}`
}
</script>

<template>
  <Popover :open="popoverOpen" @update:open="onPopoverOpenChange">
    <PopoverTrigger as-child>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="relative rounded-2xl text-muted-foreground hover:text-foreground"
        :class="{ 'bg-muted text-foreground': popoverOpen }"
        :aria-label="t('notificationCenter.bellAria')"
      >
        <Bell class="size-5" />
        <Badge
          v-if="unreadCount > 0"
          class="absolute -top-1 -right-1 min-w-5 px-1 text-[0.65rem] leading-4 shadow-[0_0_6px_rgba(254,98,142,0.5)]"
          :aria-label="t('notificationCenter.unreadCountAria', { n: unreadBadgeLabel })"
        >
          {{ unreadBadgeLabel }}
        </Badge>
      </Button>
    </PopoverTrigger>

    <PopoverContent
      align="end"
      :side-offset="12"
      :align-offset="-8"
      class="flex max-h-[min(36rem,calc(100vh-6rem))] w-[min(380px,calc(100vw-1rem))] flex-col overflow-hidden p-0 rounded-2xl shadow-lg shadow-black/15"
    >
      <!-- 标题栏 -->
      <div
        v-if="!showHistory"
        class="flex shrink-0 items-center justify-between px-4 py-3 border-b border-border/60"
      >
        <span class="text-sm font-semibold flex items-center gap-1.5">
          <Bell class="size-4" />
          {{ t("notificationCenter.title") }}
        </span>
        <Button
          v-if="filteredUnreadNotifications.length > 0"
          type="button"
          variant="ghost"
          size="sm"
          class="h-7 rounded-lg text-xs text-primary hover:bg-primary/10"
          @click="markAllRead"
        >
          {{ t("notificationCenter.markAllRead") }}
        </Button>
      </div>

      <!-- 历史模式标题栏 -->
      <div
        v-else
        class="flex shrink-0 items-center justify-between px-4 py-3 border-b border-border/60"
      >
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-7 rounded-lg text-xs text-muted-foreground hover:text-foreground"
          @click="leaveHistory"
        >
          <ArrowLeft class="size-3.5" /> {{ t("notificationCenter.back") }}
        </Button>
        <span class="text-sm font-medium">{{ t("notificationCenter.historyTitle") }}</span>
        <Button
          v-if="readNotifications.length > 0"
          type="button"
          variant="ghost"
          size="sm"
          class="h-7 rounded-lg text-xs text-destructive hover:bg-destructive/10"
          @click="handleClearAll"
        >
          {{ t("notificationCenter.clearAll") }}
        </Button>
      </div>

      <!-- 通知列表 -->
      <div class="flex shrink-0 gap-1 overflow-x-auto border-b border-border/60 px-3 py-2">
        <Button
          v-for="filter in notificationFilters"
          :key="filter.value"
          type="button"
          :variant="notificationFilter === filter.value ? 'secondary' : 'ghost'"
          size="sm"
          class="h-7 shrink-0 rounded-lg px-2.5 text-xs"
          :data-test="`notification-filter-${filter.value}`"
          @click="notificationFilter = filter.value"
        >
          {{ t(filter.labelKey) }}
        </Button>
      </div>

      <ScrollArea v-if="!showHistory" class="h-[min(18rem,calc(100vh-14rem))] min-h-0 overflow-hidden">
        <div v-if="filteredUnreadNotifications.length > 0" class="py-1">
          <div
            v-for="notif in filteredUnreadNotifications"
            :key="notif.id"
            class="group flex w-full items-start gap-1 px-4 py-3 transition-colors hover:bg-muted/60"
          >
            <button
              type="button"
              data-test="notification-row-action"
              class="flex min-w-0 flex-1 gap-2.5 text-left"
              :class="notif.source?.route ? 'cursor-pointer' : 'cursor-default'"
              @click="activateNotification(notif)"
            >
              <span
                class="mt-1.5 h-2 w-2 shrink-0 rounded-full"
                :class="dotClass(notif.type)"
              />
              <div class="min-w-0 flex-1">
                <p class="text-sm font-medium leading-snug">{{ notif.title }}</p>
                <p class="mt-0.5 text-xs text-muted-foreground leading-snug">{{ notif.message }}</p>
                <p class="mt-1.5 text-[0.65rem] text-muted-foreground/50">{{ timeAgo(notif.timestamp) }}</p>
              </div>
            </button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              data-test="notification-row-dismiss"
              class="mt-0.5 size-7 shrink-0 rounded-lg text-muted-foreground/50 hover:text-foreground"
              :aria-label="t('notificationCenter.dismiss')"
              @click.stop="dismissOne(notif.id)"
            >
              <X class="size-3.5" />
            </Button>
          </div>
        </div>

        <div
          v-else
          class="flex flex-col items-center gap-2 py-10 text-muted-foreground"
        >
          <Bell class="size-8 opacity-30" />
          <p class="text-sm">{{ emptyNotificationLabel }}</p>
        </div>
      </ScrollArea>

      <!-- 历史模式列表 -->
      <ScrollArea v-else class="h-[min(20rem,calc(100vh-14rem))] min-h-0 overflow-hidden">
        <div v-if="recentReadNotifications.length > 0" class="py-1">
          <button
            v-for="notif in recentReadNotifications"
            :key="notif.id"
            type="button"
            class="flex w-full gap-2.5 px-4 py-3 text-left opacity-55 transition-colors hover:bg-muted/40"
            :class="notif.source?.route ? 'cursor-pointer' : 'cursor-default'"
            @click="activateNotification(notif)"
          >
            <span
              class="mt-1.5 h-2 w-2 shrink-0 rounded-full opacity-30"
              :class="dotClass(notif.type)"
            />
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium leading-snug">{{ notif.title }}</p>
              <p class="mt-0.5 text-xs text-muted-foreground leading-snug">{{ notif.message }}</p>
              <p class="mt-1.5 text-[0.65rem] text-muted-foreground/50">{{ timeAgo(notif.timestamp) }}</p>
            </div>
            <span class="mt-0.5 shrink-0 text-xs text-green-400/70 font-medium flex items-center gap-0.5">
              <Check class="size-3" />
            </span>
          </button>
        </div>
        <div
          v-else
          class="flex flex-col items-center gap-2 py-10 text-muted-foreground"
        >
          <Bell class="size-8 opacity-30" />
          <p class="text-sm">
            {{
              notificationFilter === "all"
                ? t("notificationCenter.noHistory")
                : t("notificationCenter.emptyFiltered")
            }}
          </p>
        </div>
      </ScrollArea>

      <!-- 已读折叠区（非历史模式） -->
      <Collapsible
        v-if="!showHistory && filteredReadNotifications.length > 0"
        v-model:open="readExpanded"
        class="shrink-0 border-t border-border/60"
      >
        <CollapsibleTrigger
          class="flex w-full items-center justify-between px-4 py-2.5 text-xs text-muted-foreground hover:bg-muted/40 transition-colors"
        >
          <span>{{ t("notificationCenter.readSection", { n: filteredReadNotifications.length }) }}</span>
          <ChevronDown
            class="size-3.5 transition-transform"
            :class="{ 'rotate-180': readExpanded }"
          />
        </CollapsibleTrigger>
        <CollapsibleContent class="pb-1">
          <button
            v-for="notif in readPreviewNotifications"
            :key="notif.id"
            type="button"
            class="flex w-full gap-2.5 px-4 py-2.5 text-left opacity-50 transition-colors hover:bg-muted/40"
            :class="notif.source?.route ? 'cursor-pointer' : 'cursor-default'"
            @click="activateNotification(notif)"
          >
            <span
              class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full"
              :class="dotClass(notif.type)"
            />
            <div class="min-w-0 flex-1">
              <p class="text-xs font-medium leading-snug">{{ notif.title }}</p>
              <p class="mt-0.5 text-[0.7rem] text-muted-foreground/60">{{ timeAgo(notif.timestamp) }}</p>
            </div>
          </button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="flex w-full items-center justify-center rounded-none text-xs text-primary hover:bg-primary/5"
            @click="enterHistory"
          >
            {{ t("notificationCenter.viewAll") }} <ArrowRight class="size-3.5" />
          </Button>
        </CollapsibleContent>
      </Collapsible>
    </PopoverContent>
  </Popover>
</template>

<style scoped>
.notif-dot-scan {
  background-color: #60a5fa;
}
.notif-dot-scrape {
  background-color: #4ade80;
}
.notif-dot-error {
  background-color: #f87171;
}
.notif-dot-update {
  background-color: #fbbf24;
}
.notif-dot-system {
  background-color: #9ba6ba;
}
</style>
