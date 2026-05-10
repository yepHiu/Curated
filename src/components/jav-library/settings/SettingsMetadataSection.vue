<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Sparkles } from "lucide-vue-next"
import type { ProviderHealthDTO } from "@/api/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import SettingsMetadataAutomationSection from "@/components/jav-library/settings/SettingsMetadataAutomationSection.vue"
import SettingsMetadataModeSection from "@/components/jav-library/settings/SettingsMetadataModeSection.vue"
import SettingsMetadataProviderChainSection from "@/components/jav-library/settings/SettingsMetadataProviderChainSection.vue"
import SettingsMetadataProviderSelectSection from "@/components/jav-library/settings/SettingsMetadataProviderSelectSection.vue"
import SettingsMetadataTriggerScrapeSection from "@/components/jav-library/settings/SettingsMetadataTriggerScrapeSection.vue"

type MetadataMovieMode = "auto" | "specified" | "chain"

const props = defineProps<{
  useWebApi: boolean
  providerPingAllBusy: boolean
  providerPingOneName: string | null
  providerHealthPingAllSummary: string
  providerHealthPingError: string
  autoLibraryWatch: boolean
  autoLibraryWatchSaving: boolean
  autoLibraryWatchError: string
  autoActorProfileScrape: boolean
  autoActorProfileScrapeSaving: boolean
  autoActorProfileScrapeError: string
  metadataMovieModeUi: MetadataMovieMode
  metadataMovieSaving: boolean
  metadataMovieChainSaving: boolean
  canPickSpecifiedMetadata: boolean
  canUseMetadataChainMode: boolean
  metadataMovieProvider: string
  metadataMovieSelectOptions: readonly string[]
  metadataMovieError: string
  providerChainDraft: readonly string[]
  availableProvidersForChain: readonly string[]
  selectedProviderToAdd: string
  chainDragFromIndex: number | null
  metadataMovieChainError: string
  providerHealthByName: Readonly<Record<string, ProviderHealthDTO>>
  triggerScrapeCardBusy: boolean
  triggerScrapeCardSuccess: string
  triggerScrapeCardError: string
}>()

const emit = defineEmits<{
  pingAllProviders: []
  changeAutoLibraryWatch: [value: boolean]
  changeAutoActorProfileScrape: [value: boolean]
  selectAuto: []
  selectSpecified: []
  selectChain: []
  selectProvider: [value: unknown]
  pingProvider: [provider: string]
  dragStart: [event: DragEvent, index: number]
  dragOver: [event: DragEvent]
  dropProvider: [index: number]
  dragEnd: []
  removeProvider: [index: number]
  "update:selectedProviderToAdd": [provider: string]
  addProvider: []
  saveProviderChain: []
  runTriggerScrape: []
}>()

const { t } = useI18n()

const currentProviderHealth = computed(() => {
  const provider = props.metadataMovieProvider || props.metadataMovieSelectOptions[0] || ""
  return healthForProvider(provider) ?? null
})

function healthForProvider(name: string): ProviderHealthDTO | undefined {
  if (props.providerHealthByName[name]) return props.providerHealthByName[name]
  const lower = name.toLowerCase()
  for (const [key, health] of Object.entries(props.providerHealthByName)) {
    if (key.toLowerCase() === lower) return health
  }
  return undefined
}
</script>

