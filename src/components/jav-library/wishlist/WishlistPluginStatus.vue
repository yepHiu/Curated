<script setup lang="ts">
import { computed } from "vue"
import { useDocumentVisibility } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { Puzzle } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { useLibraryService } from "@/services/library-service"
import { useConnectedClients } from "@/composables/use-connected-clients"

const { t } = useI18n()
const service = useLibraryService()
const visibility = useDocumentVisibility()
const active = computed(() => service.wishlist.integrationsAvailable && visibility.value === "visible")
const { clients, sampledAt, loading, error } = useConnectedClients(service, active, { pollMs: 15_000 })
const state = computed(() => {
  if (!service.wishlist.integrationsAvailable) return "unavailable"
  if (error.value) return "unknown"
  if (loading.value && !sampledAt.value) return "checking"
  const sampled = Date.parse(sampledAt.value)
  const connected = clients.value.some((client) => {
    const age = sampled - Date.parse(client.lastSeen)
    return client.browser === "Curated Plugin" && age >= 0 && age <= 5 * 60_000
  })
  return connected ? "connected" : "disconnected"
})
</script>

<template>
  <Badge
    :variant="state === 'connected' ? 'success' : 'outline'"
    role="status"
    :title="t('wishlist.pluginStatusHint')"
    data-wishlist-plugin-status
  >
    <Puzzle aria-hidden="true" />
    {{ t(`wishlist.pluginStatus.${state}`) }}
  </Badge>
</template>
