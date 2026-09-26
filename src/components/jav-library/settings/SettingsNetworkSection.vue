<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronDown, Globe, Loader2, Network } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
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
import { useSettingsScrollPreserve } from "@/composables/use-settings-scroll-preserve"
import { useLibraryService } from "@/services/library-service"

type ProxyScheme = "http" | "socks5"

type ProxyStatusMessage = {
  text: string
  className: string
} | null

const PROXY_SCHEME_OPTIONS: readonly ProxyScheme[] = ["http", "socks5"]

const props = defineProps<{
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
  proxyJavbusStatusMessage: ProxyStatusMessage
  proxyGoogleStatusMessage: ProxyStatusMessage
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
const libraryService = useLibraryService()
const { withPreservedScroll } = useSettingsScrollPreserve()
const lanSaving = ref(false)
const lanError = ref("")

const lanEnabled = computed(() => libraryService.lanEnabled.value)
const lanListening = computed(() => libraryService.lanListening.value)
const lanAccessUrls = computed(() => libraryService.lanAccessUrls.value)
const lanSwitchDisabled = computed(() => !props.useWebApi || lanSaving.value)

/** 把局域网开关的保存错误转成卡片内文案。 */
function formatLanError(error: unknown): string {
  if (error instanceof HttpClientError && error.apiError?.message) {
    return error.apiError.message
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  return t("settings.errSaveTitle")
}

/** 保存局域网访问偏好；改绑需完全退出后重新打开。 */
async function onLanEnabledChange(next: boolean) {
  if (next === lanEnabled.value) {
    return
  }
  lanError.value = ""
  try {
    await withPreservedScroll(async () => {
      lanSaving.value = true
      try {
        await libraryService.setLANEnabled(next)
      } finally {
        lanSaving.value = false
      }
    })
  } catch (error) {
    lanError.value = formatLanError(error)
  }
}

/** 代理协议下拉的本地化标签。 */
function proxySchemeLabel(value: ProxyScheme): string {
  return value === "socks5" ? t("settings.proxySchemeSocks5") : t("settings.proxySchemeHttp")
}

/** 更新代理协议草稿。 */
function updateProxyScheme(value: unknown) {
  if (value === "http" || value === "socks5") {
    emit("update:proxyScheme", value)
  }
}

/** 更新代理主机草稿。 */
function updateProxyHost(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyHost", value)
  }
}

/** 更新代理端口草稿。 */
function updateProxyPort(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyPort", value)
  }
}

/** 更新代理用户名草稿。 */
function updateProxyUsername(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyUsername", value)
  }
}

