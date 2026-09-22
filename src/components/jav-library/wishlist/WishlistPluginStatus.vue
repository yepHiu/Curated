<script setup lang="ts">
import { computed } from "vue"
import { useDocumentVisibility } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { statusDotClass } from "@/lib/ui/status-tone"
import { Badge } from "@/components/ui/badge"
import { useLibraryService } from "@/services/library-service"
import { useConnectedClients } from "@/composables/use-connected-clients"

const { t } = useI18n()
const service = useLibraryService()
const visibility = useDocumentVisibility()
const active = computed(() => service.wishlist.integrationsAvailable && visibility.value === "visible")
const { clients, sampledAt, loading, error } = useConnectedClients(service, active, { pollMs: 15_000 })
const online = computed(() => {
  if (!service.wishlist.integrationsAvailable || error.value || (loading.value && !sampledAt.value)) return false
  const sampled = Date.parse(sampledAt.value)
  return clients.value.some((client) => {
    const age = sampled - Date.parse(client.lastSeen)
    return client.browser === "Curated Plugin" && age >= 0 && age <= 5 * 60_000
  })
})
</script>

<template>
  <Badge
    variant="outline"
    class="h-7 gap-1.5 px-2.5"
    role="status"
    :aria-label="`${t('wishlist.pluginStatusLabel')} · ${t(online ? 'wishlist.pluginStatus.online' : 'wishlist.pluginStatus.offline')}`"
    data-wishlist-plugin-status
  >
    <span class="size-1.5 shrink-0 rounded-full" :class="statusDotClass(online ? 'success' : 'danger')" aria-hidden="true" />
    {{ t(online ? 'wishlist.pluginStatus.online' : 'wishlist.pluginStatus.offline') }}
  </Badge>
</template>
