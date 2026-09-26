<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { LockKeyhole, ShieldCheck } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { AuthSessionDTO } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Skeleton } from "@/components/ui/skeleton"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"

const { t } = useI18n()

const pinDraft = ref("")
const confirmPinDraft = ref("")
const currentPinDraft = ref("")
const newPinDraft = ref("")
const confirmNewPinDraft = ref("")
const setupBusy = ref(false)
const changeBusy = ref(false)
const setupDialogOpen = ref(false)
const changeDialogOpen = ref(false)
const disableDialogOpen = ref(false)
const settingsBusy = ref(false)
const lockBusy = ref(false)
const trustedSessionsLoading = ref(false)
const trustedSessionActionBusy = ref(false)
const trustedSessionDialogOpen = ref(false)
const trustedSessionAction = ref<{ kind: "one", session: AuthSessionDTO } | { kind: "others" } | null>(null)
const errorText = ref("")
const successText = ref("")

const status = computed(() => authLockService.status.value)
const trustedSessions = computed(() => authLockService.trustedSessions.value)
const otherTrustedSessionCount = computed(() => trustedSessions.value.filter((session) => !session.current).length)
const authEnabled = computed(() => isAuthLockEnabled())
const sessionTTLValue = computed(() => String(status.value.sessionTtlMinutes || 60))
/** 启用 PIN 表单是否可以提交。 */
const canSetupPIN = computed(() =>
  /^\d{4,8}$/.test(pinDraft.value.trim()) &&
  pinDraft.value.trim() === confirmPinDraft.value.trim() &&
  !setupBusy.value,
)
/** 修改 PIN 表单是否可以提交。 */
const canChangePIN = computed(() =>
  /^\d{4,8}$/.test(currentPinDraft.value.trim()) &&
  /^\d{4,8}$/.test(newPinDraft.value.trim()) &&
  newPinDraft.value.trim() === confirmNewPinDraft.value.trim() &&
  !changeBusy.value,
)

/** 进入设置安全分区时刷新 PIN 状态和受信任会话。 */
onMounted(async () => {
  if (authEnabled.value) {
    await refreshAuthStatus()
    if (status.value.pinEnabled) {
      await refreshTrustedSessions()
    }
  }
})

/** 关闭启用 PIN 对话框时清草稿。 */
watch(setupDialogOpen, (open) => {
  if (!open) {
    resetSetupDrafts()
  }
})

/** 关闭修改 PIN 对话框时清草稿。 */
watch(changeDialogOpen, (open) => {
  if (!open) {
    resetChangeDrafts()
  }
})

/** 只保留最多 8 位数字，供 PIN 输入框使用。 */
function normalizePIN(value: string | number): string {
  return String(value).replace(/\D/g, "").slice(0, 8)
}

