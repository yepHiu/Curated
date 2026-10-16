<script setup lang="ts">
import SettingsSubtitlesSection from "./SettingsSubtitlesSection.vue"
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { AlertCircle, BarChart3, Check, ChevronRight, Eye, EyeOff, Loader2, RefreshCw, Server, ShieldCheck, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useAIGovernanceService } from "@/services/ai-governance-service"
import { useLibraryService } from "@/services/library-service"
import { defaultAIGovernance, type AISummary, type AIReport, type AIPage, type AIAuditEntry, type AIRun } from "@/services/contracts/ai-governance-service"
import type { StatusTone } from "@/lib/ui/status-tone"
import { applyAIGovernance } from "@/lib/experimental-agent"
import { AGENT_TOOL_I18N_KEYS } from "@/lib/agent-tool-labels"
import { useAISettingsAutosave } from "@/composables/use-ai-settings-autosave"
import { pushAppToast } from "@/composables/use-app-toast"
import { AI_CONTEXT_PRESETS, DEFAULT_AI_CONTEXT_WINDOW, MIN_AI_CONTEXT_WINDOW, MAX_AI_CONTEXT_WINDOW, validAIContextWindow } from "@/lib/ai-context-presets"

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
const apiKeyVisible = ref(false)
const model = ref("")
const contextWindow = ref<number | string>(DEFAULT_AI_CONTEXT_WINDOW)
const contextPreset = ref("custom")
const selectedContextPreset = computed(() => AI_CONTEXT_PRESETS.find(preset => preset.id === contextPreset.value))
const validContext = computed(() => validAIContextWindow(Number(contextWindow.value)))
function applyContextPreset(value: unknown) {
  const preset = AI_CONTEXT_PRESETS.find(item => item.id === value)
  contextPreset.value = preset?.id ?? "custom"
  if (preset) contextWindow.value = preset.tokens
}
watch(contextWindow, value => {
  if (selectedContextPreset.value?.tokens !== Number(value)) contextPreset.value = "custom"
})
const testingProvider = ref(false)
const providerTestResult = ref<{ ok: boolean; text: string } | null>(null)
let providerTestResetTimer: ReturnType<typeof setTimeout> | undefined
function clearProviderTestResetTimer() {
  if (providerTestResetTimer !== undefined) clearTimeout(providerTestResetTimer)
  providerTestResetTimer = undefined
}
watch(providerTestResult, result => {
  clearProviderTestResetTimer()
  if (result?.ok) {
    providerTestResetTimer = setTimeout(() => { providerTestResult.value = null }, 3000)
  }
})
const days = ref("30")
const channel = ref("all")
const status = ref("all")
const runsPageSize = ref("20")
const auditPageSize = ref("20")
const offset = ref(0)
const auditOffset = ref(0)
const report = ref<AIReport | null>(null)
const audit = ref<AIPage<AIAuditEntry> | null>(null)
const loading = ref(false)
const paging = ref<"runs" | "audit" | null>(null)
const pageError = ref<{ kind: "runs" | "audit"; message: string } | null>(null)
const runsList = ref<HTMLElement | null>(null)
const auditList = ref<HTMLElement | null>(null)
const runsMinHeight = ref(0)
const auditMinHeight = ref(0)
let requestId = 0
let disposed = false
const summary = ref<AISummary | null>(null)
const inputShare = computed(() => summary.value?.totalTokens ? Math.min(100, Math.max(0, summary.value.promptTokens / summary.value.totalTokens * 100)) : 0)
const recordHeading = ref<HTMLElement | null>(null)
const expandFirstFailure = ref(false)
const expandedRuns = ref<Record<string, boolean>>({})
const usagePeriod = computed(() => {
  const end = new Date()
  const start = new Date(end)
  if (days.value === "today") start.setHours(0, 0, 0, 0)
  else start.setDate(start.getDate() - Number(days.value))
  return `${start.toLocaleDateString(locale.value)} – ${end.toLocaleDateString(locale.value)}`
})
function periodQuery() {
  if (days.value !== "today") return { days: Number(days.value) }
  const start = new Date()
  start.setHours(0, 0, 0, 0)
  return { since: start.toISOString() }
}
function viewFailures() {
  expandFirstFailure.value = true
  if (status.value === "failed" && channel.value === "all") {
    if (report.value?.items[0]) expandedRuns.value[report.value.items[0].id] = true
    expandFirstFailure.value = false
  } else {
    channel.value = "all"
    status.value = "failed"
  }
  recordHeading.value?.focus({ preventScroll: true })
  recordHeading.value?.scrollIntoView?.({ block: "nearest" })
}
const validSettings = computed(() => Number.isInteger(settings.value.stepLimit) && settings.value.stepLimit >= 0 && settings.value.stepLimit <= 30
  && Number.isInteger(settings.value.writePerMinute) && settings.value.writePerMinute >= 1 && settings.value.writePerMinute <= 60
  && Number.isInteger(settings.value.retentionDays) && settings.value.retentionDays >= 7 && settings.value.retentionDays <= 365)

const lastStepLimit = ref(15)
const stepLimitEnabled = computed({
  get: () => settings.value.stepLimit !== 0,
  set: (enabled: boolean) => {
    if (!enabled && Number.isInteger(settings.value.stepLimit) && settings.value.stepLimit > 0 && settings.value.stepLimit <= 30) {
      lastStepLimit.value = settings.value.stepLimit
    }
    settings.value.stepLimit = enabled ? lastStepLimit.value : 0
  },
})

