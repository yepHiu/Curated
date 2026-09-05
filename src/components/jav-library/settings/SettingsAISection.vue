<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { BarChart3, RefreshCw, Server, ShieldCheck } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useAIGovernanceService } from "@/services/ai-governance-service"
import { useLibraryService } from "@/services/library-service"
import { defaultAIGovernance, type AIReport, type AIPage, type AIAuditEntry } from "@/services/contracts/ai-governance-service"
import { applyAIGovernance } from "@/lib/experimental-agent"
import { AGENT_TOOL_I18N_KEYS } from "@/lib/agent-tool-labels"

defineProps<{ useWebApi: boolean }>()
const { t, locale } = useI18n()
const service = useAIGovernanceService()
const library = useLibraryService()
const settings = ref(defaultAIGovernance())
const ready = ref(false)
const busy = ref(false)
const error = ref("")
const message = ref("")
const baseUrl = ref("")
const apiKey = ref("")
const model = ref("")
const days = ref("30")
const channel = ref("all")
const status = ref("all")
const offset = ref(0)
const auditOffset = ref(0)
const report = ref<AIReport | null>(null)
const audit = ref<AIPage<AIAuditEntry> | null>(null)
const loading = ref(false)
let requestId = 0
let disposed = false
const summary = computed(() => report.value?.summary)
const validSettings = computed(() => Number.isInteger(settings.value.stepLimit) && settings.value.stepLimit >= 1 && settings.value.stepLimit <= 30
  && Number.isInteger(settings.value.writePerMinute) && settings.value.writePerMinute >= 1 && settings.value.writePerMinute <= 60
  && Number.isInteger(settings.value.retentionDays) && settings.value.retentionDays >= 7 && settings.value.retentionDays <= 365)

function syncProvider() {
  const provider = library.aiProvider.value
  baseUrl.value = provider.baseUrl
  apiKey.value = provider.apiKey ?? ""
  model.value = provider.model
}
watch(() => library.aiProvider.value, syncProvider)
async function initialize() {
  error.value = ""
  try {
    const next = await service.getSettings()
    if (disposed) return
    settings.value = next
    applyAIGovernance(next)
    ready.value = true
    await refresh()
  } catch (err) { if (!disposed) error.value = (err as Error).message }
}
onMounted(() => { syncProvider(); void initialize() })
onBeforeUnmount(() => { disposed = true; requestId++ })

async function refresh() {
  const id = ++requestId
  loading.value = true
  error.value = ""
  report.value = null
  audit.value = null
  const query = { days: Number(days.value), channel: channel.value === "all" ? undefined : channel.value, limit: 25 }
  try {
    const [nextReport, nextAudit] = await Promise.all([
      service.getUsage({ ...query, status: status.value === "all" ? undefined : status.value, offset: offset.value }),
      service.getAudit({ ...query, status: status.value === "failed" ? "failed" : undefined, offset: auditOffset.value }),
    ])
    if (id !== requestId || disposed) return
    report.value = nextReport
    audit.value = nextAudit
  } catch (err) { if (id === requestId && !disposed) error.value = (err as Error).message }
  finally { if (id === requestId) loading.value = false }
}
watch([days, channel, status], () => { offset.value = 0; auditOffset.value = 0; void refresh() })
async function perform(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true; error.value = ""; message.value = ""
  try { await action() } catch (err) { error.value = (err as Error).message }
  finally { busy.value = false }
}
function saveSettings() {
  if (!validSettings.value) return
  void perform(async () => {
    settings.value = await service.saveSettings({ ...settings.value })
    applyAIGovernance(settings.value)
    message.value = t("aiSettings.saved")
  })
}
function saveProvider() {
  void perform(async () => {
    await library.setAIProvider({ baseUrl: baseUrl.value.trim(), apiKey: apiKey.value, model: model.value.trim() })
    message.value = t("aiSettings.saved")
  })
}
function testProvider() {
  void perform(async () => {
    const result = await library.testAIProvider({ kind: "openai-compatible", baseUrl: baseUrl.value.trim(), apiKey: apiKey.value, model: model.value.trim() })
    await refresh()
    if (!result.ok) error.value = t("settings.experimentalTestFail", { message: result.message ?? "" })
    else message.value = t("settings.experimentalTestOk", { ms: result.latencyMs })
  })
}
function cleanup() {
  void perform(async () => {
    const result = await service.cleanup()
    message.value = t("aiSettings.cleaned", { count: result.runs + result.audit + result.receipts })
    await refresh()
  })
}
function duration(value: number | null | undefined) { return value == null ? t("aiSettings.unknown") : t("aiSettings.milliseconds", { value: Math.round(value).toLocaleString(locale.value) }) }
function time(value: string) { return new Date(value).toLocaleString(locale.value) }
function measured(total: number, known: number, calls: number) {
  if (known === 0) return t("aiSettings.unknown")
  return total.toLocaleString(locale.value) + (known < calls ? ` (${t("aiSettings.partialUsage")})` : "")
}
function toolLabel(name: string) { const key = AGENT_TOOL_I18N_KEYS[name]; return key ? t(key) : name }
function page(kind: "runs" | "audit", delta: number) { if (kind === "runs") offset.value += delta; else auditOffset.value += delta; void refresh() }
</script>

