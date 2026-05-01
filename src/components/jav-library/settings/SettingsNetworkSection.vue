<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { ChevronDown, Globe, Loader2 } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

type ProxyScheme = "http" | "socks5"

type ProxyStatusMessage = {
  text: string
  className: string
} | null

const PROXY_SCHEME_OPTIONS: readonly ProxyScheme[] = ["http", "socks5"]

defineProps<{
  useWebApi: boolean
  proxyEnabled: boolean
  proxyScheme: ProxyScheme
  proxyHost: string
  proxyPort: string
  proxyUsername: string
  proxyPassword: string
  proxyAuthExpanded: boolean
  proxySaving: boolean
  proxyOutboundPingBusy: boolean
  proxyJavbusBusy: boolean
  proxyGoogleBusy: boolean
  proxyStatusMessage: ProxyStatusMessage
}>()

const emit = defineEmits<{
  "update:proxyEnabled": [value: boolean]
  "update:proxyScheme": [value: ProxyScheme]
  "update:proxyHost": [value: string]
  "update:proxyPort": [value: string]
  "update:proxyUsername": [value: string]
  "update:proxyPassword": [value: string]
  "update:proxyAuthExpanded": [value: boolean]
  saveProxy: []
  testProxyJavbus: []
  testProxyGoogle: []
}>()

const { t } = useI18n()

function proxySchemeLabel(value: ProxyScheme): string {
  return value === "socks5" ? t("settings.proxySchemeSocks5") : t("settings.proxySchemeHttp")
}

function updateProxyScheme(value: unknown) {
  if (value === "http" || value === "socks5") {
    emit("update:proxyScheme", value)
  }
}

function updateProxyHost(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyHost", value)
  }
}

function updateProxyPort(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyPort", value)
  }
}

function updateProxyUsername(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyUsername", value)
  }
}

