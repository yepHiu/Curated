<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronDown, Info, Loader2, ScrollText, Sparkles } from "lucide-vue-next"
import type { HealthDTO } from "@/api/types"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { formatAboutBackendVersion } from "@/lib/about-version"
import curatedLicenseUrl from "@/assets/licenses/Curated_LICENSE.txt?url&no-inline"
import harmonySansLicenseUrl from "@/assets/fonts/HarmonyOS_Sans_SC_LICENSE.txt?url&no-inline"
import notoSansLicenseUrl from "@/assets/fonts/NotoSans_LICENSE.txt?url"
import notoSansJpLicenseUrl from "@/assets/fonts/NotoSansJP_LICENSE.txt?url"
import outfitLicenseUrl from "@fontsource-variable/outfit/LICENSE?url&no-inline"
import thirdPartyNoticesUrl from "@/assets/licenses/ThirdParty_NOTICES.txt?url&no-inline"
import SettingsAppUpdateSection from "@/components/jav-library/settings/SettingsAppUpdateSection.vue"

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

const { t } = useI18n()

function repositoryHrefFromDisplay(raw: string): string {
  const s = raw.trim()
  if (!s) return "#"
  if (/^https?:\/\//i.test(s)) return s
  return `https://${s}`
}

const aboutRepositoryHref = computed(() =>
  repositoryHrefFromDisplay(t("settings.aboutRepositoryValue")),
)

const licenseItems = computed(() => [
  { name: "Curated", detail: t("settings.aboutLicenseValue"), url: curatedLicenseUrl },
])

const thirdPartyGroups = computed<Array<{
  titleKey: string
  items: Array<{ name: string; license: string; url?: string; detail?: string }>
}>>(() => [
  {
    titleKey: "settings.aboutThirdPartyFonts",
    items: [
      { name: "HarmonyOS Sans SC", license: "HarmonyOS Sans Fonts License", detail: t("settings.aboutHarmonyFontNotice"), url: harmonySansLicenseUrl },
      { name: "Noto Sans", license: "SIL Open Font License 1.1", url: notoSansLicenseUrl },
      { name: "Noto Sans JP", license: "SIL Open Font License 1.1", url: notoSansJpLicenseUrl },
      { name: "Outfit", license: "SIL Open Font License 1.1", url: outfitLicenseUrl },
    ],
  },
  {
    titleKey: "settings.aboutThirdPartyFrontend",
    items: [
      { name: "Vue", license: "MIT" },
      { name: "Vue Router", license: "MIT" },
      { name: "Vue I18n", license: "MIT" },
      { name: "Reka UI", license: "MIT" },
      { name: "hls.js", license: "Apache-2.0" },
      { name: "DOMPurify", license: "Apache-2.0 OR MPL-2.0" },
      { name: "Lucide", license: "ISC" },
      { name: "pinyin-pro", license: "MIT" },
    ],
  },
  {
    titleKey: "settings.aboutThirdPartyBackend",
    items: [
      { name: "GORM", license: "MIT" },
      { name: "go-sqlite", license: "BSD-3-Clause" },
      { name: "fsnotify", license: "BSD-3-Clause" },
      { name: "MetaTube SDK", license: "Apache-2.0" },
      { name: "Zap", license: "MIT" },
    ],
  },
  {
    titleKey: "settings.aboutThirdPartyDesktop",
    items: [
      { name: "Electron", license: "MIT" },
      { name: "FFmpeg", license: "GPL-3.0-or-later" },
    ],
  },
])

const thirdPartyCount = computed(() => thirdPartyGroups.value.reduce((count, group) => count + group.items.length, 0))
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
              <div class="rounded-lg bg-muted/5 px-4 pt-4 pb-2">
                <dl class="space-y-4">
                  <div
                    class="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-between sm:gap-6"
                  >
                    <dt class="shrink-0 font-semibold text-foreground">
                      {{ t("settings.aboutCopyrightLabel") }}
                    </dt>
                    <dd class="min-w-0 break-words text-end text-foreground/90">
                      {{ t("settings.aboutCopyrightValue") }}
                    </dd>
                  </div>
                  <div
                    class="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-between sm:gap-6"
                  >
                    <dt class="shrink-0 font-semibold text-foreground">
                      {{ t("settings.aboutRepositoryLabel") }}
                    </dt>
                    <dd class="min-w-0 break-all text-end font-mono">
                      <a
                        :href="aboutRepositoryHref"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="rounded-sm text-primary underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        {{ t("settings.aboutRepositoryValue") }}
                      </a>
                    </dd>
                  </div>
                </dl>
              </div>
              <SettingsAppUpdateSection
                v-if="useWebApi"
                :backend-version-display="backendVersionDisplay"
                :backend-build-stamp="aboutHealth?.buildStamp"
                :backend-channel="aboutHealth?.channel"
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
            <div class="rounded-lg bg-muted/5 px-4 pt-4 pb-2">
              <dl class="space-y-4">
                <div
                  class="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-between sm:gap-6"
                >
                  <dt class="shrink-0 font-semibold text-foreground">
                    {{ t("settings.aboutCopyrightLabel") }}
                  </dt>
                  <dd class="min-w-0 break-words text-end text-foreground/90">
                    {{ t("settings.aboutCopyrightValue") }}
                  </dd>
                </div>
                <div
                  class="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-between sm:gap-6"
                >
                  <dt class="shrink-0 font-semibold text-foreground">
                    {{ t("settings.aboutRepositoryLabel") }}
                  </dt>
                  <dd class="min-w-0 break-all text-end font-mono">
                    <a
                      :href="aboutRepositoryHref"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="rounded-sm text-primary underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      {{ t("settings.aboutRepositoryValue") }}
                    </a>
                  </dd>
                </div>
              </dl>
            </div>
            <SettingsAppUpdateSection
              v-if="useWebApi"
              :backend-version-display="backendVersionDisplay"
              :backend-build-stamp="aboutHealth?.buildStamp"
              :backend-channel="aboutHealth?.channel"
              :backend-version-status="backendVersionStatus"
            />
          </template>
        </CardContent>
      </Card>
    </div>
    <slot name="updates" />
    <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <ScrollText class="size-[1.15rem]" />
        </span>
        <CardTitle class="min-w-0 text-lg tracking-tight">
          {{ t("settings.aboutUsageLicensesTitle") }}
        </CardTitle>
      </CardHeader>
      <CardContent class="pt-0">
        <div class="divide-y divide-border/50 rounded-lg border border-border/50 bg-muted/5">
          <div
            v-for="license in licenseItems"
            :key="license.name"
            class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 p-4"
          >
            <div class="min-w-0">
              <p class="text-sm font-medium text-foreground">{{ license.name }}</p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ license.detail }}
              </p>
            </div>
            <a
              :href="license.url"
              :aria-label="t('settings.aboutViewLicenseFor', { name: license.name })"
              target="_blank"
              rel="noopener noreferrer"
              class="shrink-0 rounded-sm text-sm text-primary underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
            >
              {{ t("settings.aboutViewLicense") }}
            </a>
          </div>
        </div>
        <details class="group mt-3 rounded-lg border border-border/50 bg-muted/5">
          <summary class="flex cursor-pointer list-none items-center justify-between gap-3 rounded-lg p-4 text-sm font-semibold text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden">
            {{ t("settings.aboutThirdPartyTitle", { count: thirdPartyCount }) }}
            <ChevronDown class="size-4 shrink-0 transition-transform group-open:rotate-180" aria-hidden="true" />
          </summary>
          <div class="border-t border-border/50 px-4 pb-4">
            <div v-for="group in thirdPartyGroups" :key="group.titleKey" class="pt-4">
              <h4 class="text-sm font-semibold text-foreground">{{ t(group.titleKey) }}</h4>
              <ul class="mt-2 grid gap-x-5 gap-y-1 sm:grid-cols-2">
                <li
                  v-for="item in group.items"
                  :key="item.name"
                  class="flex min-w-0 flex-wrap items-baseline justify-between gap-x-3 text-xs leading-relaxed sm:text-sm"
                >
                  <span class="shrink-0 text-foreground">{{ item.name }}</span>
                  <a
                    v-if="item.url"
                    :href="item.url"
                    :aria-label="t('settings.aboutViewLicenseFor', { name: item.name })"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="rounded-sm text-primary underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
                  >{{ item.license }}</a>
                  <span v-else class="shrink-0 text-muted-foreground">{{ item.license }}</span>
                  <span v-if="item.detail" class="w-full text-muted-foreground">{{ item.detail }}</span>
                </li>
              </ul>
            </div>
            <p class="mt-4 text-xs text-muted-foreground">
              {{ t("settings.aboutFfmpegBundleNote") }}
            </p>
            <a
              :href="thirdPartyNoticesUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-2 inline-block rounded-sm text-sm text-primary underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
            >
              {{ t("settings.aboutThirdPartyNoticesLink") }}
            </a>
          </div>
        </details>
      </CardContent>
    </Card>
  </div>
</template>
