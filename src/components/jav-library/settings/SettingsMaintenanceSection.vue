<script setup lang="ts">
import { defineAsyncComponent } from "vue"
import { useI18n } from "vue-i18n"
import { ScanSearch, Wrench } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import SettingsBackupSection from "./SettingsBackupSection.vue"

const SettingsLibraryHealthSection = defineAsyncComponent(
  () => import("./SettingsLibraryHealthSection.vue"),
)

defineProps<{
  fullScanBusy: boolean
  backupSupported: boolean
  healthSupported: boolean
}>()

const emit = defineEmits<{
  runFullScan: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="break-inside-avoid">
    <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <Wrench class="size-[1.15rem]" />
        </span>
        <CardTitle class="min-w-0 text-lg tracking-tight">
          {{ t("settings.navMaintenance") }}
        </CardTitle>
      </CardHeader>

      <CardContent class="flex flex-col gap-3 pt-0">
        <SettingsLibraryHealthSection :supported="healthSupported" />

        <SettingsBackupSection :supported="backupSupported" />

        <section
          aria-labelledby="settings-manual-maintenance-title"
          class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
          data-settings-maintenance-block="manual"
        >
          <div class="flex min-w-0 flex-col gap-2">
            <h3
              id="settings-manual-maintenance-title"
              class="text-sm font-semibold text-foreground"
            >
              {{ t("settings.triggerFullScan") }}
            </h3>
            <p class="text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.triggerFullScanHint") }}
            </p>
          </div>

          <Button
            type="button"
            variant="outline"
            size="sm"
            class="h-auto min-h-11 w-full shrink-0 sm:h-8 sm:min-h-8 sm:w-auto"
            :disabled="fullScanBusy"
            data-settings-comfortable-control
            data-settings-full-scan
            @click="emit('runFullScan')"
          >
            <ScanSearch
              data-icon="inline-start"
              :class="{ 'motion-safe:animate-pulse': fullScanBusy }"
            />
            {{ t("common.run") }}
          </Button>
        </section>
      </CardContent>
    </Card>
  </div>
</template>