<template>
  <div class="flex min-w-0 flex-col gap-6" data-ai-settings>
    <p v-if="error" role="alert" class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive" data-ai-settings-error>{{ error }}</p>
    <p v-if="message" role="status" class="rounded-lg border border-border/50 bg-muted/30 px-3 py-2 text-sm text-muted-foreground">{{ message }}</p>
    <Button v-if="!ready" variant="outline" size="sm" class="self-start" @click="initialize">{{ t('aiSettings.retry') }}</Button>
    <template v-if="ready">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-governance-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
          <span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><ShieldCheck class="size-4" /></span>
          <CardTitle class="min-w-0 text-lg tracking-tight">{{ t('aiSettings.governance') }}</CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <section class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" data-ai-settings-block="policy">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.policy') }}</h3>
            <FieldGroup class="gap-3">
              <Field orientation="horizontal" class="rounded-lg border border-border/40 bg-background/30 px-3 py-2"><FieldLabel for="ai-enabled">{{ t('aiSettings.enabled') }}</FieldLabel><Switch id="ai-enabled" v-model="settings.enabled" :disabled="busy" data-ai-enabled /></Field>
              <Field orientation="horizontal" class="rounded-lg border border-border/40 bg-background/30 px-3 py-2"><FieldLabel for="ai-readonly">{{ t('aiSettings.readOnly') }}</FieldLabel><Switch id="ai-readonly" v-model="settings.readOnly" :disabled="busy" data-ai-readonly /></Field>
              <FieldDescription>{{ t('aiSettings.confirmationHint') }}</FieldDescription>
            </FieldGroup>
          </section>
          <section class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" data-ai-settings-block="limits">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.limits') }}</h3>
            <FieldGroup class="gap-3">
              <Field><FieldLabel for="ai-privacy">{{ t('aiSettings.privacy') }}</FieldLabel>
                <Select v-model="settings.privacy" :disabled="busy"><SelectTrigger id="ai-privacy"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem value="auto">{{ t('aiSettings.privacyAuto') }}</SelectItem><SelectItem value="minimal">{{ t('aiSettings.privacyMinimal') }}</SelectItem></SelectGroup></SelectContent></Select>
                <FieldDescription>{{ t('aiSettings.privacyHint') }}</FieldDescription>
              </Field>
              <FieldGroup class="grid grid-cols-1 gap-3 sm:grid-cols-3">
                <Field><FieldLabel for="ai-steps">{{ t('aiSettings.stepLimit') }}</FieldLabel><Input id="ai-steps" v-model.number="settings.stepLimit" type="number" min="1" max="30" :disabled="busy" /></Field>
                <Field><FieldLabel for="ai-rate">{{ t('aiSettings.writeLimit') }}</FieldLabel><Input id="ai-rate" v-model.number="settings.writePerMinute" type="number" min="1" max="60" :disabled="busy" /></Field>
                <Field><FieldLabel for="ai-retention">{{ t('aiSettings.retention') }}</FieldLabel><Input id="ai-retention" v-model.number="settings.retentionDays" type="number" min="7" max="365" :disabled="busy" /></Field>
              </FieldGroup>
              <FieldDescription>{{ t('aiSettings.retentionHint') }}</FieldDescription>
            </FieldGroup>
          </section>
          <div class="flex flex-wrap justify-end gap-2 rounded-lg border border-border/40 bg-background/30 p-3">
            <Button variant="outline" size="sm" :disabled="busy" data-ai-cleanup @click="cleanup">{{ t('aiSettings.cleanup') }}</Button><Button size="sm" :disabled="busy || !validSettings" data-ai-save @click="saveSettings">{{ t('settings.experimentalSave') }}</Button>
          </div>
        </CardContent>
      </Card>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-provider-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0"><span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><Server class="size-4" /></span><CardTitle class="min-w-0 text-lg tracking-tight">{{ t('aiSettings.provider') }}</CardTitle></CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <section class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <FieldGroup class="grid grid-cols-1 gap-3 md:grid-cols-2">
              <Field class="md:col-span-2"><FieldLabel for="ai-base">{{ t('settings.experimentalBaseUrl') }}</FieldLabel><Input id="ai-base" v-model="baseUrl" autocomplete="off" :disabled="busy" /></Field>
              <Field><FieldLabel for="ai-key">{{ t('settings.experimentalApiKey') }}</FieldLabel><Input id="ai-key" v-model="apiKey" type="password" autocomplete="new-password" :disabled="busy" /></Field>
              <Field><FieldLabel for="ai-model">{{ t('settings.experimentalModel') }}</FieldLabel><Input id="ai-model" v-model="model" autocomplete="off" :disabled="busy" /></Field>
            </FieldGroup>
          </section>
          <div class="flex flex-col gap-3 rounded-lg border border-border/40 bg-background/30 p-3 sm:flex-row sm:items-center sm:justify-between"><FieldDescription>{{ t('aiSettings.providerHint') }}</FieldDescription><div class="flex shrink-0 flex-wrap gap-2"><Button variant="outline" size="sm" :disabled="busy" data-ai-provider-test @click="testProvider">{{ t('settings.experimentalTest') }}</Button><Button size="sm" :disabled="busy" data-ai-provider-save @click="saveProvider">{{ t('settings.experimentalSave') }}</Button></div></div>
        </CardContent>
      </Card>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-statistics-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0"><span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><BarChart3 class="size-4" /></span><CardTitle class="min-w-0 text-lg tracking-tight">{{ t('aiSettings.statistics') }}</CardTitle></CardHeader>
        <CardContent class="flex min-w-0 flex-col gap-4">
          <p v-if="!useWebApi" class="text-sm text-muted-foreground">{{ t('aiSettings.mockHint') }}</p>
          <section class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" data-ai-settings-block="filters">
            <FieldGroup class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <Field><FieldLabel for="ai-days">{{ t('aiSettings.range') }}</FieldLabel><Select v-model="days"><SelectTrigger id="ai-days"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="day in ['7','30','90']" :key="day" :value="day">{{ t('aiSettings.days', { count: day }) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
              <Field><FieldLabel for="ai-channel">{{ t('aiSettings.channel') }}</FieldLabel><Select v-model="channel"><SelectTrigger id="ai-channel"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in ['all','chat','action','test']" :key="item" :value="item">{{ t(`aiSettings.channels.${item}`) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
              <Field><FieldLabel for="ai-status">{{ t('aiSettings.status') }}</FieldLabel><Select v-model="status"><SelectTrigger id="ai-status"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in ['all','completed','failed','partial','cancelled','needs_input']" :key="item" :value="item">{{ t(`aiSettings.statuses.${item}`) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
            </FieldGroup>
            <Button variant="outline" size="sm" :disabled="loading" class="self-end" data-ai-refresh @click="refresh"><RefreshCw data-icon="inline-start" />{{ t('aiSettings.refresh') }}</Button>
          </section>
          <p v-if="loading" role="status">{{ t('aiSettings.loading') }}</p>
          <dl v-if="summary" class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4" data-ai-summary>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.runs') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ summary.runs }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.failed') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ summary.failed }} / {{ summary.partial }} / {{ summary.cancelled }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.totalTokens') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums" data-ai-token-total>{{ measured(summary.totalTokens, summary.usageCalls, summary.modelCalls) }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.coverage') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ summary.usageCalls }} / {{ summary.modelCalls }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.inputOutput') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ measured(summary.promptTokens, summary.usageCalls, summary.modelCalls) }} / {{ measured(summary.completionTokens, summary.usageCalls, summary.modelCalls) }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.toolCalls') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ summary.toolCalls }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.firstText') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ duration(summary.avgFirstTextMs) }}</dd></div>
            <div class="rounded-lg border border-border/50 bg-muted/5 p-3"><dt class="text-xs text-muted-foreground">{{ t('aiSettings.duration') }}</dt><dd class="mt-1 text-xl font-semibold tabular-nums">{{ duration(summary.avgDurationMs) }}</dd></div>
          </dl>
          <p class="rounded-lg border border-border/40 bg-background/30 p-3 text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t('aiSettings.usageHint') }}</p>
          <section class="flex min-w-0 flex-col gap-3" data-ai-settings-block="recent-runs">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.recentRuns') }}</h3>
            <p v-if="report && !report.total" class="text-sm text-muted-foreground">{{ t('aiSettings.empty') }}</p>
            <ul class="flex min-w-0 flex-col gap-2">
              <li v-for="run in report?.items ?? []" :key="run.id" class="flex min-w-0 flex-col gap-2 rounded-lg border border-border/50 bg-muted/5 p-3">
              <div class="flex flex-wrap items-center justify-between gap-2"><span class="break-all text-sm">{{ run.model || t('aiSettings.unknown') }}</span><Badge variant="secondary">{{ t(`aiSettings.statuses.${run.status}`) }}</Badge></div>
              <p class="break-all text-xs text-muted-foreground">{{ time(run.startedAt) }} · {{ t(`aiSettings.channels.${run.channel}`) }} · {{ run.promptVersion }} · {{ run.provider }}</p>
              <p class="text-sm">{{ duration(run.durationMs) }} · {{ t('aiSettings.firstText') }}: {{ duration(run.firstTextMs) }} · {{ t('aiSettings.totalTokens') }}: {{ measured(run.totalTokens, run.usageCalls, run.modelCalls) }} · {{ t('aiSettings.toolCalls') }}: {{ run.toolCalls }}</p>
              <p v-if="run.errorCode" class="break-all text-sm text-destructive">{{ t('aiSettings.error') }}: {{ run.errorCode }}</p>
              </li>
            </ul>
            <div v-if="report && report.total > report.limit" class="flex flex-wrap items-center justify-end gap-2"><span class="mr-auto text-xs text-muted-foreground">{{ offset + 1 }}–{{ Math.min(offset + report.limit, report.total) }} / {{ report.total }}</span><Button variant="outline" size="sm" :disabled="loading || offset === 0" @click="page('runs', -25)">{{ t('aiSettings.previous') }}</Button><Button variant="outline" size="sm" :disabled="loading || offset + 25 >= report.total" @click="page('runs', 25)">{{ t('aiSettings.next') }}</Button></div>
          </section>
          <section class="flex min-w-0 flex-col gap-3" data-ai-settings-block="audit">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.audit') }}</h3>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t('aiSettings.auditHint') }}</p>
            <p v-if="audit && !audit.total" class="text-sm text-muted-foreground">{{ t('aiSettings.empty') }}</p>
            <ul class="flex min-w-0 flex-col gap-2"><li v-for="entry in audit?.items ?? []" :key="entry.id" class="flex min-w-0 flex-col gap-2 rounded-lg border border-border/50 bg-muted/5 p-3"><div class="flex flex-wrap justify-between gap-2"><span class="break-all text-sm">{{ toolLabel(entry.tool) }}</span><Badge variant="secondary">{{ t(`aiSettings.auditResults.${entry.result}`) }}</Badge></div><p class="break-all text-xs text-muted-foreground">{{ time(entry.createdAt) }} · {{ entry.permission }} · {{ duration(entry.durationMs) }}</p><p v-if="entry.errorCode" class="break-all text-sm text-destructive">{{ entry.errorCode }}</p></li></ul>
            <div v-if="audit && audit.total > audit.limit" class="flex flex-wrap justify-end gap-2"><Button variant="outline" size="sm" :disabled="loading || auditOffset === 0" @click="page('audit', -25)">{{ t('aiSettings.previous') }}</Button><Button variant="outline" size="sm" :disabled="loading || auditOffset + 25 >= audit.total" @click="page('audit', 25)">{{ t('aiSettings.next') }}</Button></div>
          </section>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
