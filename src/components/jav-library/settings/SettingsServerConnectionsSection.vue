<script setup lang="ts">
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { RefreshCw, Server } from "lucide-vue-next"
import SettingsServerAddDialog from "./SettingsServerAddDialog.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useBackendHealth } from "@/composables/use-backend-health"
import { useServerConnections } from "@/composables/use-server-connections"

const { t } = useI18n()
const { available, snapshot, currentServer, loading, failed, actionFailed, opening, refresh, openManager } = useServerConnections()
const { status, probing, checkNow } = useBackendHealth()
const busy = computed(() => opening.value || snapshot.value?.connecting)
const statusLabel = computed(() => {
  if (!snapshot.value?.currentServerUrl) return t("settings.serverConnections.disconnected")
  if (status.value === "online") return t("settings.serverConnections.online")
  return t(`nav.${{ offline: "backendOffline", checking: "backendChecking", mock: "backendMock" }[status.value]}`)
})
const statusVariant = computed(() => !snapshot.value?.currentServerUrl ? "secondary" : status.value === "online" ? "success" : status.value === "offline" ? "danger" : "secondary")
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
      <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
        <span>{{ t('settings.serverConnections.title') }}</span>
        <SettingsScopeBadge scope="desktop" />
      </CardTitle>
    </CardHeader>
    <CardContent class="flex min-w-0 flex-col gap-3 pt-0">
      <div v-if="snapshot" class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5 rounded-lg border border-border/50 bg-muted/5 p-4">
        <h3 class="shrink-0 text-xs font-medium text-muted-foreground">{{ t('settings.serverConnections.current') }}</h3>
        <template v-if="snapshot.currentServerUrl">
          <p class="min-w-0 max-w-full truncate text-sm font-medium" :title="currentServer?.name || t('settings.serverConnections.unnamed')">{{ currentServer?.name || t('settings.serverConnections.unnamed') }}</p>
          <p class="min-w-0 flex-1 basis-40 truncate text-xs text-muted-foreground" :title="snapshot.currentServerUrl">{{ snapshot.currentServerUrl }}</p>
        </template>
        <Badge :variant="statusVariant" class="ml-auto" role="status">
          {{ statusLabel }}
        </Badge>
      </div>
      <p v-else-if="loading" class="text-sm text-muted-foreground" role="status">{{ t('settings.serverConnections.loading') }}</p>
      <p v-if="failed" class="text-sm text-destructive" role="alert">{{ t('settings.serverConnections.loadError') }}</p>
      <section v-if="snapshot" class="@container flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" :aria-label="t('settings.serverConnections.saved')">
        <h3 class="text-sm font-semibold text-foreground">{{ t('settings.serverConnections.saved') }}</h3>
        <p v-if="!snapshot.servers.length" class="text-sm text-muted-foreground">{{ t('settings.serverConnections.empty') }}</p>
        <ul v-else class="min-w-0 overflow-hidden rounded-lg border border-border/50 bg-background/30 divide-y divide-border/50">
          <li
            v-for="server in snapshot.servers"
            :key="server.id"
            class="grid min-w-0 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5"
            data-saved-server
          >
            <Server class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <div class="grid min-w-0 gap-1 @lg:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)] @lg:items-center @lg:gap-3">
              <p class="truncate text-sm font-medium text-foreground" :title="server.name">{{ server.name }}</p>
              <p class="truncate text-xs text-muted-foreground" :title="server.url">{{ server.url }}</p>
            </div>
            <div class="flex w-24 shrink-0 justify-end">
              <Badge v-if="server.url === snapshot.currentServerUrl" variant="secondary" class="whitespace-nowrap">{{ t('settings.serverConnections.selected') }}</Badge>
              <Button
                v-else
                type="button"
                size="sm"
                variant="outline"
                class="h-auto min-h-11 shrink-0 rounded-full sm:min-h-8"
                data-settings-comfortable-control
                :disabled="busy || failed"
                :aria-label="t('settings.serverConnections.connectTo', { name: server.name })"
                @click="openManager(server.id)"
              >
                {{ t('settings.serverConnections.connect') }}
              </Button>
            </div>
          </li>
        </ul>
      </section>
      <p v-if="busy" class="text-sm text-muted-foreground" role="status">{{ t('settings.serverConnections.switching') }}</p>
      <p v-if="actionFailed" class="text-sm text-destructive" role="alert">{{ t('settings.serverConnections.actionError') }}</p>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <Button variant="ghost" class="rounded-full" :disabled="loading || probing || busy" @click="recheck">
          <RefreshCw class="size-4" :class="{ 'motion-safe:animate-spin': loading || probing }" aria-hidden="true" />
          {{ t('settings.serverConnections.refresh') }}
        </Button>
        <Button variant="outline" class="rounded-full" :disabled="busy" @click="openManager()">{{ t('settings.serverConnections.manage') }}</Button>
        <SettingsServerAddDialog :disabled="Boolean(busy)" @saved="refresh" />
      </div>
    </CardContent>
  </Card>
</template>
