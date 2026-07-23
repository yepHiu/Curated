<script setup lang="ts">
import { defineAsyncComponent } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen, ScanSearch, Wrench } from "lucide-vue-next"
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
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          data-settings-maintenance-block="manual"
        >
          <div class="flex min-w-0 flex-col gap-2">
            <div class="flex items-center gap-2">
              <ScanSearch class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
              <h3
                id="settings-manual-maintenance-title"
                class="text-sm font-semibold text-foreground"
              >
                {{ t("settings.manualCardTitle") }}
              </h3>
            </div>
            <p class="text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.manualCardDesc") }}
            </p>
          </div>

          <div
            class="flex flex-col gap-3 rounded-lg border border-border/40 bg-background/30 p-3 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="flex min-w-0 flex-col gap-2">
              <p class="text-sm font-medium text-foreground">
                {{ t("settings.triggerFullScan") }}
              </p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.triggerFullScanHint") }}
              </p>
            </div>
            <Button
              type="button"
              class="h-auto min-h-11 w-full shrink-0 rounded-2xl px-5 font-medium sm:w-auto"
              :disabled="fullScanBusy"
              data-settings-comfortable-control
              data-settings-full-scan
              @click="emit('runFullScan')"
            >
              <ScanSearch
                data-icon="inline-start"
                :class="fullScanBusy ? 'animate-pulse' : ''"
              />
              {{ t("common.run") }}
            </Button>
          </div>
        </section>

        <section
          aria-labelledby="settings-config-maintenance-title"
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          data-settings-maintenance-block="config"
        >
          <div class="flex min-w-0 flex-col gap-2">
            <div class="flex items-center gap-2">
              <BookOpen class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
              <h3
                id="settings-config-maintenance-title"
                class="text-sm font-semibold text-foreground"
              >
                {{ t("settings.configCardTitle") }}
              </h3>
            </div>
            <p class="text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.configCardDesc") }}
            </p>
          </div>
          <p
            class="rounded-lg border border-border/40 bg-background/30 p-3 text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm sm:leading-6"
          >
            {{ t("settings.configCardBody") }}
          </p>
        </section>
      </CardContent>
    </Card>
  </div>
</template>
