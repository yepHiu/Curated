<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { Loader2, RefreshCw } from "lucide-vue-next"
import { api } from "@/api/endpoints"
import { HttpClientError } from "@/api/http-client"
import type { HomepageDailyRecommendationsDTO } from "@/api/types"
import { pushAppToast } from "@/composables/use-app-toast"
import { Button } from "@/components/ui/button"

const emit = defineEmits<{
  refreshed: [snapshot: HomepageDailyRecommendationsDTO]
}>()

const { t } = useI18n()
const refreshing = ref(false)

async function refreshHomepageRecommendations() {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const snapshot = await api.refreshHomepageDailyRecommendations()
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
  <div class="rounded-lg border border-dashed border-border/60 bg-muted/10 px-4 py-3">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="space-y-1">
        <p class="text-sm font-medium text-foreground">
          {{ t("settings.aboutHomepageRefreshTitle") }}
        </p>
        <p class="text-xs leading-5 text-muted-foreground">
          {{ t("settings.aboutHomepageRefreshBody") }}
        </p>
      </div>

      <Button
        type="button"
        variant="outline"
        :disabled="refreshing"
        data-homepage-dev-refresh
        @click="refreshHomepageRecommendations"
      >
        <Loader2 v-if="refreshing" class="mr-2 size-4 animate-spin" aria-hidden="true" />
        <RefreshCw v-else class="mr-2 size-4" aria-hidden="true" />
        {{ refreshing ? t("settings.aboutHomepageRefreshing") : t("settings.aboutHomepageRefreshAction") }}
      </Button>
    </div>
  </div>
</template>
