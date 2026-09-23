<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { ExternalLink, Search, LoaderCircle } from "lucide-vue-next"
import { useLibraryService } from "@/services/library-service"
import type { WishlistPlaybackResult } from "@/domain/wishlist/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

defineProps<{ itemId: string; code: string }>()
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
    <CardHeader class="gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex flex-col gap-1.5">
        <CardTitle>{{ t('wishlist.playback.title') }}</CardTitle>
        <CardDescription>{{ t('wishlist.playback.description', { code }) }}</CardDescription>
      </div>
      <Button variant="outline" class="min-h-11 rounded-full sm:min-h-9" :disabled="busy || !service.integrationsAvailable" @click="check(itemId)">
        <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" />
        <Search v-else data-icon="inline-start" />
        {{ t(busy ? 'wishlist.playback.checking' : results.length ? 'wishlist.playback.recheck' : 'wishlist.playback.check') }}
      </Button>
    </CardHeader>
    <CardContent v-if="results.length" class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4" aria-live="polite">
      <div v-for="result in results" :key="result.site" class="flex min-w-0 items-center justify-between gap-2 rounded-2xl border border-border/70 bg-muted/30 p-3">
        <div class="flex min-w-0 flex-col gap-1.5">
          <span class="truncate text-sm font-medium">{{ result.site }}</span>
          <Badge :variant="result.status === 'available' ? 'success' : 'secondary'">{{ t(`wishlist.playback.${result.status}`) }}</Badge>
        </div>
        <Button v-if="result.status === 'available' && result.url" variant="ghost" size="icon" as-child class="shrink-0 rounded-full">
          <a :href="result.url" target="_blank" rel="noopener noreferrer" :aria-label="t('wishlist.playback.open', { site: result.site })"><ExternalLink /></a>
        </Button>
      </div>
    </CardContent>
    <CardContent v-else-if="failed || !service.integrationsAvailable" class="text-sm text-muted-foreground" role="status">
      {{ t(failed ? 'wishlist.playback.failed' : 'wishlist.playback.webOnly') }}
    </CardContent>
  </Card>
</template>
