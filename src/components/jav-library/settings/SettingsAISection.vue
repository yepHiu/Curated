<script setup lang="ts">
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
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
import { defaultAIGovernance, type AIReport, type AIPage, type AIAuditEntry, type AIRun } from "@/services/contracts/ai-governance-service"
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
const days = ref("30")
const channel = ref("all")
const status = ref("all")
const RUNS_PAGE_SIZE = 5
const AUDIT_PAGE_SIZE = 10
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
const summary = computed(() => report.value?.summary)
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
onBeforeUnmount(() => { disposed = true; requestId++ })

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
  const query = { days: Number(days.value), channel: channel.value === "all" ? undefined : channel.value }
  try {
    const [nextReport, nextAudit] = await Promise.all([
      service.getUsage({ ...query, status: status.value === "all" ? undefined : status.value, offset: offset.value, limit: RUNS_PAGE_SIZE }),
      service.getAudit({ ...query, status: status.value === "failed" ? "failed" : undefined, offset: auditOffset.value, limit: AUDIT_PAGE_SIZE }),
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
function duration(value: number | null | undefined) { return value == null ? t("aiSettings.unknown") : t("aiSettings.milliseconds", { value: Math.round(value).toLocaleString(locale.value) }) }
function time(value: string) { return new Date(value).toLocaleString(locale.value) }
function measured(total: number, known: number, calls: number) {
  if (known === 0) return t("aiSettings.unknown")
  return total.toLocaleString(locale.value) + (known < calls ? ` (${t("aiSettings.partialUsage")})` : "")
}
function toolLabel(name: string) { const key = AGENT_TOOL_I18N_KEYS[name]; return key ? t(key) : name }
async function page(kind: "runs" | "audit", direction: -1 | 1) {
  if (loading.value) return
  const pageSize = kind === "runs" ? RUNS_PAGE_SIZE : AUDIT_PAGE_SIZE
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
  const query = { days: Number(days.value), channel: channel.value === "all" ? undefined : channel.value, offset: nextOffset, limit: pageSize }
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
    <Button v-if="!ready" variant="outline" size="sm" class="self-start" @click="initialize">{{ t('aiSettings.retry') }}</Button>
    <template v-if="ready">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-governance-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
          <span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><ShieldCheck class="size-4" /></span>
          <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t('aiSettings.governance') }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle>
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
              <Field orientation="horizontal" class="rounded-lg border border-border/40 bg-background/30 px-3 py-2">
                <FieldLabel for="ai-step-limit-enabled">{{ t('aiSettings.stepLimitEnabled') }}</FieldLabel>
                <Switch id="ai-step-limit-enabled" v-model="stepLimitEnabled" :disabled="busy" aria-describedby="ai-step-limit-hint" />
              </Field>
              <FieldDescription id="ai-step-limit-hint">{{ t('aiSettings.stepLimitHint') }}</FieldDescription>
              <FieldGroup class="grid grid-cols-[repeat(auto-fit,minmax(min(100%,20rem),1fr))] gap-3" data-ai-limit-fields>
                <Field v-if="stepLimitEnabled" :data-invalid="!validSettings"><FieldLabel for="ai-steps">{{ t('aiSettings.stepLimit') }}</FieldLabel><Input id="ai-steps" v-model.number="settings.stepLimit" :aria-invalid="!Number.isInteger(settings.stepLimit) || settings.stepLimit < 1 || settings.stepLimit > 30" @blur="settingsAutosave.flush()" type="number" min="1" max="30" :disabled="busy" /></Field>
                <Field><FieldLabel for="ai-rate">{{ t('aiSettings.writeLimit') }}</FieldLabel><Input id="ai-rate" v-model.number="settings.writePerMinute" :aria-invalid="!Number.isInteger(settings.writePerMinute) || settings.writePerMinute < 1 || settings.writePerMinute > 60" @blur="settingsAutosave.flush()" type="number" min="1" max="60" :disabled="busy" /></Field>
                <Field><FieldLabel for="ai-retention">{{ t('aiSettings.retention') }}</FieldLabel><Input id="ai-retention" v-model.number="settings.retentionDays" :aria-invalid="!Number.isInteger(settings.retentionDays) || settings.retentionDays < 7 || settings.retentionDays > 365" @blur="settingsAutosave.flush()" type="number" min="7" max="365" :disabled="busy" /></Field>
              </FieldGroup>
              <FieldDescription>{{ t('aiSettings.retentionHint') }}</FieldDescription>
            </FieldGroup>
          </section>
          <div class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border/40 bg-background/30 p-3">
            <div class="flex min-h-8 min-w-0 flex-1 flex-wrap items-center gap-2 text-xs" data-ai-policy-save-status>
              <p v-if="!validSettings" role="alert" class="text-destructive">{{ t('aiSettings.invalidLimits') }}</p>
              <template v-else-if="settingsAutosave.error.value">
                <p role="alert" class="break-words text-destructive">{{ t('aiSettings.autoSaveFailed', { message: settingsAutosave.error.value }) }}</p>
                <Button variant="outline" size="sm" data-ai-policy-retry @click="settingsAutosave.flush()">{{ t('aiSettings.retry') }}</Button>
              </template>
              <p v-else role="status" class="text-muted-foreground">{{ settingsAutosave.saving.value || settingsAutosave.dirty.value ? t('common.saving') : settingsAutosave.saved.value ? t('settings.autoPersistSaved') : t('aiSettings.autoSaveHint') }}</p>
            </div>
            <Button variant="outline" size="sm" :disabled="busy" data-ai-cleanup @click="cleanup">{{ t('aiSettings.cleanup') }}</Button>
          </div>
        </CardContent>
      </Card>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-provider-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0"><span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><Server class="size-4" /></span><CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t('aiSettings.provider') }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle></CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <section class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <FieldGroup class="grid grid-cols-1 gap-3 md:grid-cols-2">
              <Field class="md:col-span-2"><FieldLabel for="ai-base">{{ t('settings.experimentalBaseUrl') }}</FieldLabel><Input id="ai-base" v-model="baseUrl" @blur="providerAutosave.flush()" autocomplete="off" :disabled="busy" /></Field>
              <Field><FieldLabel for="ai-key">{{ t('settings.experimentalApiKey') }}</FieldLabel><Input id="ai-key" v-model="apiKey" @blur="providerAutosave.flush()" type="password" autocomplete="new-password" :disabled="busy" /></Field>
              <Field><FieldLabel for="ai-model">{{ t('settings.experimentalModel') }}</FieldLabel><Input id="ai-model" v-model="model" @blur="providerAutosave.flush()" autocomplete="off" :disabled="busy" /></Field>
              <Field :data-invalid="!validContext || undefined">
                <FieldLabel for="ai-context-window">{{ t('aiSettings.contextWindow') }}</FieldLabel>
                <Input id="ai-context-window" v-model="contextWindow" type="number" inputmode="numeric" :min="MIN_AI_CONTEXT_WINDOW" :max="MAX_AI_CONTEXT_WINDOW" :step="1" :disabled="busy" :aria-invalid="!validContext" :aria-describedby="validContext ? 'ai-context-help' : 'ai-context-help ai-context-validation'" @blur="providerAutosave.flush()" />
              </Field>
              <Field>
                <FieldLabel for="ai-context-preset">{{ t('aiSettings.contextPreset') }}</FieldLabel>
                <Select :model-value="contextPreset" :disabled="busy" @update:model-value="applyContextPreset">
                  <SelectTrigger id="ai-context-preset"><SelectValue /></SelectTrigger>
                  <SelectContent><SelectGroup>
                    <SelectItem value="custom">{{ t('aiSettings.contextCustom') }}</SelectItem>
                    <SelectItem v-for="preset in AI_CONTEXT_PRESETS" :key="preset.id" :value="preset.id">{{ preset.label }} · {{ preset.tokens.toLocaleString(locale) }}</SelectItem>
                  </SelectGroup></SelectContent>
                </Select>
              </Field>
              <FieldDescription id="ai-context-help" class="md:col-span-2">
                {{ t('aiSettings.contextHint') }}
                <a v-if="selectedContextPreset" :href="selectedContextPreset.source" target="_blank" rel="noopener noreferrer" class="underline underline-offset-4">{{ t('aiSettings.contextSource') }}</a>
              </FieldDescription>
              <p v-if="!validContext" id="ai-context-validation" role="alert" class="text-sm text-destructive md:col-span-2">{{ t('aiSettings.contextInvalid', { min: MIN_AI_CONTEXT_WINDOW.toLocaleString(locale), max: MAX_AI_CONTEXT_WINDOW.toLocaleString(locale) }) }}</p>
            </FieldGroup>
          </section>
          <div class="flex min-h-8 min-w-0 flex-wrap items-center gap-2 text-xs" data-ai-provider-save-status>
            <p v-if="!validContext" role="status" class="text-destructive">{{ t('aiSettings.contextNotSaved') }}</p>
            <template v-else-if="providerAutosave.error.value">
              <p role="alert" class="break-words text-destructive">{{ t('aiSettings.autoSaveFailed', { message: providerAutosave.error.value }) }}</p>
              <Button variant="outline" size="sm" data-ai-provider-retry @click="providerAutosave.flush()">{{ t('aiSettings.retry') }}</Button>
            </template>
            <p v-else role="status" class="text-muted-foreground">{{ providerAutosave.saving.value || providerAutosave.dirty.value ? t('common.saving') : providerAutosave.saved.value ? t('settings.autoPersistSaved') : t('aiSettings.autoSaveHint') }}</p>
          </div>
          <div class="flex flex-col gap-3 rounded-lg border border-border/40 bg-background/30 p-3"><div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><FieldDescription>{{ t('aiSettings.providerHint') }}</FieldDescription><div class="flex shrink-0 flex-wrap gap-2"><Button variant="outline" size="sm" :disabled="busy" data-ai-provider-test @click="testProvider">{{ testingProvider ? t('settings.experimentalTestTesting') : t('settings.experimentalTest') }}</Button></div></div><p v-if="providerTestResult" :role="providerTestResult.ok ? 'status' : 'alert'" aria-live="polite" :class="['rounded-md border px-3 py-2 text-sm', providerTestResult.ok ? 'border-success/30 bg-success/10 text-success' : 'border-destructive/30 bg-destructive/10 text-destructive']" :data-status="providerTestResult.ok ? 'success' : 'failed'" data-ai-provider-test-result>{{ providerTestResult.text }}</p></div>
        </CardContent>
      </Card>
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-ai-statistics-card>
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0"><span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><BarChart3 class="size-4" /></span><CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t('aiSettings.statistics') }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle></CardHeader>
        <CardContent class="flex min-w-0 flex-col gap-4">
          <p v-if="!useWebApi" class="text-sm text-muted-foreground">{{ t('aiSettings.mockHint') }}</p>
          <section class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" data-ai-settings-block="filters">
            <FieldGroup class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <Field><FieldLabel for="ai-days">{{ t('aiSettings.range') }}</FieldLabel><Select v-model="days"><SelectTrigger id="ai-days"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="day in ['7','30','90']" :key="day" :value="day">{{ t('aiSettings.days', { count: day }) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
              <Field><FieldLabel for="ai-channel">{{ t('aiSettings.channel') }}</FieldLabel><Select v-model="channel"><SelectTrigger id="ai-channel"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in ['all','chat','action','test']" :key="item" :value="item">{{ t(`aiSettings.channels.${item}`) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
              <Field><FieldLabel for="ai-status">{{ t('aiSettings.status') }}</FieldLabel><Select v-model="status"><SelectTrigger id="ai-status"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in ['all','completed','failed','partial','cancelled','needs_input','needs_confirmation']" :key="item" :value="item">{{ t(`aiSettings.statuses.${item}`) }}</SelectItem></SelectGroup></SelectContent></Select></Field>
            </FieldGroup>
            <Button variant="outline" size="sm" :disabled="loading" class="self-end" data-ai-refresh @click="refresh"><RefreshCw data-icon="inline-start" />{{ t('aiSettings.refresh') }}</Button>
          </section>
          <p v-if="loading" role="status" :class="{ 'sr-only': paging }">{{ t('aiSettings.loading') }}</p>
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
          <section class="flex min-w-0 flex-col gap-3" :aria-busy="paging === 'runs'" data-ai-settings-block="recent-runs">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.recentRuns') }}</h3>
            <p v-if="pageError?.kind === 'runs'" role="alert" class="text-sm text-destructive">{{ pageError.message }}</p>
            <p v-if="report && !report.total" class="text-sm text-muted-foreground">{{ t('aiSettings.empty') }}</p>
            <ul ref="runsList" class="flex min-w-0 flex-col gap-1.5" :style="{ minHeight: runsMinHeight ? `${runsMinHeight}px` : undefined }">
              <li v-for="run in report?.items ?? []" :key="run.id" :title="runLine(run)" class="flex min-h-9 min-w-0 items-center gap-2 rounded-lg border border-border/50 bg-muted/5 px-3 py-1.5" :data-ai-run-row="run.id" :data-status="run.status">
                <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ run.model || t('aiSettings.unknown') }}</span>
                <span class="hidden min-w-0 truncate text-xs text-muted-foreground md:block">{{ time(run.startedAt) }} · {{ t(`aiSettings.channels.${run.channel}`) }} · {{ duration(run.durationMs) }} · {{ t('aiSettings.totalTokens') }}: {{ measured(run.totalTokens, run.usageCalls, run.modelCalls) }} · {{ t('aiSettings.toolCalls') }}: {{ run.toolCalls }}<template v-if="run.errorCode"> · {{ run.errorCode }}</template></span>
                <Badge :variant="runStatusTone(run.status)">{{ t(`aiSettings.statuses.${run.status}`) }}</Badge>
              </li>
            </ul>
            <div v-if="report && report.total > report.limit" class="flex flex-wrap items-center justify-end gap-2"><span class="mr-auto text-xs text-muted-foreground">{{ offset + 1 }}–{{ Math.min(offset + report.limit, report.total) }} / {{ report.total }}</span><Button variant="outline" size="sm" :disabled="loading || offset === 0" @click="page('runs', -1)">{{ t('aiSettings.previous') }}</Button><Button variant="outline" size="sm" :disabled="loading || offset + RUNS_PAGE_SIZE >= report.total" @click="page('runs', 1)">{{ t('aiSettings.next') }}</Button></div>
          </section>
          <section class="flex min-w-0 flex-col gap-3" :aria-busy="paging === 'audit'" data-ai-settings-block="audit">
            <h3 class="text-sm font-semibold text-foreground">{{ t('aiSettings.audit') }}</h3>
            <p v-if="pageError?.kind === 'audit'" role="alert" class="text-sm text-destructive">{{ pageError.message }}</p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t('aiSettings.auditHint') }}</p>
            <p v-if="audit && !audit.total" class="text-sm text-muted-foreground">{{ t('aiSettings.empty') }}</p>
            <ul ref="auditList" class="flex min-w-0 flex-col gap-1.5" :style="{ minHeight: auditMinHeight ? `${auditMinHeight}px` : undefined }"><li v-for="entry in audit?.items ?? []" :key="entry.id" :title="auditLine(entry)" class="flex min-h-9 min-w-0 items-center gap-2 rounded-lg border border-border/50 bg-muted/5 px-3 py-1.5" :data-ai-audit-row="entry.id" :data-status="entry.result"><span class="min-w-0 flex-1 truncate text-sm font-medium">{{ toolLabel(entry.tool) }}</span><span class="hidden min-w-0 truncate text-xs text-muted-foreground md:block">{{ time(entry.createdAt) }} · {{ entry.permission }} · {{ duration(entry.durationMs) }}<template v-if="entry.errorCode"> · {{ entry.errorCode }}</template></span><Badge :variant="auditStatusTone(entry.result)">{{ t(`aiSettings.auditResults.${entry.result}`) }}</Badge></li></ul>
            <div v-if="audit && audit.total > audit.limit" class="flex flex-wrap justify-end gap-2"><Button variant="outline" size="sm" :disabled="loading || auditOffset === 0" @click="page('audit', -1)">{{ t('aiSettings.previous') }}</Button><Button variant="outline" size="sm" :disabled="loading || auditOffset + AUDIT_PAGE_SIZE >= audit.total" @click="page('audit', 1)">{{ t('aiSettings.next') }}</Button></div>
          </section>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