function updateProxyPassword(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyPassword", value)
  }
}
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
              <Globe class="size-[1.15rem]" />
            </span>
            {{ t("settings.proxyTitle") }}
          </CardTitle>
          <CardDescription
            class="text-xs leading-relaxed text-pretty text-muted-foreground"
          >
            {{ t("settings.proxyDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-2">
          <p
            v-if="!useWebApi"
            class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-sm text-muted-foreground"
          >
            {{ t("settings.proxyMockHint") }}
          </p>
          <div
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5"
            :aria-busy="proxySaving"
          >
            <div class="min-w-0 flex-1 space-y-1">
              <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyEnabled") }}</p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.proxyEnabledHint") }}
              </p>
            </div>
            <Switch
              class="motion-safe:transition-colors motion-safe:duration-200"
              :model-value="proxyEnabled"
              :disabled="proxySaving"
              @update:model-value="emit('update:proxyEnabled', $event)"
            />
          </div>
          <div
            v-if="proxyEnabled"
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          >
            <div class="grid gap-3 md:grid-cols-[11rem_minmax(0,1fr)_10rem]">
              <div class="flex flex-col gap-3">
                <p class="text-sm font-medium">{{ t("settings.proxyScheme") }}</p>
                <Select
                  :model-value="proxyScheme"
                  :disabled="proxySaving"
                  @update:model-value="updateProxyScheme"
                >
                  <SelectTrigger>
                    <SelectValue :placeholder="t('settings.proxySchemeHttp')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="option in PROXY_SCHEME_OPTIONS"
                      :key="option"
                      :value="option"
                    >
                      {{ proxySchemeLabel(option) }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="flex flex-col gap-3">
                <p class="text-sm font-medium">{{ t("settings.proxyHost") }}</p>
                <Input
                  :model-value="proxyHost"
                  autocomplete="off"
                  class="rounded-xl border-border/50"
                  :placeholder="t('settings.proxyHostPlaceholder')"
                  :disabled="proxySaving"
                  data-proxy-host
                  @update:model-value="updateProxyHost"
                />
              </div>
              <div class="flex flex-col gap-3">
                <p class="text-sm font-medium">{{ t("settings.proxyPort") }}</p>
                <Input
                  :model-value="proxyPort"
                  inputmode="numeric"
                  class="rounded-xl border-border/50"
                  :placeholder="t('settings.proxyPortPlaceholder')"
                  :disabled="proxySaving"
                  data-proxy-port
                  @update:model-value="updateProxyPort"
                />
              </div>
            </div>
            <div class="flex flex-col gap-3">
              <button
                type="button"
                class="flex h-8 min-h-8 w-full max-h-8 items-center justify-between gap-3 rounded-xl border border-border/60 bg-background/30 px-3 py-0 text-left text-sm font-medium text-foreground transition-colors hover:bg-muted/25 disabled:opacity-60"
                :disabled="proxySaving"
                :aria-expanded="proxyAuthExpanded"
                data-proxy-auth-toggle
                @click="emit('update:proxyAuthExpanded', !proxyAuthExpanded)"
              >
                <span>{{ t("settings.proxyAuthToggle") }}</span>
                <ChevronDown
                  class="size-4 shrink-0 text-muted-foreground transition-transform duration-200 motion-safe:transition-transform"
                  :class="proxyAuthExpanded ? 'rotate-180' : ''"
                  aria-hidden="true"
                />
              </button>
              <div
                v-show="proxyAuthExpanded"
                class="flex flex-col gap-3 border-t border-border/50 pt-3"
              >
                <div class="flex flex-col gap-3">
                  <p class="text-sm font-medium">{{ t("settings.proxyUsername") }}</p>
                  <Input
                    :model-value="proxyUsername"
                    autocomplete="off"
                    class="rounded-xl border-border/50"
                    :disabled="proxySaving"
                    data-proxy-username
                    @update:model-value="updateProxyUsername"
                  />
                </div>
                <div class="flex flex-col gap-3">
                  <p class="text-sm font-medium">{{ t("settings.proxyPassword") }}</p>
                  <Input
                    :model-value="proxyPassword"
                    type="password"
                    autocomplete="new-password"
                    class="rounded-xl border-border/50"
                    :disabled="proxySaving"
                    data-proxy-password
                    @update:model-value="updateProxyPassword"
                  />
                </div>
              </div>
            </div>
          </div>
          <p
            v-if="useWebApi"
            class="text-xs text-muted-foreground"
          >
            {{ t("settings.proxyPingJavbusHint") }}
          </p>
          <p
            v-if="useWebApi"
            class="text-xs text-muted-foreground"
          >
            {{ t("settings.proxyPingGoogleHint") }}
          </p>
          <div class="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              class="rounded-lg"
              :disabled="proxySaving || proxyOutboundPingBusy"
              data-proxy-save
              @click="emit('saveProxy')"
            >
              {{ proxySaving ? t("common.saving") : t("settings.proxySave") }}
            </Button>
            <Button
              v-if="useWebApi"
              type="button"
              variant="outline"
              class="rounded-lg"
              :disabled="proxySaving || proxyOutboundPingBusy"
              :aria-busy="proxyJavbusBusy"
              data-proxy-javbus
              @click="emit('testProxyJavbus')"
            >
              <Loader2
                v-if="proxyJavbusBusy"
                class="mr-2 size-4 motion-safe:animate-spin"
                aria-hidden="true"
              />
              {{
                proxyJavbusBusy
                  ? t("settings.proxyPingJavbusTesting")
                  : t("settings.proxyPingJavbus")
              }}
            </Button>
            <Button
              v-if="useWebApi"
              type="button"
              variant="outline"
              class="rounded-lg"
              :disabled="proxySaving || proxyOutboundPingBusy"
              :aria-busy="proxyGoogleBusy"
              data-proxy-google
              @click="emit('testProxyGoogle')"
            >
              <Loader2
                v-if="proxyGoogleBusy"
                class="mr-2 size-4 motion-safe:animate-spin"
                aria-hidden="true"
              />
              {{
                proxyGoogleBusy
                  ? t("settings.proxyPingGoogleTesting")
                  : t("settings.proxyPingGoogle")
              }}
            </Button>
          </div>
          <p
            class="min-h-5 text-sm transition-colors"
            :class="proxyStatusMessage?.className ?? 'text-transparent'"
          >
            {{ proxyStatusMessage?.text ?? " " }}
          </p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
