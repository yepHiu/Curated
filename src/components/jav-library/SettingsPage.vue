<script setup lang="ts">
import { ref } from "vue"
import { FolderPlus, RefreshCw, ScanSearch } from "lucide-vue-next"
import { libraryPaths, libraryStats, scanIntervals } from "@/lib/jav-library"
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
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

const scanInterval = ref("3600")
const hardwareDecode = ref(true)
const autoScrape = ref(true)
</script>

<template>
  <div class="mx-auto flex max-w-[56rem] flex-col gap-6 pb-2">
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <Card
        v-for="stat in libraryStats"
        :key="stat.label"
        class="rounded-3xl border-border/70 bg-card/85"
      >
        <CardHeader class="gap-1">
          <CardDescription>{{ stat.label }}</CardDescription>
          <CardTitle class="text-2xl">{{ stat.value }}</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground">{{ stat.detail }}</p>
        </CardContent>
      </Card>
    </div>

    <div class="flex flex-col gap-2 px-1">
      <div class="flex flex-col gap-1">
        <h2 class="text-3xl font-semibold tracking-tight">Library settings</h2>
        <p class="text-sm text-muted-foreground">
          Paths, scan cadence, playback preferences, and metadata workflows in a centered
          waterfall layout.
        </p>
      </div>
    </div>

    <div class="w-full columns-1 gap-6">
      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
          <CardHeader>
            <CardTitle>Storage directories</CardTitle>
            <CardDescription>
              Multi-path library support is represented here as reusable card rows.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Library paths</p>
                <p class="text-sm text-muted-foreground">
                  Add or rescan folders without leaving this screen.
                </p>
              </div>
              <Button class="rounded-2xl">
                <FolderPlus data-icon="inline-start" />
                Add path
              </Button>
            </div>

            <div class="flex flex-col gap-3">
              <div
                v-for="path in libraryPaths"
                :key="path.id"
                class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-background/50 p-4"
              >
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div class="flex flex-col gap-1">
                    <p class="font-medium">{{ path.title }}</p>
                    <p class="text-sm text-muted-foreground">{{ path.path }}</p>
                  </div>
                  <Button variant="secondary" class="rounded-2xl">
                    <RefreshCw data-icon="inline-start" />
                    Rescan
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Scan cadence</CardTitle>
            <CardDescription>
              Choose how often the scanner checks watched directories.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <Select v-model:model-value="scanInterval">
              <SelectTrigger class="rounded-2xl bg-card/70">
                <SelectValue placeholder="Select interval" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem
                    v-for="interval in scanIntervals"
                    :key="interval.value"
                    :value="interval.value"
                  >
                    {{ interval.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <div class="rounded-2xl border border-border/70 bg-background/50 p-4">
              <p class="text-sm text-muted-foreground">Current mode</p>
              <p class="mt-2 text-sm font-medium">
                Scanner cadence is configured independently from metadata scraping.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Playback preferences</CardTitle>
            <CardDescription>
              Placeholder toggles for the future player and metadata pipeline.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Hardware decode</p>
                <p class="text-sm text-muted-foreground">
                  A placeholder switch for future player preferences.
                </p>
              </div>
              <Switch v-model:checked="hardwareDecode" />
            </div>

            <div class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Auto scrape metadata</p>
                <p class="text-sm text-muted-foreground">
                  Keep newly discovered titles enriched after each scan.
                </p>
              </div>
              <Switch v-model:checked="autoScrape" />
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Manual actions</CardTitle>
            <CardDescription>
              Control points reserved for scan and metadata workflows.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Trigger metadata scrape</p>
                <p class="text-sm text-muted-foreground">
                  Fetch artwork, cast, tags, and summary metadata for indexed titles.
                </p>
              </div>
              <Button class="rounded-2xl">
                <RefreshCw data-icon="inline-start" />
                Run
              </Button>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Trigger full scan</p>
                <p class="text-sm text-muted-foreground">
                  Re-index all configured folders and refresh newly discovered files.
                </p>
              </div>
              <Button class="rounded-2xl">
                <ScanSearch data-icon="inline-start" />
                Run
              </Button>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Rebuild poster cache</p>
                <p class="text-sm text-muted-foreground">
                  Regenerate poster derivatives and clear stale preview assets.
                </p>
              </div>
              <Button variant="secondary" class="rounded-2xl">
                <RefreshCw data-icon="inline-start" />
                Run
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Configuration model</CardTitle>
            <CardDescription>
              This page mirrors the document structure while staying front-end only.
            </CardDescription>
          </CardHeader>
          <CardContent class="text-sm leading-6 text-muted-foreground">
            Future desktop integration can bind these controls to typed contracts without
            rewriting the visual structure or the surrounding page shell.
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
