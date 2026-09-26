<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { CalendarDays, Loader2, RefreshCw } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { HomepageDailyRecommendationsDTO } from "@/api/types"
import { pushAppToast } from "@/composables/use-app-toast"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useLibraryService } from "@/services/library-service"

const emit = defineEmits<{
  refreshed: [snapshot: HomepageDailyRecommendationsDTO]
}>()

const { t } = useI18n()
const libraryService = useLibraryService()
const refreshing = ref(false)

async function refreshHomepageRecommendations() {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const snapshot = await libraryService.refreshHomepageDailyRecommendations()
    emit("refreshed", snapshot)
    pushAppToast(t("settings.aboutHomepageRefreshSuccess", {
      date: snapshot.dateUtc,
    }), {
      variant: "success",
    })
  } catch (err) {
    if (err instanceof HttpClientError && err.apiError?.message) {
      pushAppToast(err.apiError.message, { variant: "destructive" })
    } else if (err instanceof Error && err.message) {
      pushAppToast(err.message, { variant: "destructive" })
    } else {
      pushAppToast(t("settings.aboutHomepageRefreshFailed"), { variant: "destructive" })
    }
  } finally {
    refreshing.value = false
  }
}
</script>

<template>
  <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
        aria-hidden="true"
      >
        <CalendarDays class="size-[1.15rem]" />
      </span>
      <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
        <SettingsHint :text="t('settings.aboutHomepageRefreshBody')">
          <span>{{ t("settings.aboutHomepageRefreshTitle") }}</span>
        </SettingsHint>
        <SettingsScopeBadge scope="server" />
      </CardTitle>
    </CardHeader>
    <CardContent class="pt-0">
      <div class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between">

        <Button
          type="button"
          variant="outline"
          size="sm"
          class="h-auto min-h-11 w-full max-w-full shrink-0 whitespace-normal sm:min-h-8 sm:w-auto"
          :disabled="refreshing"
          :aria-busy="refreshing"
          data-settings-comfortable-control
          data-homepage-dev-refresh
          @click="refreshHomepageRecommendations"
        >
          <Loader2 v-if="refreshing" class="size-4 motion-safe:animate-spin" aria-hidden="true" />
          <RefreshCw v-else class="size-4" aria-hidden="true" />
          {{ refreshing ? t("settings.aboutHomepageRefreshing") : t("settings.aboutHomepageRefreshAction") }}
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
