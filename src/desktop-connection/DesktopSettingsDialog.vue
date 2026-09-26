<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { LoaderCircle } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import type { DesktopSettingsAPI, DesktopSettingsInput } from "./settings-contract"

const props = defineProps<{ open: boolean; api: DesktopSettingsAPI }>()
const emit = defineEmits<{ "update:open": [open: boolean] }>()
const { t } = useI18n()
const draft = ref<DesktopSettingsInput>({ proxyMode: "system", proxyUrl: "", launchAtLogin: false })
const loading = ref(false)
const saving = ref(false)
const ready = ref(false)
const loginSupported = ref(false)
const restartRequired = ref(false)
const error = ref("")
const saved = ref(false)
let loadID = 0

// 修改已保存的表单后移除成功提示，避免误以为新草稿已生效。
watch(draft, () => { saved.value = false }, { deep: true, flush: "sync" })

// 每次打开读取主进程与 OS 状态，不把取消的草稿当成已保存设置。
watch(() => props.open, async open => {
  const id = ++loadID
  if (!open) return
  loading.value = true; ready.value = false; error.value = ""; saved.value = false
  try {
    const state = await props.api.readSettings()
    if (id !== loadID) return
    draft.value = { proxyMode: state.proxyMode, proxyUrl: state.proxyUrl, launchAtLogin: state.launchAtLogin }
    loginSupported.value = state.loginSupported
    restartRequired.value = state.restartRequired
    ready.value = true
  } catch (reason) { if (id === loadID) error.value = String(reason) }
  finally { if (id === loadID) loading.value = false }
}, { immediate: true })

/** 保存失败保留草稿；成功用主进程返回的规范化值更新表单。 */
async function save() {
  if (!ready.value || saving.value) return
  saving.value = true; error.value = ""; saved.value = false
  try {
    const state = await props.api.saveSettings({ ...draft.value })
    draft.value = { proxyMode: state.proxyMode, proxyUrl: state.proxyUrl, launchAtLogin: state.launchAtLogin }
    restartRequired.value = state.restartRequired
    saved.value = true
  } catch (reason) { error.value = String(reason) }
  finally { saving.value = false }
}

/** 保存期间保持弹窗，避免操作结果落到已关闭页面。 */
function setOpen(open: boolean) { if (!saving.value) emit("update:open", open) }
</script>

<template>
  <Dialog :open="open" @update:open="setOpen">
    <DialogContent class="max-h-[calc(100dvh-6rem)] overflow-y-auto" :show-close-button="!saving">
      <DialogHeader>
        <DialogTitle>{{ t('deviceSettings') }}</DialogTitle>
        <DialogDescription>{{ t('deviceSettingsScope') }}</DialogDescription>
      </DialogHeader>
      <div v-if="loading" role="status" class="flex items-center gap-2 py-4 text-sm text-muted-foreground"><LoaderCircle class="size-4 motion-safe:animate-spin" aria-hidden="true" />{{ t('settingsLoading') }}</div>
      <form v-else class="flex flex-col gap-5" @submit.prevent="save">
        <fieldset :disabled="!ready || saving" class="flex min-w-0 flex-col gap-5">
          <legend class="sr-only">{{ t('deviceSettings') }}</legend>
          <Field class="gap-2">
            <FieldLabel for="desktop-proxy-mode">{{ t('proxyMode') }}</FieldLabel>
            <Select v-model="draft.proxyMode" :disabled="!ready || saving">
              <SelectTrigger id="desktop-proxy-mode" class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="system">{{ t('proxySystem') }}</SelectItem><SelectItem value="direct">{{ t('proxyDirect') }}</SelectItem><SelectItem value="manual">{{ t('proxyManual') }}</SelectItem></SelectContent>
            </Select>
          </Field>
          <Field v-if="draft.proxyMode === 'manual'" class="gap-2">
            <FieldLabel for="desktop-proxy-url">{{ t('proxyAddress') }}</FieldLabel>
            <Input id="desktop-proxy-url" v-model="draft.proxyUrl" placeholder="http://127.0.0.1:7890" :spellcheck="false" autocomplete="off" required />
            <p class="text-xs leading-relaxed text-muted-foreground">{{ t('proxyHint') }}</p>
          </Field>
          <div class="flex items-center justify-between gap-3">
            <div class="flex min-w-0 flex-col gap-1"><label for="desktop-login" class="text-sm font-medium">{{ t('launchAtLogin') }}</label><p class="text-xs text-muted-foreground">{{ loginSupported ? t('loginHint') : t('loginUnsupported') }}</p></div>
            <Switch id="desktop-login" v-model="draft.launchAtLogin" :disabled="!loginSupported || !ready || saving" />
          </div>
        </fieldset>
        <p v-if="error" role="alert" class="break-words text-sm text-destructive">{{ error }}</p>
        <p v-if="saved || restartRequired" role="status" class="text-xs leading-relaxed text-muted-foreground">{{ restartRequired ? t('proxyRestart') : t('settingsSaved') }}</p>
        <DialogFooter class="flex-row justify-end gap-2">
          <Button type="button" variant="outline" class="h-[29px] rounded-full py-0" :disabled="saving" @click="setOpen(false)">{{ t('close') }}</Button>
          <Button type="submit" class="h-[29px] rounded-full py-0" :disabled="!ready || saving"><LoaderCircle v-if="saving" class="motion-safe:animate-spin" aria-hidden="true" />{{ saving ? t('settingsSaving') : t('save') }}</Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
