<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import {
  Clock3,
  Laptop,
  Monitor,
  RefreshCw,
  Smartphone,
  Tablet,
  Terminal,
  Wifi,
} from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type { ConnectedClientDTO } from "@/api/types"

const props = withDefaults(
  defineProps<{
    clients: readonly ConnectedClientDTO[]
    total?: number
    localCount?: number
    remoteCount?: number
    loading?: boolean
    error?: string
    sampledAt?: string
  }>(),
  {
    total: undefined,
    localCount: undefined,
    remoteCount: undefined,
    loading: false,
    error: "",
    sampledAt: "",
  },
)

const emit = defineEmits<{
  refresh: []
}>()

const { t } = useI18n()

const totalDisplay = computed(() => props.total ?? props.clients.length)
const localDisplay = computed(
  () => props.localCount ?? props.clients.filter((client) => client.accessKind === "local").length,
)
const remoteDisplay = computed(
  () => props.remoteCount ?? props.clients.filter((client) => client.accessKind === "remote").length,
)

const metricItems = computed(() => [
  {
    key: "total",
    label: t("settings.connectedClientsTotal"),
    value: totalDisplay.value,
  },
  {
    key: "remote",
    label: t("settings.connectedClientsRemote"),
    value: remoteDisplay.value,
  },
  {
    key: "local",
    label: t("settings.connectedClientsLocal"),
    value: localDisplay.value,
  },
  {
    key: "sampled",
    label: t("settings.connectedClientsLastRefresh"),
    value: formatTimestamp(props.sampledAt),
  },
])

function shouldHideBrowserVersion(browser: string): boolean {
  return browser.trim().toLowerCase() === "chrome"
}

function deviceIcon(client: ConnectedClientDTO) {
  switch (client.deviceType) {
    case "mobile":
      return Smartphone
    case "tablet":
      return Tablet
    case "tool":
      return Terminal
    case "laptop":
      return Laptop
    default:
      return Monitor
  }
}

function clientTitle(client: ConnectedClientDTO): string {
  if (client.deviceType === "tool") {
    return client.browser || t("settings.connectedClientsTool")
  }
  const browserVersion =
    client.browserVersion && !shouldHideBrowserVersion(client.browser) ? client.browserVersion : ""
  const browser = [client.browser, browserVersion].filter(Boolean).join(" ")
  const osVersion = displayOSVersion(client.os, client.osVersion)
  const os = [client.os, osVersion].filter(Boolean).join(" ")
  if (browser && os) {
    return `${browser} · ${os}`
  }
  return browser || os || t("settings.connectedClientsUnknownDevice")
}

function displayOSVersion(os: string, version?: string): string {
  const normalizedOS = os.trim().toLowerCase()
  const normalizedVersion = String(version ?? "").trim()
  if (normalizedOS === "windows" && normalizedVersion === "10.0") {
    return ""
  }
  return normalizedVersion
}

function clientAddress(client: ConnectedClientDTO): string {
  const host = client.hostname?.trim()
  return host ? `${client.ip} · ${host}` : client.ip
}

function requestCountText(count: number): string {
  return t("settings.connectedClientsRequests", { count })
}

function formatTimestamp(value?: string): string {
  if (!value) {
    return "—"
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date)
}
</script>

<template>
  <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-2.5 gap-y-1 pb-0">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
        aria-hidden="true"
      >
        <Wifi class="size-[1.15rem]" />
      </span>
      <CardTitle class="min-w-0 text-lg tracking-tight">
        {{ t("settings.connectedClientsTitle") }}
      </CardTitle>
      <Button
        variant="outline"
        size="sm"
        class="row-span-2 self-start rounded-xl"
        :disabled="loading"
        @click="emit('refresh')"
      >
        <RefreshCw :class="['size-4', loading ? 'motion-safe:animate-spin' : '']" />
        {{ t("settings.connectedClientsRefresh") }}
      </Button>
      <CardDescription
        class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
      >
        {{ t("settings.connectedClientsDesc") }}
      </CardDescription>
    </CardHeader>

    <CardContent class="flex flex-col gap-3 pt-0">
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="item in metricItems"
          :key="item.key"
          class="rounded-lg border border-border/50 bg-muted/5 px-3.5 py-3"
        >
          <p class="text-xs font-medium text-muted-foreground">{{ item.label }}</p>
          <p class="mt-1 text-xl font-semibold tabular-nums">{{ item.value }}</p>
        </div>
      </div>

      <div
        v-if="error"
        class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        role="alert"
      >
        {{ error }}
      </div>

      <div v-if="loading && clients.length === 0" class="flex flex-col gap-2" aria-live="polite">
        <Skeleton v-for="i in 3" :key="i" class="h-16 rounded-lg" />
      </div>

      <div v-else-if="clients.length === 0" class="rounded-lg border border-border/50 bg-muted/5 px-4 py-3 text-sm text-muted-foreground">
        {{ t("settings.connectedClientsEmpty") }}
      </div>

      <div v-else class="flex flex-col gap-2">
        <div
          v-for="client in clients"
          :key="client.key"
          class="grid gap-3 rounded-lg border border-border/50 bg-muted/5 px-3.5 py-3 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center"
        >
          <span
            class="flex size-9 items-center justify-center rounded-lg border border-border/60 bg-background text-muted-foreground"
            aria-hidden="true"
          >
            <component :is="deviceIcon(client)" class="size-4" />
          </span>

          <div class="min-w-0">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <p class="min-w-0 truncate text-sm font-medium">
                {{ clientTitle(client) }}
              </p>
              <Badge :variant="client.accessKind === 'remote' ? 'info' : 'secondary'">
                {{ client.accessKind === "remote"
                  ? t("settings.connectedClientsRemoteBadge")
                  : t("settings.connectedClientsLocalBadge") }}
              </Badge>
              <Badge v-if="client.isLocalMachine" variant="outline">
                {{ t("settings.connectedClientsThisDevice") }}
              </Badge>
            </div>
            <p class="mt-1 truncate text-xs text-muted-foreground">
              {{ clientAddress(client) }}
            </p>
          </div>

          <div class="flex flex-col gap-1 text-xs text-muted-foreground sm:items-end">
            <span class="inline-flex items-center gap-1">
              <Clock3 class="size-3.5" />
              {{ t("settings.connectedClientsLastSeen", { time: formatTimestamp(client.lastSeen) }) }}
            </span>
            <span>{{ requestCountText(client.requestCount) }}</span>
          </div>
        </div>
      </div>

      <p class="text-xs leading-relaxed text-muted-foreground">
        {{ t("settings.connectedClientsPrivacy") }}
      </p>
    </CardContent>
  </Card>
</template>
