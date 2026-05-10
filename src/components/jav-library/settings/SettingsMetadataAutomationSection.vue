<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { RefreshCw } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"

defineProps<{
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
}>()

const emit = defineEmits<{
  pingAllProviders: []
  changeAutoLibraryWatch: [value: boolean]
  changeAutoActorProfileScrape: [value: boolean]
}>()

const { t } = useI18n()
</script>

<template>
  <div
    v-if="useWebApi"
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
  >
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <p class="text-sm font-semibold text-foreground">{{ t("settings.providerHealthTitle") }}</p>
        <p class="mt-0.5 text-xs leading-relaxed text-muted-foreground sm:text-sm">
          {{ t("settings.providerHealthHint") }}
        </p>
      </div>
      <Button
        type="button"
        variant="secondary"
        size="sm"
        class="shrink-0 rounded-xl"
        :disabled="providerPingAllBusy || providerPingOneName != null"
        data-provider-ping-all
        @click="emit('pingAllProviders')"
      >
        <RefreshCw
          class="mr-1.5 size-4"
          :class="{ 'motion-safe:animate-spin': providerPingAllBusy }"
        />
        {{
          providerPingAllBusy
            ? t("settings.providerHealthPinging")
            : t("settings.providerHealthPingAll")
        }}
      </Button>
    </div>
    <p v-if="providerHealthPingAllSummary" class="text-xs text-muted-foreground">
      {{ providerHealthPingAllSummary }}
    </p>
    <p v-if="providerHealthPingError" class="text-sm text-destructive">
      {{ providerHealthPingError }}
    </p>
  </div>
  <p
    v-else
    class="rounded-2xl border border-border/60 bg-muted/10 px-4 py-3 text-sm text-muted-foreground"
  >
    {{ t("settings.providerHealthMockHint") }}
  </p>

  <div
    class="flex items-center justify-between gap-3 rounded-2xl border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5"
    :aria-busy="autoLibraryWatchSaving"
  >
    <div class="flex min-w-0 flex-1 flex-col gap-3">
      <p class="text-sm font-semibold text-foreground">{{
        t("settings.autoScrape")
      }}</p>
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{ t("settings.autoScrapeHint") }}
      </p>
      <p
        v-if="autoLibraryWatchSaving"
        class="text-xs text-muted-foreground motion-safe:animate-pulse"
      >
        {{ t("settings.autoLibraryWatchSyncing") }}
      </p>
    </div>
    <Switch
      class="motion-safe:transition-colors motion-safe:duration-200"
      :model-value="autoLibraryWatch"
      @update:model-value="emit('changeAutoLibraryWatch', $event)"
    />
  </div>
  <p v-if="autoLibraryWatchError" class="text-sm text-destructive">
    {{ autoLibraryWatchError }}
  </p>

  <div
    class="flex items-center justify-between gap-3 rounded-2xl border border-border/50 bg-muted/[0.08] p-4"
    :aria-busy="autoActorProfileScrapeSaving"
  >
    <div class="flex min-w-0 flex-1 flex-col gap-3">
      <p class="text-sm font-semibold text-foreground">{{
        t("settings.autoActorProfileScrape")
      }}</p>
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{ t("settings.autoActorProfileScrapeHint") }}
      </p>
      <p
        v-if="autoActorProfileScrapeSaving"
        class="text-xs text-muted-foreground motion-safe:animate-pulse"
      >
        {{ t("settings.autoActorProfileScrapeSyncing") }}
      </p>
    </div>
    <Switch
      class="motion-safe:transition-colors motion-safe:duration-200"
      :model-value="autoActorProfileScrape"
      @update:model-value="emit('changeAutoActorProfileScrape', $event)"
    />
  </div>
  <p v-if="autoActorProfileScrapeError" class="text-sm text-destructive">
    {{ autoActorProfileScrapeError }}
  </p>
</template>
