<script setup lang="ts">
import { onMounted } from "vue"
import { useI18n } from "vue-i18n"
import { Loader2, RefreshCw } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useDesktopUpdate } from "@/composables/use-desktop-update"

const { t } = useI18n()
const { available, info, infoError, result, loading, load, check } = useDesktopUpdate()
onMounted(load)
</script>

<template>
  <div v-if="available" class="rounded-lg border border-border/50 bg-background/55 p-3" data-desktop-update-section>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <p class="text-xs font-medium text-muted-foreground">Curated Desktop</p>
        <div class="mt-1 flex flex-wrap items-center gap-2">
          <span v-if="info" class="break-all font-mono text-sm text-foreground" data-desktop-version>{{ info.version }}</span>
          <span v-else class="text-xs text-muted-foreground">{{ t(infoError ? 'settings.desktopInfoError' : 'settings.desktopInfoLoading') }}</span>
          <Badge v-if="info?.development" variant="secondary">{{ t('settings.desktopDevelopment') }}</Badge>
        </div>
      </div>
      <Button type="button" variant="outline" class="rounded-2xl" :disabled="loading" data-desktop-update-check @click="check">
        <Loader2 v-if="loading" class="mr-2 size-4 animate-spin" aria-hidden="true" />
        <RefreshCw v-else class="mr-2 size-4" aria-hidden="true" />
        {{ t('settings.desktopCheckAction') }}
      </Button>
    </div>
    <p v-if="result" class="mt-2 text-xs leading-relaxed text-muted-foreground" role="status" data-desktop-update-status>
      {{ t(`settings.desktopUpdateStatus.${result.status}`, { version: result.latestVersion }) }}
    </p>
    <Button v-if="result?.status === 'update-available' && result.downloadUrl" as-child variant="outline" class="mt-2 rounded-2xl">
      <a :href="result.downloadUrl" target="_blank" rel="noopener noreferrer">{{ t('settings.desktopDownloadAction') }}</a>
    </Button>
  </div>
</template>
