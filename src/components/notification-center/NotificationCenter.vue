<script setup lang="ts">
import { ref, computed } from "vue"
import { useI18n } from "vue-i18n"
import { ArrowLeft, ArrowRight, Bell, Check, ChevronDown } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
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
const {
  unreadNotifications,
  readNotifications,
  unreadCount,
  markAllRead,
  dismissOne,
  clearAll,
} = useNotificationCenter()

const popoverOpen = ref(false)
const showHistory = ref(false)
const readExpanded = ref(false)

function onPopoverOpenChange(open: boolean) {
  popoverOpen.value = open
  if (open) {
    markAllRead()
  } else {
    showHistory.value = false
    readExpanded.value = false
  }
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

const recentReadNotifications = computed(() => readNotifications.value.slice(0, 20))

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
        <span
          v-if="unreadCount > 0"
          class="absolute top-1 right-1 h-2 w-2 rounded-full bg-primary shadow-[0_0_6px_rgba(254,98,142,0.5)]"
        />
      </Button>
    </PopoverTrigger>

    <PopoverContent
      align="end"
      :side-offset="12"
      :align-offset="-8"
      class="w-[380px] p-0 rounded-2xl shadow-lg shadow-black/15"
    >
      <!-- 标题栏 -->
      <div
        v-if="!showHistory"
        class="flex items-center justify-between px-4 py-3 border-b border-border/60"
      >
        <span class="text-sm font-semibold flex items-center gap-1.5">
          <Bell class="size-4" />
          {{ t("notificationCenter.title") }}
        </span>
        <Button
          v-if="unreadNotifications.length > 0"
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
        class="flex items-center justify-between px-4 py-3 border-b border-border/60"
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
      <ScrollArea v-if="!showHistory" class="max-h-72">
        <div v-if="unreadNotifications.length > 0" class="py-1">
          <button
            v-for="notif in unreadNotifications"
            :key="notif.id"
            type="button"
            class="flex w-full gap-2.5 px-4 py-3 text-left transition-colors hover:bg-muted/60"
            @click="dismissOne(notif.id)"
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
            <span class="mt-0.5 shrink-0 text-muted-foreground/30 text-xs leading-6">✕</span>
          </button>
        </div>

        <div
          v-else
          class="flex flex-col items-center gap-2 py-10 text-muted-foreground"
        >
          <Bell class="size-8 opacity-30" />
          <p class="text-sm">{{ t("notificationCenter.empty") }}</p>
        </div>
      </ScrollArea>

      <!-- 历史模式列表 -->
      <ScrollArea v-else class="max-h-80">
        <div v-if="recentReadNotifications.length > 0" class="py-1">
          <div
            v-for="notif in recentReadNotifications"
            :key="notif.id"
            class="flex gap-2.5 px-4 py-3 opacity-55"
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
          </div>
        </div>
        <div
          v-else
          class="flex flex-col items-center gap-2 py-10 text-muted-foreground"
        >
          <Bell class="size-8 opacity-30" />
          <p class="text-sm">{{ t("notificationCenter.noHistory") }}</p>
        </div>
      </ScrollArea>

      <!-- 已读折叠区（非历史模式） -->
      <Collapsible
        v-if="!showHistory && readNotifications.length > 0"
        v-model:open="readExpanded"
        class="border-t border-border/60"
      >
        <CollapsibleTrigger
          class="flex w-full items-center justify-between px-4 py-2.5 text-xs text-muted-foreground hover:bg-muted/40 transition-colors"
        >
          <span>{{ t("notificationCenter.readSection", { n: readNotifications.length }) }}</span>
          <ChevronDown
            class="size-3.5 transition-transform"
            :class="{ 'rotate-180': readExpanded }"
          />
        </CollapsibleTrigger>
        <CollapsibleContent class="pb-1">
          <div
            v-for="notif in recentReadNotifications"
            :key="notif.id"
            class="flex gap-2.5 px-4 py-2.5 opacity-50"
          >
            <span
              class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full"
              :class="dotClass(notif.type)"
            />
            <div class="min-w-0 flex-1">
              <p class="text-xs font-medium leading-snug">{{ notif.title }}</p>
              <p class="mt-0.5 text-[0.7rem] text-muted-foreground/60">{{ timeAgo(notif.timestamp) }}</p>
            </div>
          </div>
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
