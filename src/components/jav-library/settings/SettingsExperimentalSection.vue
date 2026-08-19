<script setup lang="ts">
import { onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { FlaskConical, Loader2 } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useLibraryService } from "@/services/library-service"

type StatusMessage = { tone: "ok" | "fail"; text: string } | null

defineProps<{
  useWebApi: boolean
}>()

const { t } = useI18n()
const { enabled, setEnabled } = useExperimentalAgent()
const libraryService = useLibraryService()

const baseUrlDraft = ref("")
const apiKeyDraft = ref("")
const modelDraft = ref("")
const saving = ref(false)
const testing = ref(false)
const status = ref<StatusMessage>(null)

function syncDraftsFromService() {
  const provider = libraryService.aiProvider.value
  baseUrlDraft.value = provider.baseUrl
  apiKeyDraft.value = provider.apiKey ?? ""
  modelDraft.value = provider.model
}

onMounted(syncDraftsFromService)
watch(() => libraryService.aiProvider.value, syncDraftsFromService)

function draftProvider() {
  return {
    kind: "openai-compatible",
    baseUrl: baseUrlDraft.value.trim(),
    apiKey: apiKeyDraft.value,
    model: modelDraft.value.trim(),
  }
}

async function saveProvider() {
  if (saving.value) return
  saving.value = true
  status.value = null
  try {
    await libraryService.setAIProvider({
      baseUrl: baseUrlDraft.value.trim(),
      apiKey: apiKeyDraft.value,
      model: modelDraft.value.trim(),
    })
    status.value = { tone: "ok", text: t("settings.experimentalSaved") }
  } catch (err) {
    status.value = {
      tone: "fail",
      text: t("settings.experimentalSaveFailed", { message: (err as Error).message }),
    }
  } finally {
    saving.value = false
  }
}

async function testProvider() {
  if (testing.value) return
  testing.value = true
  status.value = null
  try {
    const result = await libraryService.testAIProvider(draftProvider())
    status.value = result.ok
      ? { tone: "ok", text: t("settings.experimentalTestOk", { ms: result.latencyMs }) }
      : {
          tone: "fail",
          text: t("settings.experimentalTestFail", { message: result.message ?? "" }),
        }
  } catch (err) {
    status.value = {
      tone: "fail",
      text: t("settings.experimentalTestFail", { message: (err as Error).message }),
    }
  } finally {
    testing.value = false
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
            <FlaskConical class="size-[1.15rem]" />
          </span>
          <CardTitle class="min-w-0 text-lg tracking-tight">
            {{ t("settings.experimentalTitle") }}
          </CardTitle>
          <CardDescription
            class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
          >
            {{ t("settings.experimentalDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <div class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5">
            <div class="min-w-0 flex-1 space-y-1">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.experimentalAgentToggle") }}
              </p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.experimentalAgentHint") }}
              </p>
            </div>
            <Switch
              class="motion-safe:transition-colors motion-safe:duration-200"
              :model-value="enabled"
              data-experimental-agent-toggle
              @update:model-value="setEnabled(Boolean($event))"
            />
          </div>

          <div
            v-if="enabled"
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          >
            <div class="space-y-1">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.experimentalProviderTitle") }}
              </p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.experimentalProviderDesc") }}
              </p>
            </div>
            <div class="flex flex-col gap-3">
              <p class="text-sm font-medium text-foreground">{{ t("settings.experimentalBaseUrl") }}</p>
              <Input
                v-model="baseUrlDraft"
                autocomplete="off"
                spellcheck="false"
                class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
                :placeholder="t('settings.experimentalBaseUrlPlaceholder')"
                :disabled="saving"
                data-experimental-base-url
              />
            </div>
            <div class="flex flex-col gap-3">
              <p class="text-sm font-medium text-foreground">{{ t("settings.experimentalApiKey") }}</p>
              <Input
                v-model="apiKeyDraft"
                type="password"
                autocomplete="new-password"
                class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
                :placeholder="t('settings.experimentalApiKeyPlaceholder')"
                :disabled="saving"
                data-experimental-api-key
              />
            </div>
            <div class="flex flex-col gap-3">
              <p class="text-sm font-medium text-foreground">{{ t("settings.experimentalModel") }}</p>
              <Input
                v-model="modelDraft"
                autocomplete="off"
                spellcheck="false"
                class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50"
                :placeholder="t('settings.experimentalModelPlaceholder')"
                :disabled="saving"
                data-experimental-model
              />
            </div>
            <p
              v-if="!useWebApi"
              class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-xs leading-relaxed text-muted-foreground sm:text-sm"
            >
              {{ t("settings.experimentalMockHint") }}
            </p>
            <div class="flex flex-wrap items-center justify-end gap-3">
              <Button
                type="button"
                variant="outline"
                class="rounded-full"
                :disabled="saving || testing"
                :aria-busy="testing"
                data-experimental-test
                @click="testProvider"
              >
                <Loader2
                  v-if="testing"
                  class="mr-2 size-4 motion-safe:animate-spin"
                  aria-hidden="true"
                />
                {{
                  testing
                    ? t("settings.experimentalTestTesting")
                    : t("settings.experimentalTest")
                }}
              </Button>
              <Button
                type="button"
                class="rounded-full"
                :disabled="saving || testing"
                :aria-busy="saving"
                data-experimental-save
                @click="saveProvider"
              >
                <Loader2
                  v-if="saving"
                  class="mr-2 size-4 motion-safe:animate-spin"
                  aria-hidden="true"
                />
                {{ saving ? t("common.saving") : t("settings.experimentalSave") }}
              </Button>
            </div>
            <p
              class="min-h-5 text-sm transition-colors"
              :class="
                status === null
                  ? 'text-transparent'
                  : status.tone === 'ok'
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : 'text-destructive'
              "
              data-experimental-status
            >
              {{ status?.text ?? " " }}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
