<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Activity, GripVertical, Loader2, Plus, X } from "lucide-vue-next"
import type { ProviderHealthDTO, ProviderHealthStatus } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { statusDotClass, statusPanelClass } from "@/lib/ui/status-tone"

const props = defineProps<{
  useWebApi: boolean
  canPickSpecifiedMetadata: boolean
  providerChainDraft: readonly string[]
  availableProvidersForChain: readonly string[]
  selectedProviderToAdd: string
  chainDragFromIndex: number | null
  metadataMovieChainSaving: boolean
  metadataMovieChainError: string
  providerPingAllBusy: boolean
  providerPingOneName: string | null
  providerHealthByName: Readonly<Record<string, ProviderHealthDTO>>
}>()

const emit = defineEmits<{
  dragStart: [event: DragEvent, index: number]
  dragOver: [event: DragEvent]
  dropProvider: [index: number]
  dragEnd: []
  pingProvider: [provider: string]
  removeProvider: [index: number]
  "update:selectedProviderToAdd": [provider: string]
  addProvider: []
  saveProviderChain: []
}>()

const { t } = useI18n()

function healthForProvider(name: string): ProviderHealthDTO | undefined {
  if (props.providerHealthByName[name]) return props.providerHealthByName[name]
  const lower = name.toLowerCase()
  for (const [key, health] of Object.entries(props.providerHealthByName)) {
    if (key.toLowerCase() === lower) return health
  }
  return undefined
}

function providerHealthTone(status: ProviderHealthStatus): "success" | "warning" | "danger" {
  if (status === "ok") return "success"
  if (status === "degraded") return "warning"
  return "danger"
}

function providerHealthStatusLabel(status: ProviderHealthStatus): string {
  if (status === "ok") return t("settings.providerHealthStatusOk")
  if (status === "degraded") return t("settings.providerHealthStatusDegraded")
  return t("settings.providerHealthStatusFail")
}

function providerHealthDotClass(status: ProviderHealthStatus): string {
  return statusDotClass(providerHealthTone(status))
}

function providerChainRowPinging(name: string): boolean {
  return props.providerPingAllBusy || props.providerPingOneName === name
}

function onProviderToAddChange(value: unknown) {
  emit("update:selectedProviderToAdd", typeof value === "string" ? value : String(value ?? ""))
}
</script>

