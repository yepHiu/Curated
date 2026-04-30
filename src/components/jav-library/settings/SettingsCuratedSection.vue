<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { FolderOpen, ImageDown, Info } from "lucide-vue-next"
import type { CuratedFrameSaveMode } from "@/domain/curated-frame/types"
import type { CuratedFrameExportFormat } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from "reka-ui"
import {
  statusPanelClass,
  statusTextClass,
} from "@/lib/ui/status-tone"
import { cn } from "@/lib/utils"
import SettingsCuratedShortcutSection from "./SettingsCuratedShortcutSection.vue"

defineProps<{
  captureShortcutLabel: string
  curatedSaveMode: CuratedFrameSaveMode
  directorySupported: boolean
  curatedFrameExportFormat: CuratedFrameExportFormat
  curatedExportFormatOptions: readonly {
    value: CuratedFrameExportFormat
    label: string
  }[]
  curatedExportFormatSaving: boolean
  curatedExportDirLabel: string
  curatedExportPickBusy: boolean
  curatedExportError: string
  curatedExportFormatError: string
}>()

const emit = defineEmits<{
  "update:curatedSaveMode": [value: CuratedFrameSaveMode]
  changeExportFormat: [value: unknown]
  pickExportDirectory: []
  clearExportDirectory: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <div class="break-inside-avoid">
      <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="space-y-3 pb-2">
          <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <ImageDown class="size-[1.15rem]" />
            </span>
            {{ t("settings.curatedCardTitle") }}
          </CardTitle>
          <CardDescription class="space-y-2 text-xs leading-relaxed text-pretty text-muted-foreground">
            <p>
              {{ t("settings.curatedCardDescShort") }}
            </p>
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-normal text-muted-foreground">{{
                t("settings.curatedCardHow")
              }}</span>
              <kbd
                class="pointer-events-none inline-flex h-7 min-w-7 select-none items-center justify-center rounded-lg border border-border bg-muted px-2 font-mono text-xs font-semibold text-foreground shadow-sm"
              >
                {{ captureShortcutLabel }}
              </kbd>
              <span class="font-normal text-muted-foreground">{{
                t("settings.curatedCardOr")
              }}</span>
              <span
                class="inline-flex items-center rounded-lg border border-border/80 bg-background px-2.5 py-1 text-xs font-semibold tracking-wide text-foreground shadow-xs"
              >
                {{ t("player.curatedLabel") }}
              </span>
            </div>
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-2">
          <fieldset class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-3">
            <legend class="sr-only">{{ t("settings.savePolicy") }}</legend>
            <div class="mb-0.5 flex items-center gap-3 px-0.5">
              <span class="text-sm font-medium text-foreground">{{
                t("settings.savePolicy")
              }}</span>
              <TooltipProvider :delay-duration="280">
                <TooltipRoot>
                  <TooltipTrigger as-child>
                    <button
                      type="button"
                      class="inline-flex size-8 items-center justify-center rounded-full text-muted-foreground transition hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                    >
                      <Info class="size-4" aria-hidden="true" />
                      <span class="sr-only">{{
                        t("settings.curatedStorageTechAria")
                      }}</span>
                    </button>
                  </TooltipTrigger>
                  <TooltipPortal>
                    <TooltipContent
                      side="top"
                      :side-offset="6"
                      class="z-50 max-w-[min(22rem,calc(100vw-2rem))] rounded-xl border border-border/50 bg-popover px-3 py-2 text-xs leading-relaxed text-pretty text-popover-foreground shadow-lg"
                    >
                      {{ t("settings.curatedStorageTechTooltip") }}
                    </TooltipContent>
                  </TooltipPortal>
                </TooltipRoot>
              </TooltipProvider>
            </div>
            <label
              class="flex cursor-pointer items-start gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]"
            >
              <input
                class="mt-0.5 size-4 shrink-0 accent-primary"
                type="radio"
                name="curated-save-mode"
                value="app"
                :checked="curatedSaveMode === 'app'"
                @change="emit('update:curatedSaveMode', 'app')"
              />
              <span class="min-w-0 flex-1">
                <span class="text-sm font-medium">{{ t("settings.curatedApp") }}</span>
                <span class="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
                  {{ t("settings.curatedAppHint") }}
                </span>
              </span>
            </label>
            <label
              class="flex cursor-pointer items-start gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]"
            >
              <input
                class="mt-0.5 size-4 shrink-0 accent-primary"
                type="radio"
                name="curated-save-mode"
                value="download"
                :checked="curatedSaveMode === 'download'"
                @change="emit('update:curatedSaveMode', 'download')"
              />
              <span class="min-w-0 flex-1">
                <span class="text-sm font-medium">{{ t("settings.curatedDownload") }}</span>
                <span class="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
                  {{ t("settings.curatedDownloadHint") }}
                </span>
              </span>
            </label>
            <label
              class="flex cursor-pointer items-start gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06] has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-60"
            >
              <input
                class="mt-0.5 size-4 shrink-0 accent-primary"
                type="radio"
                name="curated-save-mode"
                value="directory"
                :checked="curatedSaveMode === 'directory'"
                :disabled="!directorySupported"
                @change="emit('update:curatedSaveMode', 'directory')"
              />
              <span class="min-w-0 flex-1">
                <span class="text-sm font-medium">{{ t("settings.curatedDir") }}</span>
                <span class="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
                  {{ t("settings.curatedDirHint") }}
                </span>
                <span
                  v-if="!directorySupported"
                  :class="cn('mt-1 block text-xs', statusTextClass('warning'))"
                >
                  {{ t("settings.curatedDirUnsupported") }}
                </span>
              </span>
            </label>
          </fieldset>

          <SettingsCuratedShortcutSection />

          <fieldset class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-3">
            <legend class="sr-only">{{ t("settings.curatedExportFormatTitle") }}</legend>
            <div
              class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4"
            >
              <div class="min-w-0 flex-1 space-y-1">
                <p class="text-sm font-medium text-foreground">
                  {{ t("settings.curatedExportFormatTitle") }}
                </p>
                <p class="text-xs leading-relaxed text-muted-foreground">
                  {{ t("settings.curatedExportFormatHint") }}
                </p>
              </div>
              <div
                class="flex w-full min-w-0 flex-wrap items-center justify-end gap-2 sm:w-auto sm:flex-shrink-0"
              >
                <span
                  v-if="curatedExportFormatSaving"
                  class="shrink-0 text-xs text-muted-foreground"
                >
                  {{ t("common.saving") }}
                </span>
                <Select
                  :model-value="curatedFrameExportFormat"
                  :disabled="curatedExportFormatSaving"
                  @update:model-value="emit('changeExportFormat', $event)"
                >
                  <SelectTrigger
                    class="h-9 w-full min-w-0 rounded-xl border-border/50 sm:w-32 sm:min-w-0 sm:shrink-0"
                    :aria-label="t('settings.curatedExportFormatLabel')"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent class="rounded-xl border-border/50">
                    <SelectItem
                      v-for="option in curatedExportFormatOptions"
                      :key="`curated-export-format-${option.value}`"
                      class="rounded-lg"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </fieldset>

          <div
            v-if="curatedSaveMode === 'directory' && directorySupported"
            class="flex flex-col gap-3 rounded-2xl border border-border/50 bg-muted/20 p-4"
          >
            <p class="text-sm font-medium">{{ t("settings.exportFolder") }}</p>
            <p class="text-sm text-muted-foreground">
              {{
                curatedExportDirLabel
                  ? t("settings.exportChosen", { name: curatedExportDirLabel })
                  : t("settings.exportNone")
              }}
            </p>
            <div class="flex flex-wrap gap-3">
              <Button
                type="button"
                variant="secondary"
                class="rounded-2xl"
                :disabled="curatedExportPickBusy"
                data-curated-pick-directory
                @click="emit('pickExportDirectory')"
              >
                <FolderOpen data-icon="inline-start" />
                {{
                  curatedExportPickBusy
                    ? t("settings.picking")
                    : curatedExportDirLabel
                      ? t("settings.changeExportFolder")
                      : t("settings.pickExportFolder")
                }}
              </Button>
              <Button
                v-if="curatedExportDirLabel"
                type="button"
                variant="outline"
                class="rounded-2xl"
                :disabled="curatedExportPickBusy"
                data-curated-clear-directory
                @click="emit('clearExportDirectory')"
              >
                {{ t("settings.clearExportFolder") }}
              </Button>
            </div>
            <div
              v-if="curatedExportDirLabel"
              class="flex flex-col gap-3 rounded-xl border border-dashed border-border/60 bg-background/40 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between"
            >
              <p class="text-xs leading-relaxed text-muted-foreground">
                {{ t("settings.curatedReauthorizeHelp") }}
              </p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="h-8 shrink-0 rounded-xl px-3 text-xs font-medium"
                :disabled="curatedExportPickBusy"
                @click="emit('pickExportDirectory')"
              >
                {{ t("settings.curatedReauthorizeExport") }}
              </Button>
            </div>
          </div>

          <p v-if="curatedExportError" class="text-sm text-destructive" role="alert">
            {{ curatedExportError }}
          </p>
          <p v-if="curatedExportFormatError" class="text-sm text-destructive" role="alert">
            {{ curatedExportFormatError }}
          </p>

          <div
            :class="statusPanelClass('info')"
            role="note"
          >
            <p :class="cn('text-sm font-medium', statusTextClass('info'))">
              {{ t("settings.curatedCorsTitle") }}
            </p>
            <p class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
              {{ t("settings.curatedCorsNote") }}
            </p>
            <a
              class="mt-2 inline-flex text-xs font-medium text-primary underline-offset-4 hover:underline"
              href="https://developer.mozilla.org/en-US/docs/Web/HTML/Cross-origin_images_and_canvas"
              target="_blank"
              rel="noopener noreferrer"
              :aria-label="t('settings.curatedCorsLearnAria')"
            >
              {{ t("settings.curatedCorsLearnMore") }}
            </a>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