const settingsAutosave = useAISettingsAutosave({
  read: () => ({ ...settings.value }),
  enabled: () => ready.value,
  valid: () => validSettings.value,
  delay: (next, previous) => next.enabled !== previous.enabled || next.readOnly !== previous.readOnly || next.privacy !== previous.privacy || (next.stepLimit === 0) !== (previous.stepLimit === 0) ? 0 : 550,
  save: async (value) => { applyAIGovernance(await service.saveSettings(value)) },
  onDetachedError: (detail) => pushAppToast(t("aiSettings.autoSaveFailed", { message: detail }), { variant: "destructive" }),
})
const providerAutosave = useAISettingsAutosave({
  read: () => ({ baseUrl: baseUrl.value.trim(), apiKey: apiKey.value, model: model.value.trim(), contextWindow: Number(contextWindow.value) }),
  enabled: () => ready.value,
  valid: () => validContext.value,
  save: (value) => library.setAIProvider(value),
  onDetachedError: (detail) => pushAppToast(t("aiSettings.autoSaveFailed", { message: detail }), { variant: "destructive" }),
})

function syncProvider() {
  if (ready.value && (providerAutosave.dirty.value || providerAutosave.saving.value)) return
  const provider = library.aiProvider.value
  const capacity = provider.contextWindow || DEFAULT_AI_CONTEXT_WINDOW
  providerAutosave.initialize({ baseUrl: provider.baseUrl.trim(), apiKey: provider.apiKey ?? "", model: provider.model.trim(), contextWindow: capacity })
  baseUrl.value = provider.baseUrl
  apiKey.value = provider.apiKey ?? ""
  model.value = provider.model
  contextWindow.value = capacity
}
watch(() => library.aiProvider.value, syncProvider)
watch([baseUrl, apiKey, model, contextWindow], () => { providerTestResult.value = null })
async function initialize() {
  error.value = ""
  try {
    const next = await service.getSettings()
    if (disposed) return
    settingsAutosave.initialize(next)
    settings.value = next
    applyAIGovernance(next)
    ready.value = true
    await refresh()
  } catch (err) { if (!disposed) error.value = (err as Error).message }
}
onMounted(() => { syncProvider(); void initialize() })
onBeforeUnmount(() => { disposed = true; requestId++; clearProviderTestResetTimer() })

