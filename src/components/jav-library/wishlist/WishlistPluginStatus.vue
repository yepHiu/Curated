<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useDocumentVisibility } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { statusDotClass } from "@/lib/ui/status-tone"
import { Badge } from "@/components/ui/badge"
import { useLibraryService } from "@/services/library-service"
import { useConnectedClients } from "@/composables/use-connected-clients"

const { t } = useI18n()
const service = useLibraryService()
const preferenceLoaded = ref(false)
onMounted(async () => {
  if (!service.wishlist.integrationsAvailable) return
  try {
    await service.refreshBrowserPluginEnabled()
    preferenceLoaded.value = true
  } catch {
    preferenceLoaded.value = false
  }
})
const visibility = useDocumentVisibility()
const active = computed(() => service.wishlist.integrationsAvailable && preferenceLoaded.value && service.browserPluginEnabled.value && visibility.value === "visible")
const { clients, sampledAt, loading, error } = useConnectedClients(service, active, { pollMs: 15_000 })
const online = computed(() => {
  if (!service.browserPluginEnabled.value || !service.wishlist.integrationsAvailable || error.value || (loading.value && !sampledAt.value)) return false
  const sampled = Date.parse(sampledAt.value)
  return clients.value.some((client) => {
    const age = sampled - Date.parse(client.lastSeen)
    return client.browser === "Curated Plugin" && age >= 0 && age <= 5 * 60_000
  })
})
const status = computed(() => preferenceLoaded.value && !service.browserPluginEnabled.value && service.wishlist.integrationsAvailable
  ? "disabled" : online.value ? "online" : "offline")
</script>

<template>
  <Badge
    variant="outline"
    class="h-7 gap-1.5 px-2.5"
    role="status"
    :aria-label="`${t('wishlist.pluginStatusLabel')} · ${t(`wishlist.pluginStatus.${status}`)}`"
    data-wishlist-plugin-status
  >
    <span class="size-1.5 shrink-0 rounded-full" :class="status === 'disabled' ? 'bg-muted-foreground' : statusDotClass(online ? 'success' : 'danger')" aria-hidden="true" />
    {{ t(`wishlist.pluginStatus.${status}`) }}
  </Badge>
</template>
