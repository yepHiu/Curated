<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { ExternalLink, Search, LoaderCircle } from "lucide-vue-next"
import { useLibraryService } from "@/services/library-service"
import type { WishlistPlaybackResult } from "@/domain/wishlist/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

defineProps<{ itemId: string }>()
const { t } = useI18n()
const service = useLibraryService().wishlist
const results = ref<WishlistPlaybackResult[]>([])
const busy = ref(false)
const failed = ref(false)

async function check(itemId: string) {
  if (busy.value) return
  busy.value = true
  failed.value = false
  try {
    results.value = (await service.playback(itemId)).results
  } catch {
    failed.value = true
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Card data-wishlist-playback class="rounded-3xl border-border/70 bg-card/85">
    <CardHeader class="flex-row items-center justify-between gap-3">
      <CardTitle>{{ t('wishlist.playback.title') }}</CardTitle>
      <Button variant="outline" size="sm" class="rounded-full" :disabled="busy || !service.integrationsAvailable" @click="check(itemId)">
        <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" />
        <Search v-else data-icon="inline-start" />
        {{ t(busy ? 'wishlist.playback.checking' : results.length ? 'wishlist.playback.recheck' : 'wishlist.playback.check') }}
      </Button>
    </CardHeader>
    <CardContent v-if="results.length" class="flex flex-wrap gap-2" aria-live="polite">
      <div v-for="result in results" :key="result.site" class="flex w-56 max-w-full min-w-0 items-center justify-between gap-2 rounded-2xl border border-border/70 bg-muted/30 p-3">
        <div class="flex min-w-0 flex-col gap-1.5">
          <span class="truncate text-sm font-medium">{{ result.site }}</span>
          <Badge :variant="result.status === 'available' ? 'success' : result.status === 'blocked' ? 'warning' : 'secondary'">{{ t(`wishlist.playback.${result.status}`) }}</Badge>
        </div>
        <Button v-if="result.url && (result.status === 'available' || result.status === 'blocked')" variant="outline" size="sm" as-child class="shrink-0 rounded-full">
          <a :href="result.url" target="_blank" rel="noopener noreferrer" :aria-label="t(result.status === 'blocked' ? 'wishlist.playback.verify' : 'wishlist.playback.open', { site: result.site })">
            {{ t(result.status === 'blocked' ? 'wishlist.playback.manual' : 'wishlist.playback.watch') }}
            <ExternalLink data-icon="inline-end" />
          </a>
        </Button>
      </div>
    </CardContent>
    <CardContent v-else-if="failed || !service.integrationsAvailable" class="text-sm text-muted-foreground" role="status">
      {{ t(failed ? 'wishlist.playback.failed' : 'wishlist.playback.webOnly') }}
    </CardContent>
  </Card>
</template>
