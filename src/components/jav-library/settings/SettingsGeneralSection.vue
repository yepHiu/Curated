<script setup lang="ts">
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Languages, Power } from "lucide-vue-next"
import {
  Card,
  CardContent,
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

type ThemePreference = "light" | "dark" | "system"

const props = defineProps<{
  locale: string
  themePreference: ThemePreference
  launchAtLogin: boolean
  launchAtLoginSaving: boolean
  launchAtLoginDisabled: boolean
  launchAtLoginError: string
}>()

const emit = defineEmits<{
  "update:locale": [value: string]
  changeTheme: [value: unknown]
  changeLaunchAtLogin: [value: boolean]
}>()

const { t } = useI18n()

const selectedLocaleLabel = computed(() => {
  if (props.locale === "en") return t("settings.langEn")
  if (props.locale === "ja") return t("settings.langJa")
  return t("settings.langZh")
})

const selectedThemeLabel = computed(() => {
  if (props.themePreference === "light") return t("settings.themeLight")
  if (props.themePreference === "dark") return t("settings.themeDark")
  return t("settings.themeSystem")
})

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
            <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
              <span>{{ t("settings.generalSubsectionLocaleAppearance") }}</span>
              <SettingsScopeBadge scope="client" />
            </CardTitle>
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
                  <SelectValue>{{ selectedLocaleLabel }}</SelectValue>
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
                  <SelectValue>{{ selectedThemeLabel }}</SelectValue>
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
              <Power class="size-4" />
            </span>
            <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
              <span>{{ t("settings.launchAtLoginTitle") }}</span>
              <SettingsScopeBadge scope="server" />
            </CardTitle>
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
                <p
                  v-if="launchAtLoginSaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  {{ t("settings.launchAtLoginSyncing") }}
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
  </div>
</template>