/** 把安全设置请求错误转成页面可见文案。 */
function formatAuthError(error: unknown): string {
  if (error instanceof HttpClientError && error.apiError?.message) {
    return error.apiError.message
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  return t("settings.securitySaveFailed")
}

/** 清空启用 PIN 对话框草稿。 */
function resetSetupDrafts() {
  pinDraft.value = ""
  confirmPinDraft.value = ""
}

/** 清空修改 PIN 对话框草稿。 */
function resetChangeDrafts() {
  currentPinDraft.value = ""
  newPinDraft.value = ""
  confirmNewPinDraft.value = ""
}

/** 打开启用 PIN 对话框。 */
function openSetupDialog() {
  errorText.value = ""
  successText.value = ""
  resetSetupDrafts()
  setupDialogOpen.value = true
}

/** 打开修改 PIN 对话框。 */
function openChangeDialog() {
  errorText.value = ""
  successText.value = ""
  resetChangeDrafts()
  changeDialogOpen.value = true
}

/** 关闭启用 PIN 对话框。 */
function closeSetupDialog() {
  errorText.value = ""
  setupDialogOpen.value = false
}

/** 关闭修改 PIN 对话框。 */
function closeChangeDialog() {
  errorText.value = ""
  changeDialogOpen.value = false
}

/** 打开关闭 PIN 确认框。 */
function openDisableDialog() {
  errorText.value = ""
  successText.value = ""
  disableDialogOpen.value = true
}

/** 收起关闭 PIN 确认框。 */
function closeDisableDialog() {
  errorText.value = ""
  disableDialogOpen.value = false
}

/** 在已解锁会话下关闭 PIN 锁。 */
async function disablePIN() {
  errorText.value = ""
  successText.value = ""
  try {
    settingsBusy.value = true
    await authLockService.patchSettings({ pinEnabled: false })
    disableDialogOpen.value = false
    successText.value = t("settings.securityDisablePinSaved")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    settingsBusy.value = false
  }
}

/** 保存重启后是否锁定普通会话。 */
function onLockOnRestartChange(value: boolean) {
  void patchAuthSettings({ lockOnRestart: value })
}

/** 刷新当前 PIN 锁状态。 */
async function refreshAuthStatus() {
  try {
    await authLockService.refreshStatus()
  } catch (error) {
    errorText.value = formatAuthError(error)
  }
}

/** 提交启用 PIN 表单。 */
async function setupPIN() {
  errorText.value = ""
  successText.value = ""
  const pin = normalizePIN(pinDraft.value)
  const confirmPin = normalizePIN(confirmPinDraft.value)
  if (pin.length < 4 || pin !== confirmPin) {
    errorText.value = t("settings.securitySetupInvalid")
    return
  }
  try {
    setupBusy.value = true
    await authLockService.setupPin({
      pin,
      confirmPin,
      sessionTtlMinutes: status.value.sessionTtlMinutes,
      lockOnRestart: status.value.lockOnRestart,
    })
    await refreshTrustedSessions()
    setupDialogOpen.value = false
    resetSetupDrafts()
    successText.value = t("settings.securitySetupSaved")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    setupBusy.value = false
  }
}

/** 提交修改 PIN 表单。 */
async function changePIN() {
  errorText.value = ""
  successText.value = ""
  const currentPin = normalizePIN(currentPinDraft.value)
  const newPin = normalizePIN(newPinDraft.value)
  const confirmPin = normalizePIN(confirmNewPinDraft.value)
  if (currentPin.length < 4 || newPin.length < 4 || newPin !== confirmPin) {
    errorText.value = t("settings.securitySetupInvalid")
    return
  }
  try {
    changeBusy.value = true
    await authLockService.changePin({
      currentPin,
      newPin,
      confirmPin,
    })
    changeDialogOpen.value = false
    resetChangeDrafts()
    successText.value = t("settings.securityPinChanged")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    changeBusy.value = false
  }
}

/** 保存非密钥安全设置。 */
async function patchAuthSettings(patch: Parameters<typeof authLockService.patchSettings>[0]) {
  errorText.value = ""
  successText.value = ""
  try {
    settingsBusy.value = true
    await authLockService.patchSettings(patch)
    successText.value = t("settings.securitySettingsSaved")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    settingsBusy.value = false
  }
}

/** 保存无操作后锁定时长。 */
async function onSessionTTLChange(value: unknown) {
  const ttl = Number(value)
  if (!Number.isFinite(ttl) || ttl <= 0 || ttl === status.value.sessionTtlMinutes) {
    return
  }
  await patchAuthSettings({ sessionTtlMinutes: ttl })
}

/** 立即锁定当前浏览器会话。 */
async function lockNow() {
  errorText.value = ""
  successText.value = ""
  try {
    lockBusy.value = true
    await authLockService.lock()
    successText.value = t("settings.securityLockedNow")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    lockBusy.value = false
  }
}

/** 刷新永久信任会话列表。 */
async function refreshTrustedSessions() {
  if (!authEnabled.value || !status.value.pinEnabled) {
    return
  }
  try {
    trustedSessionsLoading.value = true
    await authLockService.listTrustedSessions()
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    trustedSessionsLoading.value = false
  }
}

/** 把会话时间格式化成本地可读文本。 */
function formatSessionTime(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(parsed)
}

/** 打开撤销单个受信任会话的确认框。 */
function requestTrustedSessionRevoke(session: AuthSessionDTO) {
  trustedSessionAction.value = { kind: "one", session }
  trustedSessionDialogOpen.value = true
}

/** 打开撤销其他受信任设备的确认框。 */
function requestOtherTrustedSessionsRevoke() {
  trustedSessionAction.value = { kind: "others" }
  trustedSessionDialogOpen.value = true
}

/** 确认撤销所选受信任会话。 */
async function confirmTrustedSessionRevoke() {
  const action = trustedSessionAction.value
  if (!action) {
    return
  }
  errorText.value = ""
  successText.value = ""
  try {
    trustedSessionActionBusy.value = true
    if (action.kind === "one") {
      await authLockService.revokeTrustedSession(action.session.publicId)
    } else {
      await authLockService.revokeOtherTrustedSessions()
    }
    trustedSessionDialogOpen.value = false
    trustedSessionAction.value = null
    successText.value = t("settings.securityTrustedSessionsRevoked")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    trustedSessionActionBusy.value = false
  }
}
</script>

<template>
  <div class="break-inside-avoid">
    <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <ShieldCheck class="size-4" />
        </span>
        <SettingsHint :text="t('settings.securityDesc')">
          <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t("settings.securityTitle") }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle>
        </SettingsHint>
      </CardHeader>

      <CardContent class="flex flex-col gap-3 pt-0">
        <div
          data-security-block
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
        >
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex min-w-0 flex-col gap-1">
              <SettingsHint :text="status.pinEnabled ? t('settings.securityEnabledHint') : t('settings.securitySetupHint')">
                <p class="text-sm font-semibold text-foreground">
                  {{ t("settings.securitySetupTitle") }}
                </p>
              </SettingsHint>
            </div>
            <div
              v-if="!status.pinEnabled"
              class="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center"
            >
              <Button
                data-setup-pin-trigger
                type="button"
                class="shrink-0 rounded-xl"
                :disabled="!authEnabled"
                @click="openSetupDialog"
              >
                {{ t("settings.securityEnablePin") }}
              </Button>
            </div>
            <div
              v-else
              class="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center"
            >
              <Button
                data-change-pin-trigger
                type="button"
                variant="outline"
                class="shrink-0 rounded-xl"
                :disabled="!authEnabled"
                @click="openChangeDialog"
              >
                {{ t("settings.securityChangePin") }}
              </Button>
              <Button
                data-disable-pin-trigger
                type="button"
                variant="outline"
                class="shrink-0 rounded-xl"
                :disabled="!authEnabled || settingsBusy"
                @click="openDisableDialog"
              >
                {{ t("settings.securityDisablePin") }}
              </Button>
            </div>
          </div>

          <Dialog v-model:open="setupDialogOpen">
            <DialogContent class="rounded-2xl border-border/70 sm:max-w-md">
              <form
                data-setup-pin-form
                class="flex flex-col gap-4"
                @submit.prevent="setupPIN"
              >
                <DialogHeader>
                  <DialogTitle>{{ t("settings.securityEnablePin") }}</DialogTitle>
                  <DialogDescription class="text-pretty">
                    {{ t("settings.securitySetupHint") }}
                  </DialogDescription>
                </DialogHeader>
                <div class="flex flex-col gap-3">
                  <Input
                    v-model="pinDraft"
                    data-setup-pin-input
                    type="password"
                    inputmode="numeric"
                    autocomplete="new-password"
                    class="h-9 min-h-9 max-h-9 px-3 py-0 text-sm"
                    :placeholder="t('settings.securityPinPlaceholder')"
                    @update:model-value="pinDraft = normalizePIN($event)"
                  />
                  <Input
                    v-model="confirmPinDraft"
                    data-confirm-pin-input
                    type="password"
                    inputmode="numeric"
                    autocomplete="new-password"
                    class="h-9 min-h-9 max-h-9 px-3 py-0 text-sm"
                    :placeholder="t('settings.securityConfirmPinPlaceholder')"
                    @update:model-value="confirmPinDraft = normalizePIN($event)"
                  />
                </div>
                <DialogFooter class="gap-3">
                  <Button
                    type="button"
                    variant="outline"
                    class="rounded-xl"
                    :disabled="setupBusy"
                    @click="closeSetupDialog"
                  >
                    {{ t("common.cancel") }}
                  </Button>
                  <Button type="submit" class="rounded-xl" :disabled="!authEnabled || !canSetupPIN">
                    {{ setupBusy ? t("settings.securitySaving") : t("settings.securityEnablePin") }}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>

          <Dialog v-model:open="changeDialogOpen">
            <DialogContent class="rounded-2xl border-border/70 sm:max-w-md">
              <form
                data-change-pin-form
                class="flex flex-col gap-4"
                @submit.prevent="changePIN"
              >
                <DialogHeader>
                  <DialogTitle>{{ t("settings.securityChangePin") }}</DialogTitle>
                  <DialogDescription class="text-pretty">
                    {{ t("settings.securityEnabledHint") }}
                  </DialogDescription>
                </DialogHeader>
                <div class="flex flex-col gap-3">
                  <Input
                    v-model="currentPinDraft"
                    data-current-pin-input
                    type="password"
                    inputmode="numeric"
                    autocomplete="current-password"
                    class="h-9 min-h-9 max-h-9 px-3 py-0 text-sm"
                    :placeholder="t('settings.securityCurrentPinPlaceholder')"
                    @update:model-value="currentPinDraft = normalizePIN($event)"
                  />
                  <Input
                    v-model="newPinDraft"
                    data-new-pin-input
                    type="password"
                    inputmode="numeric"
                    autocomplete="new-password"
                    class="h-9 min-h-9 max-h-9 px-3 py-0 text-sm"
                    :placeholder="t('settings.securityNewPinPlaceholder')"
                    @update:model-value="newPinDraft = normalizePIN($event)"
                  />
                  <Input
                    v-model="confirmNewPinDraft"
                    data-confirm-new-pin-input
                    type="password"
                    inputmode="numeric"
                    autocomplete="new-password"
                    class="h-9 min-h-9 max-h-9 px-3 py-0 text-sm"
                    :placeholder="t('settings.securityConfirmNewPinPlaceholder')"
                    @update:model-value="confirmNewPinDraft = normalizePIN($event)"
                  />
                </div>
                <DialogFooter class="gap-3">
                  <Button
                    type="button"
                    variant="outline"
                    class="rounded-xl"
                    :disabled="changeBusy"
                    @click="closeChangeDialog"
                  >
                    {{ t("common.cancel") }}
                  </Button>
                  <Button type="submit" class="rounded-xl" :disabled="!authEnabled || !canChangePIN">
                    {{ changeBusy ? t("settings.securitySaving") : t("settings.securityChangePin") }}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>

          <Dialog v-model:open="disableDialogOpen">
            <DialogContent class="rounded-2xl border-border/70 sm:max-w-md">
              <DialogHeader>
                <DialogTitle>{{ t("settings.securityDisablePinTitle") }}</DialogTitle>
                <DialogDescription class="text-pretty">
                  {{ t("settings.securityDisablePinHint") }}
                </DialogDescription>
              </DialogHeader>
              <DialogFooter class="gap-3">
                <Button
                  type="button"
                  variant="outline"
                  class="rounded-xl"
                  :disabled="settingsBusy"
                  @click="closeDisableDialog"
                >
                  {{ t("common.cancel") }}
                </Button>
                <Button
                  data-confirm-disable-pin
                  type="button"
                  variant="destructive"
                  class="rounded-xl"
                  :disabled="!authEnabled || settingsBusy"
                  @click="disablePIN"
                >
                  {{ settingsBusy ? t("settings.securitySaving") : t("settings.securityDisablePinConfirm") }}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <SettingsHint :text="t('settings.securityLockNowHint')">
              <Button
                variant="outline"
                size="sm"
                class="shrink-0 rounded-xl"
                :disabled="!status.pinEnabled || lockBusy"
                @click="lockNow"
              >
                <LockKeyhole data-icon="inline-start" />
                {{ t("settings.securityLockNow") }}
              </Button>
            </SettingsHint>
          </div>
        </div>

        <div
          data-security-block
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex min-w-0 flex-col gap-1">
            <SettingsHint :text="t('settings.securitySessionHint')">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.securitySessionTitle") }}
              </p>
            </SettingsHint>
          </div>
          <Select
            :model-value="sessionTTLValue"
            @update:model-value="onSessionTTLChange"
          >
            <SelectTrigger
              size="sm"
              class="h-9 w-full min-w-[9rem] shrink-0 rounded-xl border-border/50 sm:w-36"
              :aria-label="t('settings.securitySessionTitle')"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end" class="rounded-xl border-border/50">
              <SelectGroup>
                <SelectItem value="15">{{ t("settings.securitySession15") }}</SelectItem>
                <SelectItem value="60">{{ t("settings.securitySession60") }}</SelectItem>
                <SelectItem value="240">{{ t("settings.securitySession240") }}</SelectItem>
                <SelectItem value="1440">{{ t("settings.securitySession1440") }}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div
          data-security-block
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex min-w-0 flex-col gap-1">
            <SettingsHint :text="t('settings.securityLockOnRestartHint')">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.securityLockOnRestart") }}
              </p>
            </SettingsHint>
          </div>
          <Switch
            :model-value="status.lockOnRestart"
            :disabled="settingsBusy"
            :aria-label="t('settings.securityLockOnRestart')"
            @update:model-value="onLockOnRestartChange"
          />
        </div>

        <div
          data-security-block
          data-trusted-sessions
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
        >
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div class="flex min-w-0 flex-col gap-1">
              <SettingsHint :text="t('settings.securityTrustedSessionsHint')">
                <p class="text-sm font-semibold text-foreground">
                  {{ t("settings.securityTrustedSessionsTitle") }}
                </p>
              </SettingsHint>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              class="shrink-0 rounded-xl"
              :disabled="!status.pinEnabled || trustedSessionsLoading"
              @click="refreshTrustedSessions"
            >
              {{ t("settings.securityTrustedSessionsRefresh") }}
            </Button>
          </div>

          <div v-if="trustedSessionsLoading" class="flex flex-col gap-2" aria-busy="true">
            <Skeleton class="h-16 w-full rounded-xl" />
            <Skeleton class="h-16 w-full rounded-xl" />
          </div>

          <p
            v-else-if="trustedSessions.length === 0"
            class="rounded-xl border border-dashed border-border/60 px-3 py-4 text-sm text-muted-foreground"
          >
            {{ t("settings.securityTrustedSessionsEmpty") }}
          </p>

          <div v-else class="flex flex-col">
            <template v-for="(session, index) in trustedSessions" :key="session.publicId">
              <div class="flex flex-col gap-3 py-3 first:pt-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex min-w-0 flex-col gap-1.5">
                  <div class="flex flex-wrap items-center gap-1.5">
                    <Badge v-if="session.current" variant="default">
                      {{ t("settings.securityTrustedSessionsCurrent") }}
                    </Badge>
                    <Badge variant="secondary">
                      {{ t("settings.securityTrustedSessionsTrusted") }}
                    </Badge>
                  </div>
                  <p class="truncate text-sm font-medium text-foreground" :title="session.userAgent">
                    {{ session.userAgent || t("settings.securityTrustedSessionsUnknownClient") }}
                  </p>
                  <p class="text-xs leading-relaxed text-muted-foreground">
                    {{ session.ip || t("settings.securityTrustedSessionsUnknownIp") }} ·
                    {{ t("settings.securityTrustedSessionsLastSeen", { time: formatSessionTime(session.lastSeenAt) }) }}
                  </p>
                </div>
                <Button
                  v-if="!session.current"
                  type="button"
                  variant="outline"
                  size="sm"
                  class="shrink-0 rounded-xl"
                  :disabled="trustedSessionActionBusy"
                  @click="requestTrustedSessionRevoke(session)"
                >
                  {{ t("settings.securityTrustedSessionsRevoke") }}
                </Button>
              </div>
              <Separator v-if="index < trustedSessions.length - 1" />
            </template>
          </div>

          <div v-if="otherTrustedSessionCount > 0" class="flex justify-end">
            <Button
              type="button"
              variant="outline"
              size="sm"
              class="rounded-xl"
              :disabled="trustedSessionActionBusy"
              @click="requestOtherTrustedSessionsRevoke"
            >
              {{ t("settings.securityTrustedSessionsRevokeOthers") }}
            </Button>
          </div>
        </div>

        <Dialog v-model:open="trustedSessionDialogOpen">
          <DialogContent class="rounded-2xl border-border/70 sm:max-w-md">
            <DialogHeader>
              <DialogTitle>{{ t("settings.securityTrustedSessionsConfirmTitle") }}</DialogTitle>
              <DialogDescription class="text-pretty">
                {{ trustedSessionAction?.kind === "others"
                  ? t("settings.securityTrustedSessionsConfirmOthers")
                  : t("settings.securityTrustedSessionsConfirmOne") }}
              </DialogDescription>
            </DialogHeader>
            <DialogFooter class="gap-3">
              <Button
                type="button"
                variant="outline"
                class="rounded-xl"
                :disabled="trustedSessionActionBusy"
                @click="trustedSessionDialogOpen = false"
              >
                {{ t("common.cancel") }}
              </Button>
              <Button
                data-confirm-session-revoke
                type="button"
                variant="destructive"
                class="rounded-xl"
                :disabled="trustedSessionActionBusy"
                @click="confirmTrustedSessionRevoke"
              >
                {{ t("settings.securityTrustedSessionsRevoke") }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <div
          data-security-block
          role="note"
          class="flex flex-col gap-2 rounded-2xl border border-border/40 border-l-[3px] border-l-muted-foreground/40 bg-surface-muted px-4 py-3"
        >
          <SettingsHint :text="t('settings.securityLanPolicyHint')">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securityLanPolicyTitle") }}
            </p>
          </SettingsHint>
        </div>

        <p
          v-if="errorText"
          class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          role="alert"
        >
          {{ errorText }}
        </p>
        <p
          v-if="successText"
          class="text-sm text-muted-foreground"
          role="status"
        >
          {{ successText }}
        </p>
      </CardContent>
    </Card>
  </div>
</template>
