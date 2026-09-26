<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { DesktopInfo, DesktopUpdateResult } from "../../../../electron/desktop-contract"

const { t } = useI18n()
defineProps<{
  info: DesktopInfo | null
  infoError: boolean
  result: DesktopUpdateResult | null
}>()
</script>

<template>
  <div class="rounded-lg border border-border/50 bg-background/55 p-3" data-desktop-update-section>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <p class="text-xs font-medium text-muted-foreground">Curated Desktop</p>
        <div class="mt-1 flex flex-wrap items-center gap-2">
          <span v-if="info" class="break-all font-mono text-sm text-foreground" data-desktop-version>{{ info.version }}</span>
          <span v-else class="text-xs text-muted-foreground">{{ t(infoError ? 'settings.desktopInfoError' : 'settings.desktopInfoLoading') }}</span>
          <Badge v-if="info?.development" variant="secondary">{{ t('settings.desktopDevelopment') }}</Badge>
        </div>
      </div>
    </div>
    <p v-if="result" class="mt-2 text-xs leading-relaxed text-muted-foreground" role="status" data-desktop-update-status>
      {{ t(`settings.desktopUpdateStatus.${result.status}`, { version: result.latestVersion }) }}
    </p>
    <Button v-if="result?.status === 'update-available' && result.downloadUrl" as-child variant="outline" class="mt-2 rounded-2xl">
      <a :href="result.downloadUrl" target="_blank" rel="noopener noreferrer">{{ t('settings.desktopDownloadAction') }}</a>
    </Button>
  </div>
</template>