async function refresh() {
  const id = ++requestId
  paging.value = null
  pageError.value = null
  runsMinHeight.value = 0
  auditMinHeight.value = 0
  loading.value = true
  error.value = ""
  report.value = null
  audit.value = null
  summary.value = null
  expandedRuns.value = {}
  const query = { ...periodQuery(), channel: channel.value === "all" ? undefined : channel.value }
  try {
    const [nextReport, nextAudit, overview] = await Promise.all([
      service.getUsage({ ...query, status: status.value === "all" ? undefined : status.value, offset: offset.value, limit: Number(runsPageSize.value) }),
      service.getAudit({ ...query, status: status.value === "failed" ? "failed" : undefined, offset: auditOffset.value, limit: Number(auditPageSize.value) }),
      channel.value !== "all" || status.value !== "all" ? service.getUsage({ ...periodQuery(), limit: 1, offset: 0 }) : Promise.resolve(null),
    ])
    if (id !== requestId || disposed) return
    report.value = nextReport
    audit.value = nextAudit
    summary.value = overview?.summary ?? nextReport.summary
    if (expandFirstFailure.value && nextReport.items[0]) expandedRuns.value[nextReport.items[0].id] = true
    expandFirstFailure.value = false
  } catch (err) { if (id === requestId && !disposed) error.value = (err as Error).message }
  finally { if (id === requestId) loading.value = false }
}
watch([days, channel, status, runsPageSize, auditPageSize], () => { offset.value = 0; auditOffset.value = 0; void refresh() })
async function perform(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true; error.value = ""; message.value = ""
  try { await action() } catch (err) { error.value = (err as Error).message }
  finally { busy.value = false }
}
async function testProvider() {
  if (busy.value) return
  busy.value = true
  testingProvider.value = true
  error.value = ""
  message.value = ""
  providerTestResult.value = null
  try {
    if (!await providerAutosave.flush() || disposed) return
    const result = await library.testAIProvider({ kind: "openai-compatible", baseUrl: baseUrl.value.trim(), apiKey: apiKey.value, model: model.value.trim(), contextWindow: Number(contextWindow.value) })
    await refresh()
    if (disposed) return
    providerTestResult.value = result.ok
      ? { ok: true, text: t("settings.experimentalTestOk", { ms: result.latencyMs }) }
      : { ok: false, text: t("settings.experimentalTestFail", { message: result.message ?? "" }) }
  } catch (err) {
    if (!disposed) providerTestResult.value = { ok: false, text: t("settings.experimentalTestFail", { message: (err as Error).message }) }
  } finally {
    busy.value = false
    testingProvider.value = false
  }
}
function cleanup() {
  void perform(async () => {
    const result = await service.cleanup()
    message.value = t("aiSettings.cleaned", { count: result.runs + result.audit + result.receipts })
    await refresh()
  })
}
function duration(value: number | null | undefined) {
  if (value == null) return t("aiSettings.unknown")
  if (value >= 1000) return t("aiSettings.seconds", { value: (value / 1000).toLocaleString(locale.value, { maximumFractionDigits: 1 }) })
  return t("aiSettings.milliseconds", { value: Math.round(value).toLocaleString(locale.value) })
}
const knownRunErrors = new Set(["timeout", "authentication", "rate_limit", "network", "configuration", "stream_interrupted", "provider_http", "invalid_response"])
function runError(run: AIRun) { return t(`aiSettings.runErrors.${knownRunErrors.has(run.errorCode) ? run.errorCode : "other"}`) }
function time(value: string) { return new Date(value).toLocaleString(locale.value) }
function measured(total: number, known: number, calls: number) {
  if (known === 0) return t("aiSettings.unknown")
  return total.toLocaleString(locale.value) + (known < calls ? ` (${t("aiSettings.partialUsage")})` : "")
}
function toolLabel(name: string) { const key = AGENT_TOOL_I18N_KEYS[name]; return key ? t(key) : name }
async function page(kind: "runs" | "audit", direction: -1 | 1) {
  if (loading.value) return
  const pageSize = kind === "runs" ? Number(runsPageSize.value) : Number(auditPageSize.value)
  const current = kind === "runs" ? report.value : audit.value
  if (!current) return
  const nextOffset = Math.max(0, current.offset + direction * pageSize)
  if (nextOffset === current.offset || nextOffset >= current.total) return
  // Keep the list footprint when loading and when the final page has fewer rows.
  const list = kind === "runs" ? runsList : auditList
  const minHeight = kind === "runs" ? runsMinHeight : auditMinHeight
  minHeight.value = Math.max(minHeight.value, list.value?.getBoundingClientRect().height ?? 0)
  const id = ++requestId
  loading.value = true
  paging.value = kind
  pageError.value = null
  const query = { ...periodQuery(), channel: channel.value === "all" ? undefined : channel.value, offset: nextOffset, limit: pageSize }
  try {
    if (kind === "runs") {
      const next = await service.getUsage({ ...query, status: status.value === "all" ? undefined : status.value })
      if (id !== requestId || disposed) return
      report.value = next
      offset.value = next.offset
    } else {
      const next = await service.getAudit({ ...query, status: status.value === "failed" ? "failed" : undefined })
      if (id !== requestId || disposed) return
      audit.value = next
      auditOffset.value = next.offset
    }
  } catch (err) {
    if (id === requestId && !disposed) pageError.value = { kind, message: (err as Error).message }
  } finally {
    if (id === requestId) { loading.value = false; paging.value = null }
  }
}
function runStatusTone(value: string): StatusTone {
  if (value === "completed") return "success"
  if (value === "failed") return "danger"
  if (value === "partial" || value === "needs_input" || value === "needs_confirmation") return "warning"
  return "info"
}
function auditStatusTone(value: string): StatusTone {
  if (value === "ok" || value === "confirmed") return "success"
  if (value === "error" || value === "rejected") return "danger"
  if (value === "previewed") return "warning"
  return "info"
}
function runLine(run: AIRun) {
  return `${run.model || t("aiSettings.unknown")} · ${t(`aiSettings.statuses.${run.status}`)} · ${time(run.startedAt)} · ${t(`aiSettings.channels.${run.channel}`)} · ${duration(run.durationMs)}${run.errorCode ? ` · ${run.errorCode}` : ""}`
}
function auditLine(entry: AIAuditEntry) {
  return `${toolLabel(entry.tool)} · ${t(`aiSettings.auditResults.${entry.result}`)} · ${time(entry.createdAt)} · ${entry.permission} · ${duration(entry.durationMs)}${entry.errorCode ? ` · ${entry.errorCode}` : ""}`
}
</script>

