<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Activity } from "lucide-vue-next"
import type { ProviderHealthDTO, ProviderHealthStatus } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { StatusTone } from "@/lib/ui/status-tone"

const props = defineProps<{
  useWebApi: boolean
  metadataMovieProvider: string
  metadataMovieSelectOptions: readonly string[]
  metadataMovieSaving: boolean
  providerPingAllBusy: boolean
  providerPingOneName: string | null
  currentProviderHealth: ProviderHealthDTO | null
}>()

const emit = defineEmits<{
  selectProvider: [value: unknown]
  pingProvider: [provider: string]
}>()

const { t } = useI18n()

const currentProviderName = computed(
  () => props.metadataMovieProvider || props.metadataMovieSelectOptions[0] || "",
)

function providerHealthTone(status: ProviderHealthStatus): StatusTone {
  if (status === "ok") return "success"
  if (status === "degraded") return "warning"
  return "danger"
}

function providerHealthStatusLabel(status: ProviderHealthStatus): string {
  if (status === "ok") return t("settings.providerHealthStatusOk")
  if (status === "degraded") return t("settings.providerHealthStatusDegraded")
  return t("settings.providerHealthStatusFail")
}
</script>

<template>
  <div class="flex flex-col gap-3 rounded-xl border border-border/50 bg-muted/10 p-4">
    <p class="text-sm font-medium">{{ t("settings.metadataMovieProviderSelectLabel") }}</p>
    <div class="flex flex-wrap items-start gap-3">
      <Select
        class="min-w-0 flex-1"
        :model-value="currentProviderName"
        :disabled="metadataMovieSaving"
        @update:model-value="emit('selectProvider', $event)"
      >
        <SelectTrigger class="w-full max-w-md rounded-2xl">
          <SelectValue :placeholder="t('settings.metadataMovieProviderSelectPh')" />
        </SelectTrigger>
        <SelectContent class="rounded-xl border-border/50">
          <SelectItem
            v-for="p in metadataMovieSelectOptions"
            :key="p"
            class="rounded-lg"
            :value="p"
          >
            {{ p }}
          </SelectItem>
        </SelectContent>
      </Select>
      <Button
        v-if="useWebApi"
        type="button"
        variant="outline"
        size="icon"
        class="size-10 shrink-0 rounded-xl"
        :disabled="providerPingAllBusy || providerPingOneName != null || !currentProviderName"
        :aria-label="t('settings.providerHealthPingCurrentAria')"
        data-provider-ping-current
        @click="emit('pingProvider', currentProviderName)"
      >
        <Activity
          class="size-4"
          :class="{
            'motion-safe:animate-pulse': providerPingOneName === currentProviderName,
          }"
        />
      </Button>
    </div>
    <div
      v-if="useWebApi && currentProviderHealth"
      class="flex flex-wrap items-center gap-3"
    >
      <Badge
        :variant="providerHealthTone(currentProviderHealth.status)"
        class="text-xs font-normal"
      >
        {{ providerHealthStatusLabel(currentProviderHealth.status) }}
        &middot;
        {{ currentProviderHealth.latencyMs }}ms
      </Badge>
      <span
        v-if="currentProviderHealth.message"
        class="text-xs text-muted-foreground"
      >
        {{ currentProviderHealth.message }}
      </span>
    </div>
  </div>
</template>
