<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Languages, Power, RefreshCw } from "lucide-vue-next"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import SettingsLoggingSection from "./SettingsLoggingSection.vue"

type ThemePreference = "light" | "dark" | "system"

defineProps<{
  locale: string
  themePreference: ThemePreference
  autoDownloadUpdates: boolean
  autoDownloadUpdatesSaving: boolean
  autoDownloadUpdatesError: string
  launchAtLogin: boolean
  launchAtLoginSaving: boolean
  launchAtLoginDisabled: boolean
  launchAtLoginUnavailableHint: string
  launchAtLoginError: string
  autoSaveReady: boolean
}>()

const emit = defineEmits<{
  "update:locale": [value: string]
  changeTheme: [value: unknown]
  changeAutoDownloadUpdates: [value: boolean]
  changeLaunchAtLogin: [value: boolean]
}>()

const { t } = useI18n()

function updateLocale(value: unknown) {
  if (typeof value === "string") {
    emit("update:locale", value)
  }
}
</script>

<template>
  <div class="flex w-full flex-col gap-8">
    <div class="space-y-4">
      <div class="break-inside-avoid">
        <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
          <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <Languages class="size-4" />
            </span>
            <CardTitle class="min-w-0 text-lg tracking-tight">
              {{ t("settings.generalSubsectionLocaleAppearance") }}
            </CardTitle>
            <CardDescription
              class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
            >
              {{ t("settings.languageHint") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3 pt-0">
            <div
              class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <p class="text-sm font-semibold text-foreground">{{ t("settings.language") }}</p>
              <Select
                :model-value="locale"
                @update:model-value="updateLocale"
              >
                <SelectTrigger
                  size="sm"
                  class="h-9 w-full min-w-[11rem] shrink-0 rounded-xl border-border/50 sm:w-44"
                  :aria-label="t('settings.language')"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end" class="rounded-xl border-border/50">
                  <SelectItem value="zh-CN">{{ t("settings.langZh") }}</SelectItem>
                  <SelectItem value="en">{{ t("settings.langEn") }}</SelectItem>
                  <SelectItem value="ja">{{ t("settings.langJa") }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div
              class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="min-w-0 space-y-1">
                <p class="text-sm font-semibold text-foreground">{{ t("settings.appearance") }}</p>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                  {{ t("settings.appearanceHint") }}
                </p>
              </div>
              <Select
                :model-value="themePreference"
                @update:model-value="emit('changeTheme', $event)"
              >
                <SelectTrigger
                  size="sm"
                  class="h-9 w-full min-w-[11rem] shrink-0 rounded-xl border-border/50 sm:w-44"
                  :aria-label="t('settings.appearance')"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end" class="rounded-xl border-border/50">
                  <SelectItem value="light">{{ t("settings.themeLight") }}</SelectItem>
                  <SelectItem value="dark">{{ t("settings.themeDark") }}</SelectItem>
                  <SelectItem value="system">{{ t("settings.themeSystem") }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      </div>
      <div class="break-inside-avoid">
        <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
          <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <RefreshCw class="size-4" />
            </span>
            <CardTitle class="min-w-0 text-lg tracking-tight">
              {{ t("settings.autoDownloadUpdatesTitle") }}
            </CardTitle>
            <CardDescription
              class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
            >
              {{ t("settings.autoDownloadUpdatesDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3 pt-0">
            <div
              class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
              :aria-busy="autoDownloadUpdatesSaving"
            >
              <div class="min-w-0 space-y-1">
                <p class="text-sm font-semibold text-foreground">
                  {{ t("settings.autoDownloadUpdatesSwitch") }}
                </p>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                  {{ t("settings.autoDownloadUpdatesHint") }}
                </p>
                <p
                  v-if="autoDownloadUpdatesSaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  {{ t("settings.autoDownloadUpdatesSyncing") }}
                </p>
              </div>
              <Switch
                class="motion-safe:transition-colors motion-safe:duration-200"
                :model-value="autoDownloadUpdates"
                :aria-label="t('settings.autoDownloadUpdatesSwitch')"
                @update:model-value="emit('changeAutoDownloadUpdates', $event)"
              />
            </div>
            <p v-if="autoDownloadUpdatesError" class="text-sm text-destructive">
              {{ autoDownloadUpdatesError }}
            </p>
          </CardContent>
        </Card>
      </div>
      <div class="break-inside-avoid">
        <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
          <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <Power class="size-4" />
            </span>
            <CardTitle class="min-w-0 text-lg tracking-tight">
              {{ t("settings.launchAtLoginTitle") }}
            </CardTitle>
            <CardDescription
              class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
            >
              {{ t("settings.launchAtLoginDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3 pt-0">
            <div
              class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
              :aria-busy="launchAtLoginSaving"
            >
              <div class="min-w-0 space-y-1">
                <p class="text-sm font-semibold text-foreground">
                  {{ t("settings.launchAtLoginSwitch") }}
                </p>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                  {{ t("settings.launchAtLoginHint") }}
                </p>
                <p
                  v-if="launchAtLoginSaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  {{ t("settings.launchAtLoginSyncing") }}
                </p>
                <p
                  v-else-if="launchAtLoginUnavailableHint"
                  class="text-xs leading-relaxed text-muted-foreground sm:text-sm"
                >
                  {{ launchAtLoginUnavailableHint }}
                </p>
              </div>
              <Switch
                class="motion-safe:transition-colors motion-safe:duration-200"
                :model-value="launchAtLogin"
                :disabled="launchAtLoginDisabled"
                :aria-label="t('settings.launchAtLoginSwitch')"
                @update:model-value="emit('changeLaunchAtLogin', $event)"
              />
            </div>
            <p v-if="launchAtLoginError" class="text-sm text-destructive">
              {{ launchAtLoginError }}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
    <SettingsLoggingSection :auto-save-ready="autoSaveReady" />
  </div>
</template>