<template>
  <div class="flex min-w-0 flex-col gap-6" data-ai-settings>
    <p v-if="error" role="alert" class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive" data-ai-settings-error>{{ error }}</p>
    <p v-if="message" role="status" class="rounded-lg border border-border/50 bg-muted/30 px-3 py-2 text-sm text-muted-foreground">{{ message }}</p>
    <Button v-if="!ready" variant="outline" size="sm" class="self-end" @click="initialize">{{ t('aiSettings.retry') }}</Button>
    <template v-if="ready">
      <section class="rounded-xl border border-border bg-card p-6 shadow-sm" data-ai-enable-section>
        <FieldGroup>
          <Field orientation="horizontal" class="items-center justify-between">
            <FieldLabel for="ai-enabled" class="text-lg font-semibold">{{ t('aiSettings.enabled') }}</FieldLabel>
            <Switch id="ai-enabled" v-model="settings.enabled" :disabled="busy" data-ai-enabled />
          </Field>
        </FieldGroup>
        <div v-if="!settings.enabled && settingsAutosave.error.value" class="mt-3 flex flex-wrap items-center justify-end gap-2 text-xs">
          <p role="alert" class="text-destructive">{{ t('aiSettings.autoSaveFailed', { message: settingsAutosave.error.value }) }}</p>
          <Button variant="outline" size="sm" class="rounded-full" data-ai-policy-retry @click="settingsAutosave.flush()">{{ t('aiSettings.retry') }}</Button>
        </div>
      </section>
      <div v-show="settings.enabled" class="flex min-w-0 flex-col gap-6" data-ai-dependent-settings>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-governance-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
          <span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><ShieldCheck class="size-4" /></span>
          <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t('aiSettings.governance') }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <section class="flex min-w-0 flex-col gap-3" data-ai-settings-block="policy">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.policy') }}</h3>
            <FieldGroup class="gap-3">
              <Field orientation="horizontal" class="min-w-0 justify-between rounded-lg border border-border/50 bg-muted/5 p-4"><SettingsHint :text="[t('aiSettings.confirmationHint'), t('aiSettings.autoSaveHint')].join('\n\n')">
                  <FieldLabel for="ai-readonly">{{ t('aiSettings.readOnly') }}</FieldLabel>
                </SettingsHint><Switch id="ai-readonly" v-model="settings.readOnly" :disabled="busy" data-ai-readonly /></Field>
            </FieldGroup>
          </section>
          <section class="flex min-w-0 flex-col gap-3" data-ai-settings-block="limits">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.limits') }}</h3>
            <FieldGroup class="gap-3">
              <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"><SettingsHint :text="t('aiSettings.privacyHint')">
                  <FieldLabel for="ai-privacy">{{ t('aiSettings.privacy') }}</FieldLabel>
                </SettingsHint>
                <Select v-model="settings.privacy" :disabled="busy"><SelectTrigger id="ai-privacy" class="w-full min-w-0 sm:w-52 sm:shrink-0"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem value="auto">{{ t('aiSettings.privacyAuto') }}</SelectItem><SelectItem value="minimal">{{ t('aiSettings.privacyMinimal') }}</SelectItem></SelectGroup></SelectContent></Select>
              </Field>
              <Field orientation="horizontal" class="min-w-0 justify-between rounded-lg border border-border/50 bg-muted/5 p-4">
                <SettingsHint :text="t('aiSettings.stepLimitHint')">
                  <FieldLabel for="ai-step-limit-enabled">{{ t('aiSettings.stepLimitEnabled') }}</FieldLabel>
                </SettingsHint>
                <Switch id="ai-step-limit-enabled" v-model="stepLimitEnabled" :disabled="busy" aria-describedby="ai-step-limit-hint" />
              </Field>
              <span id="ai-step-limit-hint" class="sr-only">{{ t('aiSettings.stepLimitHint') }}</span>
              <FieldGroup class="gap-3" data-ai-limit-fields>
                <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4" v-if="stepLimitEnabled" :data-invalid="!validSettings"><FieldLabel for="ai-steps">{{ t('aiSettings.stepLimit') }}</FieldLabel><Input class="w-full min-w-0 sm:w-52 sm:shrink-0" id="ai-steps" v-model.number="settings.stepLimit" :aria-invalid="!Number.isInteger(settings.stepLimit) || settings.stepLimit < 1 || settings.stepLimit > 30" @blur="settingsAutosave.flush()" type="number" min="1" max="30" :disabled="busy" /></Field>
                <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"><FieldLabel for="ai-rate">{{ t('aiSettings.writeLimit') }}</FieldLabel><Input class="w-full min-w-0 sm:w-52 sm:shrink-0" id="ai-rate" v-model.number="settings.writePerMinute" :aria-invalid="!Number.isInteger(settings.writePerMinute) || settings.writePerMinute < 1 || settings.writePerMinute > 60" @blur="settingsAutosave.flush()" type="number" min="1" max="60" :disabled="busy" /></Field>
                <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"><SettingsHint :text="t('aiSettings.retentionHint')">
                  <FieldLabel for="ai-retention">{{ t('aiSettings.retention') }}</FieldLabel>
                </SettingsHint><Input class="w-full min-w-0 sm:w-52 sm:shrink-0" id="ai-retention" v-model.number="settings.retentionDays" :aria-invalid="!Number.isInteger(settings.retentionDays) || settings.retentionDays < 7 || settings.retentionDays > 365" @blur="settingsAutosave.flush()" type="number" min="7" max="365" :disabled="busy" /></Field>
              </FieldGroup>
            </FieldGroup>
          </section>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex min-h-8 min-w-0 flex-1 flex-wrap items-center gap-2 text-xs" data-ai-policy-save-status>
              <p v-if="!validSettings" role="alert" class="text-destructive">{{ t('aiSettings.invalidLimits') }}</p>
              <template v-else-if="settingsAutosave.error.value">
                <p role="alert" class="break-words text-destructive">{{ t('aiSettings.autoSaveFailed', { message: settingsAutosave.error.value }) }}</p>
                <Button variant="outline" size="sm" data-ai-policy-retry @click="settingsAutosave.flush()">{{ t('aiSettings.retry') }}</Button>
              </template>
              <p v-else-if="settingsAutosave.saving.value || settingsAutosave.dirty.value || settingsAutosave.saved.value" role="status" class="text-muted-foreground">{{ settingsAutosave.saving.value || settingsAutosave.dirty.value ? t('common.saving') : t('settings.autoPersistSaved') }}</p>
            </div>
            <Button variant="outline" size="sm" class="ml-auto rounded-full" :disabled="busy" data-ai-cleanup @click="cleanup">{{ t('aiSettings.cleanup') }}</Button>
          </div>
        </CardContent>
      </Card>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-provider-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0"><span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><Server class="size-4" /></span><CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t('aiSettings.provider') }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle></CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <section class="flex min-w-0 flex-col gap-3">
            <FieldGroup class="gap-3">
              <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"><FieldLabel for="ai-base">{{ t('settings.experimentalBaseUrl') }}</FieldLabel><Input class="w-full min-w-0 sm:w-72 sm:shrink-0" id="ai-base" v-model="baseUrl" @blur="providerAutosave.flush()" autocomplete="off" :disabled="busy" /></Field>
              <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                <FieldLabel for="ai-key">{{ t('settings.experimentalApiKey') }}</FieldLabel>
                <div class="flex w-full min-w-0 items-center gap-2 sm:w-72 sm:shrink-0">
                  <Input id="ai-key" v-model="apiKey" @blur="providerAutosave.flush()" :type="apiKeyVisible ? 'text' : 'password'" autocomplete="new-password" :disabled="busy" />
                  <Button type="button" variant="ghost" size="icon" class="shrink-0 rounded-full" :disabled="busy" :aria-label="t(apiKeyVisible ? 'subtitles.hideKey' : 'subtitles.showKey')" :aria-pressed="apiKeyVisible" @click="apiKeyVisible = !apiKeyVisible"><EyeOff v-if="apiKeyVisible" aria-hidden="true" /><Eye v-else aria-hidden="true" /></Button>
                </div>
              </Field>
              <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"><FieldLabel for="ai-model">{{ t('settings.experimentalModel') }}</FieldLabel><Input class="w-full min-w-0 sm:w-72 sm:shrink-0" id="ai-model" v-model="model" @blur="providerAutosave.flush()" autocomplete="off" :disabled="busy" /></Field>
              <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4" :data-invalid="!validContext || undefined">
                <SettingsHint :text="t('aiSettings.contextHint')">
                  <FieldLabel for="ai-context-window">{{ t('aiSettings.contextWindow') }}</FieldLabel>
                  <template #extra><a v-if="selectedContextPreset" :href="selectedContextPreset.source" target="_blank" rel="noopener noreferrer" class="underline underline-offset-4">{{ t('aiSettings.contextSource') }}</a></template>
                </SettingsHint>
                <Input class="w-full min-w-0 sm:w-72 sm:shrink-0" id="ai-context-window" v-model="contextWindow" type="number" inputmode="numeric" :min="MIN_AI_CONTEXT_WINDOW" :max="MAX_AI_CONTEXT_WINDOW" :step="1" :disabled="busy" :aria-invalid="!validContext" :aria-describedby="validContext ? 'ai-context-help' : 'ai-context-help ai-context-validation'" @blur="providerAutosave.flush()" />
              </Field>
              <Field orientation="horizontal" class="flex min-w-0 flex-col items-stretch gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                <FieldLabel for="ai-context-preset">{{ t('aiSettings.contextPreset') }}</FieldLabel>
                <Select :model-value="contextPreset" :disabled="busy" @update:model-value="applyContextPreset">
                  <SelectTrigger id="ai-context-preset" class="w-full min-w-0 sm:w-72 sm:shrink-0"><SelectValue /></SelectTrigger>
                  <SelectContent><SelectGroup>
                    <SelectItem value="custom">{{ t('aiSettings.contextCustom') }}</SelectItem>
                    <SelectItem v-for="preset in AI_CONTEXT_PRESETS" :key="preset.id" :value="preset.id">{{ preset.label }} · {{ preset.tokens.toLocaleString(locale) }}</SelectItem>
                  </SelectGroup></SelectContent>
                </Select>
              </Field>
              <span id="ai-context-help" class="sr-only">{{ t('aiSettings.contextHint') }}</span>
              <p v-if="!validContext" id="ai-context-validation" role="alert" class="text-sm text-destructive">{{ t('aiSettings.contextInvalid', { min: MIN_AI_CONTEXT_WINDOW.toLocaleString(locale), max: MAX_AI_CONTEXT_WINDOW.toLocaleString(locale) }) }}</p>
            </FieldGroup>
          </section>
          <div v-if="!validContext || providerAutosave.error.value || providerAutosave.saving.value || providerAutosave.dirty.value || providerAutosave.saved.value" class="flex min-h-8 min-w-0 flex-wrap items-center gap-2 text-xs" data-ai-provider-save-status>
            <p v-if="!validContext" role="status" class="text-destructive">{{ t('aiSettings.contextNotSaved') }}</p>
            <template v-else-if="providerAutosave.error.value">
              <p role="alert" class="break-words text-destructive">{{ t('aiSettings.autoSaveFailed', { message: providerAutosave.error.value }) }}</p>
              <Button variant="outline" size="sm" data-ai-provider-retry @click="providerAutosave.flush()">{{ t('aiSettings.retry') }}</Button>
            </template>
            <p v-else-if="providerAutosave.saving.value || providerAutosave.dirty.value || providerAutosave.saved.value" role="status" class="text-muted-foreground">{{ providerAutosave.saving.value || providerAutosave.dirty.value ? t('common.saving') : t('settings.autoPersistSaved') }}</p>
          </div>
          <div class="flex justify-end">
            <SettingsHint :text="providerTestResult?.text || [t('aiSettings.providerHint'), t('aiSettings.autoSaveHint')].join('\n\n')">
              <Button
                variant="outline"
                size="sm"
                class="rounded-full data-[status=success]:border-success/35 data-[status=success]:bg-success/10 data-[status=success]:text-success data-[status=success]:hover:bg-success/20 data-[status=success]:hover:text-success dark:data-[status=success]:border-success/35 dark:data-[status=success]:bg-success/10 dark:data-[status=success]:hover:bg-success/20"
                :disabled="busy"
                :aria-busy="testingProvider"
                :aria-label="providerTestResult ? `${t('settings.experimentalTest')}: ${providerTestResult.text}` : undefined"
                aria-live="polite"
                :data-status="testingProvider ? 'testing' : providerTestResult ? (providerTestResult.ok ? 'success' : 'failed') : undefined"
                data-ai-provider-test
                @click="testProvider"
              >
                <Loader2 v-if="testingProvider" data-icon="inline-start" class="motion-safe:animate-spin" aria-hidden="true" />
                <Check v-else-if="providerTestResult?.ok" data-icon="inline-start" aria-hidden="true" />
                <X v-else-if="providerTestResult" data-icon="inline-start" aria-hidden="true" />
                {{ testingProvider ? t('settings.experimentalTestTesting') : providerTestResult ? t(providerTestResult.ok ? 'aiSettings.connectivityPassed' : 'aiSettings.connectivityFailed') : t('settings.experimentalTest') }}
              </Button>
            </SettingsHint>
          </div>
        </CardContent>
      </Card>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-statistics-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0"><span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><BarChart3 class="size-4" /></span><CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <SettingsHint :text="t('aiSettings.usageHint')">
              <span>{{ t('aiSettings.statistics') }}</span>
            </SettingsHint>
            <SettingsScopeBadge scope="server" />
          </CardTitle></CardHeader>
        <CardContent class="flex min-w-0 flex-col gap-4">
          <p v-if="!useWebApi" class="text-sm text-muted-foreground">{{ t('aiSettings.mockHint') }}</p>
          <section class="flex flex-wrap items-end justify-end gap-3" data-ai-settings-block="filters">
            <FieldGroup class="w-40">
              <Field><FieldLabel for="ai-days">{{ t('aiSettings.range') }}</FieldLabel><Select v-model="days"><SelectTrigger id="ai-days"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem value="today">{{ t('aiSettings.today') }}</SelectItem><SelectItem v-for="day in ['7','30','90']" :key="day" :value="day">{{ t('aiSettings.days', { count: day }) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
            </FieldGroup>
            <Button variant="ghost" size="icon" :disabled="loading" :aria-label="t('aiSettings.refresh')" data-ai-refresh @click="refresh"><RefreshCw /></Button>
          </section>
          <p v-if="loading" role="status" :class="{ 'sr-only': paging }">{{ t('aiSettings.loading') }}</p>
          <div v-if="summary" class="flex min-w-0 flex-col gap-4 pb-6" data-ai-summary>
            <p class="text-xs text-muted-foreground">{{ usagePeriod }}</p>
            <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1"><span class="text-4xl font-semibold tracking-tight tabular-nums sm:text-5xl" data-ai-token-total>{{ summary.modelCalls === 0 ? '0' : summary.usageCalls === 0 ? t('aiSettings.unknown') : summary.totalTokens.toLocaleString(locale) }}</span><span class="text-sm text-muted-foreground">Token</span></div>
            <dl class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
              <div class="flex items-center gap-2"><span class="size-2 rounded-full bg-primary" aria-hidden="true" /><dt class="text-muted-foreground">{{ t('aiSettings.inputTokens') }}</dt><dd class="tabular-nums">{{ summary.modelCalls === 0 ? '0' : measured(summary.promptTokens, summary.usageCalls, summary.modelCalls) }}</dd></div>
              <div class="flex items-center gap-2"><span class="size-2 rounded-full bg-chart-2" aria-hidden="true" /><dt class="text-muted-foreground">{{ t('aiSettings.outputTokens') }}</dt><dd class="tabular-nums">{{ summary.modelCalls === 0 ? '0' : measured(summary.completionTokens, summary.usageCalls, summary.modelCalls) }}</dd></div>
              <div class="flex gap-2"><dt class="text-muted-foreground">{{ t('aiSettings.tasks') }}</dt><dd class="tabular-nums">{{ summary.runs.toLocaleString(locale) }}</dd></div>
            </dl>
            <div v-if="summary.totalTokens > 0 && summary.usageCalls > 0" class="flex h-1 w-72 max-w-full gap-1 overflow-hidden rounded-full" aria-hidden="true"><span class="rounded-full bg-primary" :style="{ width: `${inputShare}%` }" /><span class="rounded-full bg-chart-2" :style="{ width: `${100 - inputShare}%` }" /></div>
            <p class="text-xs text-muted-foreground">{{ t('aiSettings.tokenExplanation') }}</p>
            <div class="flex flex-wrap items-center justify-between gap-3">
              <Button v-if="summary.failed > 0" variant="link" class="min-h-11 px-0 text-destructive" :disabled="loading" data-ai-view-failures @click="viewFailures"><AlertCircle data-icon="inline-start" />{{ t('aiSettings.failedTasks', { count: summary.failed }) }}</Button>
              <p class="text-xs text-muted-foreground" data-ai-usage-coverage>{{ t(summary.usageCalls < summary.modelCalls ? 'aiSettings.partialCoverage' : 'aiSettings.completeCoverage') }}</p>
            </div>
          </div>
          <section class="flex min-w-0 flex-col gap-3" :aria-busy="paging === 'runs'" data-ai-settings-block="recent-runs">
            <div class="flex flex-wrap items-end justify-between gap-4">
              <h3 ref="recordHeading" tabindex="-1" class="text-sm font-semibold text-foreground">{{ t('aiSettings.recentRuns') }}</h3>
              <FieldGroup class="grid w-full grid-cols-2 gap-3 sm:w-80 sm:shrink-0">
                <Field><FieldLabel for="ai-channel">{{ t('aiSettings.channel') }}</FieldLabel><Select v-model="channel"><SelectTrigger id="ai-channel" class="sm:w-36"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in ['all','chat','action','test','subtitle']" :key="item" :value="item">{{ t(`aiSettings.channels.${item}`) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
                <Field><FieldLabel for="ai-status">{{ t('aiSettings.status') }}</FieldLabel><Select v-model="status"><SelectTrigger id="ai-status" class="sm:w-36"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in ['all','completed','failed','partial','cancelled','needs_input','needs_confirmation']" :key="item" :value="item">{{ t(`aiSettings.statuses.${item}`) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
              </FieldGroup>
            </div>
            <p v-if="channel !== 'all' || status !== 'all'" class="text-xs text-muted-foreground">{{ t('aiSettings.recordFiltersHint') }}</p>
            <p v-if="pageError?.kind === 'runs'" role="alert" class="text-sm text-destructive">{{ pageError.message }}</p>
            <p v-if="report && !report.total" class="text-sm text-muted-foreground">{{ t('aiSettings.empty') }}</p>
            <ul ref="runsList" class="flex min-w-0 flex-col" :style="{ minHeight: runsMinHeight ? `${runsMinHeight}px` : undefined }">
              <li v-for="run in report?.items ?? []" :key="run.id" class="min-h-9 min-w-0 border-t border-border/50" :data-ai-run-row="run.id" :data-status="run.status">
                <Collapsible v-model:open="expandedRuns[run.id]">
                  <CollapsibleTrigger as-child><button type="button" :aria-label="runLine(run)" class="group flex min-h-11 w-full items-center gap-2 rounded-lg px-2 py-2 sm:min-h-10 sm:py-1.5 text-left hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                    <span class="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-3"><span class="shrink-0 text-sm font-medium">{{ t(`aiSettings.channels.${run.channel}`) }}</span><span class="truncate text-xs text-muted-foreground">{{ time(run.startedAt) }} · {{ run.model || t('aiSettings.unknown') }}</span></span>
                    <span class="flex shrink-0 flex-col items-end gap-1 sm:flex-row sm:items-center sm:gap-5"><span class="text-right text-sm tabular-nums">{{ measured(run.totalTokens, run.usageCalls, run.modelCalls) }} <span class="text-xs text-muted-foreground">Token</span></span><span class="hidden text-xs text-muted-foreground sm:block">{{ duration(run.durationMs) }}</span><Badge :variant="runStatusTone(run.status)">{{ t(`aiSettings.statuses.${run.status}`) }}</Badge></span>
                    <ChevronRight class="size-4 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90" aria-hidden="true" />
                  </button></CollapsibleTrigger>
                  <CollapsibleContent>
                    <div class="mb-5 ml-2 flex min-w-0 flex-col gap-4 border-l-2 border-border/60 pl-4 text-sm" data-ai-run-detail>
                      <p v-if="run.status === 'failed' || run.status === 'partial'" class="text-destructive">{{ runError(run) }}</p>
                      <p v-if="run.usageCalls < run.modelCalls" class="text-xs text-muted-foreground">{{ t('aiSettings.partialCoverage') }}</p>
                      <dl class="grid grid-cols-2 gap-4 sm:grid-cols-3">
                        <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.inputTokens') }}</dt><dd class="mt-1 tabular-nums">{{ measured(run.promptTokens, run.usageCalls, run.modelCalls) }}</dd></div>
                        <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.outputTokens') }}</dt><dd class="mt-1 tabular-nums">{{ measured(run.completionTokens, run.usageCalls, run.modelCalls) }}</dd></div>
                        <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.taskDuration') }}</dt><dd class="mt-1">{{ duration(run.durationMs) }}</dd></div>
                        <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.modelCalls') }}</dt><dd class="mt-1 tabular-nums">{{ run.modelCalls }}</dd></div>
                        <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.toolCalls') }}</dt><dd class="mt-1 tabular-nums">{{ run.toolCalls }}</dd></div>
                        <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.firstTextSingle') }}</dt><dd class="mt-1">{{ duration(run.firstTextMs) }}</dd></div>
                      </dl>
                      <details v-if="run.errorCode"><summary class="min-h-11 cursor-pointer py-3 text-xs text-muted-foreground">{{ t('aiSettings.technicalDetails') }}</summary><p class="break-all font-mono text-xs">{{ run.errorCode }}</p></details>
                    </div>
                  </CollapsibleContent>
                </Collapsible>
              </li>
            </ul>
            <div class="flex flex-wrap items-center justify-end gap-x-4 gap-y-2" data-ai-runs-pagination>
              <span v-if="report" class="mr-auto text-xs text-muted-foreground">
                {{ report.total ? offset + 1 : 0 }}–{{ Math.min(offset + report.limit, report.total) }} / {{ report.total }}
              </span>
              <div class="flex flex-wrap items-center justify-end gap-2">
                <FieldGroup class="w-auto shrink-0 [container-type:normal]">
                  <Field orientation="horizontal" class="items-center gap-2">
                    <FieldLabel for="ai-runs-page-size" class="whitespace-nowrap text-xs text-muted-foreground">{{ t('aiSettings.rowsPerPage') }}</FieldLabel>
                    <Select v-model="runsPageSize" :disabled="loading">
                      <SelectTrigger id="ai-runs-page-size" class="w-20 shrink-0"><SelectValue /></SelectTrigger>
                      <SelectContent><SelectGroup><SelectItem v-for="size in ['20','50','100']" :key="size" :value="size">{{ size }}</SelectItem></SelectGroup></SelectContent>
                    </Select>
                  </Field>
                </FieldGroup>
                <div class="flex shrink-0 items-center gap-2">
                  <Button variant="outline" size="sm" :disabled="loading || !report || offset === 0" @click="page('runs', -1)">{{ t('aiSettings.previous') }}</Button>
                  <Button variant="outline" size="sm" :disabled="loading || !report || offset + Number(runsPageSize) >= report.total" @click="page('runs', 1)">{{ t('aiSettings.next') }}</Button>
                </div>
              </div>
            </div>
          </section>
          <details class="mt-2" data-ai-more-statistics><summary class="min-h-11 cursor-pointer py-3 text-sm text-muted-foreground">{{ t('aiSettings.moreStatistics') }}</summary>
          <dl v-if="summary" class="my-4 flex flex-wrap gap-6 text-sm">
            <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.modelCalls') }}</dt><dd class="mt-1 tabular-nums">{{ summary.modelCalls }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.firstText') }}</dt><dd class="mt-1">{{ duration(summary.avgFirstTextMs) }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.duration') }}</dt><dd class="mt-1">{{ duration(summary.avgDurationMs) }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.toolCalls') }}</dt><dd class="mt-1">{{ summary.toolCalls }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.coverage') }}</dt><dd class="mt-1">{{ summary.usageCalls }} / {{ summary.modelCalls }}</dd></div>
            <div><dt class="text-xs text-muted-foreground">{{ t('aiSettings.failed') }}</dt><dd class="mt-1">{{ summary.failed }} / {{ summary.partial }} / {{ summary.cancelled }}</dd></div>
          </dl>
          <p class="mb-5 text-xs text-muted-foreground">{{ t('aiSettings.timingHint') }}</p>
          <section class="flex min-w-0 flex-col gap-3" :aria-busy="paging === 'audit'" data-ai-settings-block="audit">
            <SettingsHint :text="t('aiSettings.auditHint')">
              <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.audit') }}</h3>
            </SettingsHint>
            <p v-if="pageError?.kind === 'audit'" role="alert" class="text-sm text-destructive">{{ pageError.message }}</p>
            <p v-if="audit && !audit.total" class="text-sm text-muted-foreground">{{ t('aiSettings.empty') }}</p>
            <ul ref="auditList" class="flex min-w-0 flex-col" :style="{ minHeight: auditMinHeight ? `${auditMinHeight}px` : undefined }">
              <li v-for="entry in audit?.items ?? []" :key="entry.id" :title="auditLine(entry)" class="flex min-h-11 min-w-0 items-center gap-2 border-t border-border/50 px-2 py-2 sm:min-h-10 sm:py-1.5" :data-ai-audit-row="entry.id" :data-status="entry.result">
                <span class="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-3">
                  <span class="shrink-0 text-sm font-medium">{{ toolLabel(entry.tool) }}</span>
                  <span class="truncate text-xs text-muted-foreground">{{ time(entry.createdAt) }} · {{ entry.permission }}<template v-if="entry.errorCode"> · {{ entry.errorCode }}</template></span>
                </span>
                <span class="flex shrink-0 items-center gap-5">
                  <span class="hidden text-xs text-muted-foreground sm:block">{{ duration(entry.durationMs) }}</span>
                  <Badge :variant="auditStatusTone(entry.result)">{{ t(`aiSettings.auditResults.${entry.result}`) }}</Badge>
                </span>
              </li>
            </ul>
            <div class="flex flex-wrap items-center justify-end gap-x-4 gap-y-2" data-ai-audit-pagination>
              <span v-if="audit" class="mr-auto text-xs text-muted-foreground">
                {{ audit.total ? auditOffset + 1 : 0 }}–{{ Math.min(auditOffset + audit.limit, audit.total) }} / {{ audit.total }}
              </span>
              <div class="flex flex-wrap items-center justify-end gap-2">
                <FieldGroup class="w-auto shrink-0 [container-type:normal]">
                  <Field orientation="horizontal" class="items-center gap-2">
                    <FieldLabel for="ai-audit-page-size" class="whitespace-nowrap text-xs text-muted-foreground">{{ t('aiSettings.rowsPerPage') }}</FieldLabel>
                    <Select v-model="auditPageSize" :disabled="loading">
                      <SelectTrigger id="ai-audit-page-size" class="w-20 shrink-0"><SelectValue /></SelectTrigger>
                      <SelectContent><SelectGroup><SelectItem v-for="size in ['20','50','100']" :key="size" :value="size">{{ size }}</SelectItem></SelectGroup></SelectContent>
                    </Select>
                  </Field>
                </FieldGroup>
                <div class="flex shrink-0 items-center gap-2">
                  <Button variant="outline" size="sm" :disabled="loading || !audit || auditOffset === 0" @click="page('audit', -1)">{{ t('aiSettings.previous') }}</Button>
                  <Button variant="outline" size="sm" :disabled="loading || !audit || auditOffset + Number(auditPageSize) >= audit.total" @click="page('audit', 1)">{{ t('aiSettings.next') }}</Button>
                </div>
              </div>
            </div>
          </section>
          </details>
        </CardContent>
      </Card>
      <SettingsSubtitlesSection />
      </div>
    </template>
  </div>
</template>
