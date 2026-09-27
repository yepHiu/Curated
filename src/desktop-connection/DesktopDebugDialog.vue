<script setup lang="ts">
import { nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { LoaderCircle } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { DesktopDebugAPI, DesktopDebugInfo, DesktopDebugProbe } from "./debug-contract"

const props = defineProps<{ open: boolean; api: DesktopDebugAPI; initialAddress: string }>()
const emit = defineEmits<{ "update:open": [open: boolean] }>()
const { t } = useI18n()
const tab = ref("status")
const info = ref<DesktopDebugInfo>()
const loading = ref(false)
const loadError = ref("")
const target = ref("")
const probing = ref(false)
const probeResult = ref<DesktopDebugProbe>()
const probeError = ref("")
const actionError = ref("")
let loadId = 0

async function refresh() {
  const id = ++loadId
  loading.value = true
  loadError.value = ""
  try {
    const result = await props.api.readDebugInfo()
    if (id === loadId) info.value = result
  } catch (reason) { if (id === loadId) loadError.value = String(reason) }
  finally { if (id === loadId) loading.value = false }
}

watch(() => props.open, open => {
  if (open) {
    tab.value = "status"
    target.value = props.initialAddress
    probeResult.value = undefined
    probeError.value = ""
    actionError.value = ""
    void refresh()
  } else ++loadId
}, { immediate: true })

async function probe() {
  if (probing.value || !target.value.trim()) return
  probing.value = true
  probeResult.value = undefined
  probeError.value = ""
  try { probeResult.value = await props.api.probeDebugServer(target.value) }
  catch (reason) { probeError.value = String(reason) }
  finally { probing.value = false }
}

async function openDevTools() {
  actionError.value = ""
  try { await props.api.openDebugDevTools() }
  catch (reason) { actionError.value = String(reason) }
}

function restoreFocus(event: Event) {
  event.preventDefault()
  void nextTick(() => document.querySelector<HTMLElement>("[data-connection-debug-trigger]")?.focus())
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="flex max-h-[calc(100dvh-2rem)] flex-col overflow-hidden p-4 sm:p-6" @close-auto-focus="restoreFocus">
      <DialogHeader class="shrink-0 pr-6">
        <DialogTitle>{{ t("debugTitle") }}</DialogTitle>
        <DialogDescription>{{ t("debugDescription") }}</DialogDescription>
      </DialogHeader>
      <Tabs v-model="tab" class="flex min-h-0 flex-col gap-4">
        <TabsList class="grid w-full shrink-0 grid-cols-2">
          <TabsTrigger value="status">{{ t("debugStatus") }}</TabsTrigger>
          <TabsTrigger value="checks">{{ t("debugChecks") }}</TabsTrigger>
        </TabsList>
        <div class="min-h-0 overflow-y-auto overscroll-contain">
          <TabsContent value="status" class="mt-0 flex flex-col gap-3">
            <section class="rounded-xl border border-border bg-card p-4">
              <div class="flex items-center justify-between gap-3">
                <h3 class="text-sm font-semibold">{{ t("debugDesktop") }}</h3>
                <Button variant="outline" size="sm" :disabled="loading" @click="refresh">{{ t("refresh") }}</Button>
              </div>
              <p v-if="loading" role="status" class="mt-3 flex items-center gap-2 text-sm text-muted-foreground"><LoaderCircle class="size-4 motion-safe:animate-spin" aria-hidden="true" />{{ t("debugLoading") }}</p>
              <dl v-if="info" class="mt-3 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
                <dt class="text-muted-foreground">{{ t("debugVersion") }}</dt><dd class="min-w-0 break-words">{{ info.version }}</dd>
                <dt class="text-muted-foreground">{{ t("debugBuild") }}</dt><dd class="min-w-0 break-words">{{ info.buildStamp }}</dd>
                <dt class="text-muted-foreground">{{ t("debugPlatform") }}</dt><dd class="min-w-0 break-words">{{ info.platform }} / {{ info.arch }}</dd>
                <dt class="text-muted-foreground">{{ t("debugProxy") }}</dt><dd class="min-w-0 break-words">{{ t(`debugProxy_${info.proxyMode}`) }}</dd>
              </dl>
            </section>
            <section v-if="info" class="rounded-xl border border-border bg-card p-4">
              <h3 class="text-sm font-semibold">{{ t("debugConnections") }}</h3>
              <dl class="mt-3 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
                <dt class="text-muted-foreground">{{ t("debugActive") }}</dt><dd class="min-w-0 break-all">{{ info.activeUrl || t("debugNone") }}</dd>
                <dt class="text-muted-foreground">{{ t("debugLast") }}</dt><dd class="min-w-0 break-all">{{ info.lastUrl || t("debugNone") }}</dd>
                <dt class="text-muted-foreground">{{ t("debugSaved") }}</dt><dd>{{ info.savedCount }}</dd>
              </dl>
            </section>
            <p v-if="loadError" role="alert" class="break-words text-sm text-destructive">{{ loadError }}</p>
          </TabsContent>
          <TabsContent value="checks" class="mt-0 flex flex-col gap-3">
            <section class="rounded-xl border border-border bg-card p-4">
              <form class="flex flex-col gap-3" @submit.prevent="probe">
                <Field class="gap-2">
                  <FieldLabel for="debug-server-address">{{ t("debugProbeAddress") }}</FieldLabel>
                  <Input id="debug-server-address" v-model="target" placeholder="http://192.168.1.20:8081" autocomplete="url" autocapitalize="off" :spellcheck="false" :disabled="probing" required />
                </Field>
                <Button type="submit" variant="outline" size="sm" class="self-end" :disabled="probing || !target.trim()">
                  <LoaderCircle v-if="probing" class="motion-safe:animate-spin" aria-hidden="true" />{{ probing ? t("debugProbing") : t("debugProbe") }}
                </Button>
              </form>
              <p v-if="probeResult" role="status" class="mt-3 break-words text-sm">
                {{ t("debugProbeSuccess") }} · {{ probeResult.name }} · {{ probeResult.version || t("debugLegacy") }} · {{ probeResult.latencyMs }} ms<br />
                <span class="text-xs text-muted-foreground">{{ probeResult.url }} · {{ probeResult.serverId }}</span>
              </p>
              <p v-if="probeError" role="alert" class="mt-3 break-words text-sm text-destructive">{{ probeError }}</p>
            </section>
            <section class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-4">
              <h3 class="text-sm font-semibold">{{ t("debugDevTools") }}</h3>
              <Button variant="outline" size="sm" @click="openDevTools">{{ t("debugOpenDevTools") }}</Button>
              <p v-if="actionError" role="alert" class="w-full break-words text-sm text-destructive">{{ actionError }}</p>
            </section>
          </TabsContent>
        </div>
      </Tabs>
    </DialogContent>
  </Dialog>
</template>
