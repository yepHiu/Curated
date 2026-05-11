<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Info, Loader2, Sparkles } from "lucide-vue-next"
import type { HealthDTO } from "@/api/types"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { formatAboutBackendVersion } from "@/lib/about-version"
import SettingsAppUpdateSection from "@/components/jav-library/settings/SettingsAppUpdateSection.vue"
import SettingsHomepageDevTools from "@/components/jav-library/settings/SettingsHomepageDevTools.vue"

defineProps<{
  isViteDev: boolean
  useWebApi: boolean
  viteMode: string
  aboutHealth: HealthDTO | null
  aboutHealthLoading: boolean
  aboutHealthError: string
  backendVersionDisplay: string
  backendVersionStatus: "default" | "loading" | "error"
}>()

const emit = defineEmits<{
  refreshHealth: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <div class="break-inside-avoid">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <Info class="size-[1.15rem]" />
          </span>
          <CardTitle class="min-w-0 text-lg tracking-tight">
            {{ t("settings.aboutCardTitle") }}
          </CardTitle>
          <div class="col-span-full flex w-full justify-center pt-1">
            <div
              class="font-curated inline-flex w-fit max-w-full items-center gap-3 px-1 py-1.5 text-xl font-semibold tracking-wide text-primary sm:text-2xl"
              title="Curated"
            >
              <Sparkles class="size-8 shrink-0 text-primary sm:size-9" aria-hidden="true" />
              <span class="truncate">Curated</span>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-3 pt-0 text-xs leading-relaxed text-muted-foreground sm:text-sm">
          <template v-if="isViteDev">
            <div class="space-y-4">
              <div v-if="!useWebApi" class="rounded-lg border border-border/50 bg-muted/5 p-4">
                <dt class="font-semibold text-foreground">
                  {{ t("settings.aboutVersionLabel") }}
                </dt>
                <dd class="mt-1.5">
                  <span v-if="!useWebApi">{{ t("settings.aboutVersionMock") }}</span>
                  <span v-else-if="aboutHealthLoading" class="inline-flex items-center gap-3">
                    <Loader2 class="size-3.5 animate-spin text-muted-foreground" aria-hidden="true" />
                    {{ t("settings.aboutVersionLoading") }}
                  </span>
                  <span v-else-if="aboutHealthError" class="text-destructive">
                    {{ aboutHealthError }}
                  </span>
                  <span v-else-if="aboutHealth" class="font-mono text-foreground/90">
                    {{ formatAboutBackendVersion(aboutHealth) }}
                  </span>
                  <span v-else>-</span>
                </dd>
              </div>
              <dl class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                  <dt class="font-semibold text-foreground">
                    {{ t("settings.aboutCopyrightLabel") }}
                  </dt>
                  <dd class="mt-1.5">
                    {{ t("settings.aboutCopyrightValue") }}
                  </dd>
                </div>
                <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                  <dt class="font-semibold text-foreground">
                    {{ t("settings.aboutLicenseLabel") }}
                  </dt>
                  <dd class="mt-1.5 font-mono text-foreground/90">
                    {{ t("settings.aboutLicenseValue") }}
                  </dd>
                </div>
                <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                  <dt class="font-semibold text-foreground">
                    {{ t("settings.aboutRepositoryLabel") }}
                  </dt>
                  <dd class="mt-1.5 break-all font-mono text-foreground/90">
                    {{ t("settings.aboutRepositoryValue") }}
                  </dd>
                </div>
              </dl>
              <SettingsAppUpdateSection
                v-if="useWebApi"
                :backend-version-display="backendVersionDisplay"
                :backend-version-status="backendVersionStatus"
              />
              <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                <dt class="font-semibold text-foreground">
                  {{ t("settings.aboutDataModeLabel") }}
                </dt>
                <dd class="mt-1.5">
                  {{
                    useWebApi
                      ? t("settings.aboutDataModeWebApi")
                      : t("settings.aboutDataModeMock")
                  }}
                </dd>
              </div>
              <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                <dt class="font-semibold text-foreground">
                  {{ t("settings.aboutFrontendBuildLabel") }}
                </dt>
                <dd class="mt-1.5">
                  {{ t("settings.aboutFrontendBuildDev", { mode: viteMode }) }}
                </dd>
              </div>
            </div>
            <p class="text-xs leading-relaxed text-muted-foreground/90 sm:text-sm">
              {{ t("settings.aboutDevProxyHint") }}
            </p>
          </template>
          <template v-else>
            <div v-if="!useWebApi" class="rounded-lg border border-border/50 bg-muted/5 p-4">
              <p class="font-semibold text-foreground">
                {{ t("settings.aboutVersionLabel") }}
              </p>
              <p class="mt-1.5 font-mono text-sm text-foreground/90">
                <span v-if="!useWebApi">{{ t("settings.aboutVersionMock") }}</span>
                <span
                  v-else-if="aboutHealthLoading"
                  class="inline-flex items-center gap-3 font-sans text-muted-foreground"
                >
                  <Loader2 class="size-3.5 animate-spin" aria-hidden="true" />
                  {{ t("settings.aboutVersionLoading") }}
                </span>
                <span v-else-if="aboutHealthError" class="font-sans text-destructive">
                  {{ aboutHealthError }}
                </span>
                <span v-else-if="aboutHealth">{{ formatAboutBackendVersion(aboutHealth) }}</span>
                <span v-else>-</span>
              </p>
            </div>
            <dl class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                <dt class="font-semibold text-foreground">
                  {{ t("settings.aboutCopyrightLabel") }}
                </dt>
                <dd class="mt-1.5">
                  {{ t("settings.aboutCopyrightValue") }}
                </dd>
              </div>
              <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                <dt class="font-semibold text-foreground">
                  {{ t("settings.aboutLicenseLabel") }}
                </dt>
                <dd class="mt-1.5 font-mono text-foreground/90">
                  {{ t("settings.aboutLicenseValue") }}
                </dd>
              </div>
              <div class="rounded-lg border border-border/50 bg-muted/5 p-4">
                <dt class="font-semibold text-foreground">
                  {{ t("settings.aboutRepositoryLabel") }}
                </dt>
                <dd class="mt-1.5 break-all font-mono text-foreground/90">
                  {{ t("settings.aboutRepositoryValue") }}
                </dd>
              </div>
            </dl>
            <SettingsAppUpdateSection
              v-if="useWebApi"
              :backend-version-display="backendVersionDisplay"
              :backend-version-status="backendVersionStatus"
            />
          </template>
          <SettingsHomepageDevTools
            v-if="isViteDev && useWebApi"
            @refreshed="emit('refreshHealth')"
          />
        </CardContent>
      </Card>
    </div>
  </div>
</template>
