<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Plus, RefreshCw, Server } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useBackendHealth } from "@/composables/use-backend-health"
import { useServerConnections } from "@/composables/use-server-connections"
import { statusDotClass } from "@/lib/ui/status-tone"

const { t } = useI18n()
const { available, snapshot, currentServer, loading, failed, actionFailed, opening, refresh, openManager } = useServerConnections()
const { status, probing, checkNow } = useBackendHealth()
const busy = computed(() => opening.value || snapshot.value?.connecting)
const statusLabel = computed(() => {
  if (!snapshot.value?.currentServerUrl) return t("settings.serverConnections.disconnected")
  return t(`nav.${{ online: "backendOnline", offline: "backendOffline", checking: "backendChecking", mock: "backendMock" }[status.value]}`)
})
const dotClass = computed(() => status.value === "online" ? statusDotClass("success") : status.value === "offline" ? statusDotClass("danger") : "bg-muted-foreground")
function recheck() {
  void refresh()
  checkNow()
}
</script>

<template>
  <Card v-if="available" class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-server-connections>
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
      <span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true">
        <Server class="size-[1.15rem]" />
      </span>
      <CardTitle class="min-w-0 text-lg tracking-tight">{{ t('settings.serverConnections.title') }}</CardTitle>
    </CardHeader>
    <CardContent class="flex min-w-0 flex-col gap-4 pt-0">
      <div v-if="snapshot" class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-sm font-semibold">{{ t('settings.serverConnections.current') }}</h3>
          <span class="inline-flex items-center gap-2 text-xs text-muted-foreground" role="status">
            <span class="size-2 rounded-full" :class="snapshot.currentServerUrl ? dotClass : 'bg-muted-foreground'" aria-hidden="true" />
            {{ statusLabel }}
          </span>
        </div>
        <div v-if="snapshot.currentServerUrl" class="min-w-0">
          <p class="break-words text-sm font-medium">{{ currentServer?.name || t('settings.serverConnections.unnamed') }}</p>
          <p class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ snapshot.currentServerUrl }}</p>
        </div>
      </div>
      <p v-else-if="loading" class="text-sm text-muted-foreground" role="status">{{ t('settings.serverConnections.loading') }}</p>
      <p v-if="failed" class="text-sm text-destructive" role="alert">{{ t('settings.serverConnections.loadError') }}</p>
      <div v-if="snapshot" class="min-w-0 space-y-3">
        <h3 class="text-sm font-semibold">{{ t('settings.serverConnections.saved') }}</h3>
        <p v-if="!snapshot.servers.length" class="text-sm text-muted-foreground">{{ t('settings.serverConnections.empty') }}</p>
        <ul v-else class="divide-y divide-border/60">
          <li v-for="server in snapshot.servers" :key="server.id" class="flex min-w-0 flex-wrap items-center justify-between gap-3 py-3" data-saved-server>
            <div class="min-w-0 flex-1 basis-40">
              <div class="flex flex-wrap items-center gap-2">
                <p class="break-all text-sm font-medium">{{ server.name }}</p>
                <Badge v-if="server.url === snapshot.currentServerUrl" variant="secondary">{{ t('settings.serverConnections.selected') }}</Badge>
              </div>
              <p class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ server.url }}</p>
            </div>
            <Button v-if="server.url !== snapshot.currentServerUrl" variant="outline" class="shrink-0 rounded-full" :disabled="busy || failed" :aria-label="t('settings.serverConnections.connectTo', { name: server.name })" @click="openManager(server.id)">
              {{ t('settings.serverConnections.connect') }}
            </Button>
          </li>
        </ul>
      </div>
      <p v-if="busy" class="text-sm text-muted-foreground" role="status">{{ t('settings.serverConnections.switching') }}</p>
      <p v-if="actionFailed" class="text-sm text-destructive" role="alert">{{ t('settings.serverConnections.actionError') }}</p>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <Button variant="ghost" class="rounded-full" :disabled="loading || probing || busy" @click="recheck">
          <RefreshCw class="size-4" :class="{ 'motion-safe:animate-spin': loading || probing }" aria-hidden="true" />
          {{ t('settings.serverConnections.refresh') }}
        </Button>
        <Button variant="outline" class="rounded-full" :disabled="busy" @click="openManager()">{{ t('settings.serverConnections.manage') }}</Button>
        <Button class="rounded-full" :disabled="busy" @click="openManager()">
          <Plus class="size-4" aria-hidden="true" />
          {{ t('settings.serverConnections.add') }}
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