<template>
  <div class="break-inside-avoid">
    <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="space-y-3 pb-2">
        <CardTitle
          class="flex flex-wrap items-center gap-2.5 text-lg font-semibold tracking-tight"
        >
          <span class="flex min-w-0 items-center gap-2.5">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <Sparkles class="size-4" />
            </span>
            <span class="min-w-0">{{ t("settings.metadataMovieProviderTitle") }}</span>
          </span>
        </CardTitle>
        <CardDescription
          class="text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
        >
          {{ t("settings.metadataMovieProviderDesc") }}
        </CardDescription>
      </CardHeader>
      <CardContent class="flex flex-col gap-3 pt-2">
        <SettingsMetadataAutomationSection
          :use-web-api="useWebApi"
          :provider-ping-all-busy="providerPingAllBusy"
          :provider-ping-one-name="providerPingOneName"
          :provider-health-ping-all-summary="providerHealthPingAllSummary"
          :provider-health-ping-error="providerHealthPingError"
          :auto-library-watch="autoLibraryWatch"
          :auto-library-watch-saving="autoLibraryWatchSaving"
          :auto-library-watch-error="autoLibraryWatchError"
          :auto-actor-profile-scrape="autoActorProfileScrape"
          :auto-actor-profile-scrape-saving="autoActorProfileScrapeSaving"
          :auto-actor-profile-scrape-error="autoActorProfileScrapeError"
          @ping-all-providers="emit('pingAllProviders')"
          @change-auto-library-watch="emit('changeAutoLibraryWatch', $event)"
          @change-auto-actor-profile-scrape="emit('changeAutoActorProfileScrape', $event)"
        />

        <SettingsMetadataModeSection
          :metadata-movie-mode-ui="metadataMovieModeUi"
          :metadata-movie-saving="metadataMovieSaving"
          :metadata-movie-chain-saving="metadataMovieChainSaving"
          :provider-ping-all-busy="providerPingAllBusy"
          :can-pick-specified-metadata="canPickSpecifiedMetadata"
          :can-use-metadata-chain-mode="canUseMetadataChainMode"
          @select-auto="emit('selectAuto')"
          @select-specified="emit('selectSpecified')"
          @select-chain="emit('selectChain')"
        />

        <SettingsMetadataProviderSelectSection
          v-if="metadataMovieModeUi === 'specified' && canPickSpecifiedMetadata"
          :use-web-api="useWebApi"
          :metadata-movie-provider="metadataMovieProvider"
          :metadata-movie-select-options="metadataMovieSelectOptions"
          :metadata-movie-saving="metadataMovieSaving"
          :provider-ping-all-busy="providerPingAllBusy"
          :provider-ping-one-name="providerPingOneName"
          :current-provider-health="currentProviderHealth"
          @select-provider="emit('selectProvider', $event)"
          @ping-provider="emit('pingProvider', $event)"
        />

        <SettingsMetadataProviderChainSection
          v-if="metadataMovieModeUi === 'chain'"
          :use-web-api="useWebApi"
          :can-pick-specified-metadata="canPickSpecifiedMetadata"
          :provider-chain-draft="providerChainDraft"
          :available-providers-for-chain="availableProvidersForChain"
          :selected-provider-to-add="selectedProviderToAdd"
          :chain-drag-from-index="chainDragFromIndex"
          :metadata-movie-chain-saving="metadataMovieChainSaving"
          :metadata-movie-chain-error="metadataMovieChainError"
          :provider-ping-all-busy="providerPingAllBusy"
          :provider-ping-one-name="providerPingOneName"
          :provider-health-by-name="providerHealthByName"
          @drag-start="(event, index) => emit('dragStart', event, index)"
          @drag-over="emit('dragOver', $event)"
          @drop-provider="emit('dropProvider', $event)"
          @drag-end="emit('dragEnd')"
          @ping-provider="emit('pingProvider', $event)"
          @remove-provider="emit('removeProvider', $event)"
          @update:selected-provider-to-add="emit('update:selectedProviderToAdd', $event)"
          @add-provider="emit('addProvider')"
          @save-provider-chain="emit('saveProviderChain')"
        />

        <p
          v-if="!canPickSpecifiedMetadata && metadataMovieModeUi !== 'chain'"
          class="text-sm text-muted-foreground"
        >
          {{ t("settings.metadataMovieProviderNoList") }}
        </p>
        <p
          v-if="metadataMovieSaving"
          class="text-xs text-muted-foreground motion-safe:animate-pulse"
        >
          {{ t("settings.metadataMovieProviderSyncing") }}
        </p>
        <p v-if="metadataMovieError" class="text-sm text-destructive">
          {{ metadataMovieError }}
        </p>

        <SettingsMetadataTriggerScrapeSection
          :busy="triggerScrapeCardBusy"
          :success="triggerScrapeCardSuccess"
          :error="triggerScrapeCardError"
          @run="emit('runTriggerScrape')"
        />
      </CardContent>
    </Card>
  </div>
</template>