<template>
  <div class="flex flex-col gap-3 rounded-xl border border-border/50 bg-muted/10 p-3">
    <p
      v-if="!canPickSpecifiedMetadata"
      :class="[statusPanelClass('warning'), 'rounded-xl px-3 py-2 text-sm']"
    >
      {{ t("settings.metadataMovieProviderChainNoList") }}
    </p>
    <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <p class="text-sm font-medium">{{ t("settings.metadataMovieProviderChainLabel") }}</p>
        <p class="mt-0.5 text-xs text-muted-foreground">
          {{ t("settings.metadataMovieProviderChainDragHint") }}
        </p>
      </div>
      <span class="shrink-0 text-xs text-muted-foreground">
        {{ providerChainDraft.length }} {{ t("settings.providersSelected") }}
      </span>
    </div>

    <div class="flex flex-col gap-3">
      <div
        v-for="(provider, index) in providerChainDraft"
        :key="provider + '-' + index"
        class="flex flex-wrap items-center gap-3 rounded-xl border border-border/60 bg-background/50 px-2 py-2 transition-[opacity,box-shadow] sm:px-3"
        :class="{
          'opacity-55': chainDragFromIndex === index,
          'ring-2 ring-primary/25 ring-offset-2 ring-offset-background':
            chainDragFromIndex === index,
        }"
        draggable="true"
        @dragstart="emit('dragStart', $event, index)"
        @dragover="emit('dragOver', $event)"
        @drop.prevent="emit('dropProvider', index)"
        @dragend="emit('dragEnd')"
      >
        <span
          class="inline-flex cursor-grab touch-none items-center rounded-md p-1 text-muted-foreground active:cursor-grabbing"
          :aria-label="t('settings.metadataMovieProviderChainDragHandleAria')"
        >
          <GripVertical class="size-4 shrink-0" aria-hidden="true" />
        </span>
        <div
          class="flex shrink-0 items-center gap-3"
          :title="
            useWebApi && healthForProvider(provider)
              ? providerHealthStatusLabel(healthForProvider(provider)!.status) +
                ' · ' +
                healthForProvider(provider)!.latencyMs +
                'ms'
              : undefined
          "
        >
          <Loader2
            v-if="useWebApi && providerChainRowPinging(provider)"
            class="size-3.5 shrink-0 animate-spin text-muted-foreground"
            aria-hidden="true"
          />
          <span
            v-else-if="useWebApi && healthForProvider(provider)"
            class="size-2.5 shrink-0 rounded-full"
            :class="providerHealthDotClass(healthForProvider(provider)!.status)"
            aria-hidden="true"
          />
          <span
            v-else
            class="size-2.5 shrink-0 rounded-full bg-muted-foreground/25"
            aria-hidden="true"
          />
          <span
            v-if="useWebApi && healthForProvider(provider) && !providerChainRowPinging(provider)"
            class="w-[3.25rem] shrink-0 tabular-nums text-[0.65rem] text-muted-foreground"
          >
            {{ healthForProvider(provider)!.latencyMs }}ms
          </span>
        </div>
        <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ provider }}</span>
        <Button
          v-if="useWebApi"
          type="button"
          variant="outline"
          size="icon"
          class="size-8 shrink-0 rounded-lg"
          :disabled="providerPingAllBusy || providerPingOneName === provider"
          :aria-label="t('settings.providerHealthPingOneAria', { name: provider })"
          :data-provider-chain-ping="provider"
          @click="emit('pingProvider', provider)"
        >
          <Activity
            class="size-4"
            :class="{ 'motion-safe:animate-pulse': providerPingOneName === provider }"
          />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="size-8 shrink-0 rounded-lg text-muted-foreground hover:bg-destructive/15 hover:text-destructive"
          :aria-label="t('settings.metadataMovieProviderRemoveFromChainAria', { name: provider })"
          :data-provider-chain-remove="provider"
          @click="emit('removeProvider', index)"
        >
          <X class="size-4" />
        </Button>
      </div>

      <div
        v-if="providerChainDraft.length === 0"
        class="rounded-xl border border-dashed border-border/60 bg-background/30 px-3 py-6 text-center text-sm text-muted-foreground"
      >
        {{ t("settings.metadataMovieProviderChainEmpty") }}
      </div>
    </div>

    <div v-if="availableProvidersForChain.length > 0" class="flex items-center gap-3 pt-2">
      <Select
        :model-value="selectedProviderToAdd"
        @update:model-value="onProviderToAddChange"
      >
        <SelectTrigger class="h-9 flex-1 rounded-xl text-sm">
          <SelectValue :placeholder="t('settings.selectProviderToAdd')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem
            v-for="provider in availableProvidersForChain"
            :key="provider"
            :value="provider"
          >
            {{ provider }}
          </SelectItem>
        </SelectContent>
      </Select>
      <Button
        type="button"
        variant="secondary"
        size="sm"
        class="h-9 rounded-xl px-3"
        :disabled="!selectedProviderToAdd"
        data-provider-chain-add
        @click="emit('addProvider')"
      >
        <Plus class="mr-1 size-4" />
        {{ t("common.add") }}
      </Button>
    </div>

    <div class="flex flex-col gap-3 pt-2">
      <p class="text-xs text-muted-foreground">
        {{ t("settings.metadataMovieProviderChainAutoSave") }}
      </p>
      <Button
        type="button"
        variant="default"
        class="w-fit rounded-xl font-medium"
        :disabled="metadataMovieChainSaving"
        data-provider-chain-save
        @click="emit('saveProviderChain')"
      >
        {{ metadataMovieChainSaving ? t("common.saving") : t("common.save") }}
      </Button>
    </div>

    <p
      v-if="metadataMovieChainSaving"
      class="text-xs text-muted-foreground motion-safe:animate-pulse"
    >
      {{ t("settings.metadataMovieProviderSyncing") }}
    </p>
    <p v-if="metadataMovieChainError" class="text-sm text-destructive">
      {{ metadataMovieChainError }}
    </p>
  </div>
</template>