/** 更新代理密码草稿。 */
function updateProxyPassword(value: unknown) {
  if (typeof value === "string") {
    emit("update:proxyPassword", value)
  }
}
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
            <Network class="size-[1.15rem]" />
          </span>
          <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t("settings.lanAccessTitle") }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <p
            v-if="!useWebApi"
            class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-xs leading-relaxed text-muted-foreground sm:text-sm"
          >
            {{ t("settings.lanAccessMockHint") }}
          </p>
          <div
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5"
            :aria-busy="lanSaving"
          >
            <div class="min-w-0 flex-1 space-y-1">
              <p class="text-sm font-semibold text-foreground">{{ t("settings.lanAccessSwitch") }}</p>
              <p
                v-if="lanSaving"
                class="text-xs text-muted-foreground motion-safe:animate-pulse"
              >
                {{ t("common.saving") }}
              </p>
            </div>
            <Switch
              class="motion-safe:transition-colors motion-safe:duration-200"
              data-lan-access-switch
              :model-value="lanEnabled"
              :disabled="lanSwitchDisabled"
              :aria-label="t('settings.lanAccessSwitch')"
              @update:model-value="onLanEnabledChange"
            />
          </div>
          <p
            v-if="useWebApi && lanEnabled !== lanListening"
            class="text-xs leading-relaxed text-muted-foreground sm:text-sm"
          >
            {{ t("settings.lanAccessRestartHint") }}
          </p>
          <ul
            v-if="useWebApi && lanEnabled && lanAccessUrls.length > 0"
            class="space-y-1 rounded-lg border border-border/50 bg-muted/5 p-4 font-mono text-xs text-foreground sm:text-sm"
          >
            <li v-for="url in lanAccessUrls" :key="url">{{ url }}</li>
          </ul>
          <p v-if="lanError" class="text-sm text-destructive">
            {{ lanError }}
          </p>
        </CardContent>
      </Card>
    </div>
    <div class="break-inside-avoid">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <Globe class="size-[1.15rem]" />
          </span>
          <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t("settings.proxyTitle") }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <p
            v-if="!useWebApi"
            class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-xs leading-relaxed text-muted-foreground sm:text-sm"
          >
            {{ t("settings.proxyMockHint") }}
          </p>
          <div
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5"
            :aria-busy="proxySaving"
          >
            <div class="min-w-0 flex-1 space-y-1">
              <SettingsHint :text="t('settings.proxyEnabledHint')">
                <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyEnabled") }}</p>
              </SettingsHint>
            </div>
            <Switch
              class="motion-safe:transition-colors motion-safe:duration-200"
              data-proxy-enabled
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
                <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyScheme") }}</p>
                <Select
                  :model-value="proxyScheme"
                  :disabled="proxySaving"
                  @update:model-value="updateProxyScheme"
                >
                  <SelectTrigger size="sm" class="w-full rounded-xl border-border/50">
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
                <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyHost") }}</p>
                <Input
                  :model-value="proxyHost"
                  autocomplete="off"
                  class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
                  :placeholder="t('settings.proxyHostPlaceholder')"
                  :disabled="proxySaving"
                  data-proxy-host
                  @update:model-value="updateProxyHost"
                />
              </div>
              <div class="flex flex-col gap-3">
                <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyPort") }}</p>
                <Input
                  :model-value="proxyPort"
                  inputmode="numeric"
                  class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
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
                  <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyUsername") }}</p>
                  <Input
                    :model-value="proxyUsername"
                    autocomplete="off"
                    class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
                    :disabled="proxySaving"
                    data-proxy-username
                    @update:model-value="updateProxyUsername"
                  />
                </div>
                <div class="flex flex-col gap-3">
                  <p class="text-sm font-semibold text-foreground">{{ t("settings.proxyPassword") }}</p>
                  <Input
                    :model-value="proxyPassword"
                    type="password"
                    autocomplete="new-password"
                    class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
                    :disabled="proxySaving"
                    data-proxy-password
                    @update:model-value="updateProxyPassword"
                  />
                </div>
              </div>
            </div>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-3">
            <Button
              v-if="useWebApi"
              type="button"
              variant="outline"
              class="max-w-full rounded-full"
              :disabled="proxySaving || proxyOutboundPingBusy"
              :aria-busy="proxyJavbusBusy"
              :title="proxyJavbusStatusMessage?.text"
              aria-live="polite"
              aria-atomic="true"
              data-proxy-javbus
              @click="emit('testProxyJavbus')"
            >
              <Loader2
                v-if="proxyJavbusBusy"
                class="motion-safe:animate-spin"
                data-icon="inline-start"
                aria-hidden="true"
              />
              <span class="truncate" :class="!proxyJavbusBusy && proxyJavbusStatusMessage?.className">
                {{
                  proxyJavbusBusy
                    ? t("settings.proxyPingJavbusTesting")
                    : proxyJavbusStatusMessage?.text || t("settings.proxyPingJavbus")
                }}
              </span>
            </Button>
            <Button
              v-if="useWebApi"
              type="button"
              variant="outline"
              class="max-w-full rounded-full"
              :disabled="proxySaving || proxyOutboundPingBusy"
              :aria-busy="proxyGoogleBusy"
              :title="proxyGoogleStatusMessage?.text"
              aria-live="polite"
              aria-atomic="true"
              data-proxy-google
              @click="emit('testProxyGoogle')"
            >
              <Loader2
                v-if="proxyGoogleBusy"
                class="motion-safe:animate-spin"
                data-icon="inline-start"
                aria-hidden="true"
              />
              <span class="truncate" :class="!proxyGoogleBusy && proxyGoogleStatusMessage?.className">
                {{
                  proxyGoogleBusy
                    ? t("settings.proxyPingGoogleTesting")
                    : proxyGoogleStatusMessage?.text || t("settings.proxyPingGoogle")
                }}
              </span>
            </Button>
            <Button
              type="button"
              class="rounded-full"
              :disabled="proxySaving || proxyOutboundPingBusy"
              data-proxy-save
              @click="emit('saveProxy')"
            >
              {{ proxySaving ? t("common.saving") : t("settings.proxySave") }}
            </Button>
          </div>
          <p
            v-if="proxyStatusMessage"
            role="alert"
            class="text-sm transition-colors"
            :class="proxyStatusMessage.className"
          >
            {{ proxyStatusMessage.text }}
          </p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
