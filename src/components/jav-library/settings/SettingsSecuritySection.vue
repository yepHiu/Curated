<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { LockKeyhole, ShieldCheck } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
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
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
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
const settingsBusy = ref(false)
const lockBusy = ref(false)
const errorText = ref("")
const successText = ref("")

const status = computed(() => authLockService.status.value)
const authEnabled = computed(() => isAuthLockEnabled())
const sessionTTLValue = computed(() => String(status.value.sessionTtlMinutes || 60))
const canSetupPIN = computed(() =>
  /^\d{4,8}$/.test(pinDraft.value.trim()) &&
  pinDraft.value.trim() === confirmPinDraft.value.trim() &&
  !setupBusy.value,
)
const canChangePIN = computed(() =>
  /^\d{4,8}$/.test(currentPinDraft.value.trim()) &&
  /^\d{4,8}$/.test(newPinDraft.value.trim()) &&
  newPinDraft.value.trim() === confirmNewPinDraft.value.trim() &&
  !changeBusy.value,
)

onMounted(() => {
  if (authEnabled.value) {
    void refreshAuthStatus()
  }
})

watch(setupDialogOpen, (open) => {
  if (!open) {
    resetSetupDrafts()
  }
})

watch(changeDialogOpen, (open) => {
  if (!open) {
    resetChangeDrafts()
  }
})

function normalizePIN(value: string | number): string {
  return String(value).replace(/\D/g, "").slice(0, 8)
}

function formatAuthError(error: unknown): string {
  if (error instanceof HttpClientError && error.apiError?.message) {
    return error.apiError.message
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  return t("settings.securitySaveFailed")
}

function resetSetupDrafts() {
  pinDraft.value = ""
  confirmPinDraft.value = ""
}

function resetChangeDrafts() {
  currentPinDraft.value = ""
  newPinDraft.value = ""
  confirmNewPinDraft.value = ""
}

function openSetupDialog() {
  errorText.value = ""
  successText.value = ""
  resetSetupDrafts()
  setupDialogOpen.value = true
}

function openChangeDialog() {
  errorText.value = ""
  successText.value = ""
  resetChangeDrafts()
  changeDialogOpen.value = true
}

function closeSetupDialog() {
  errorText.value = ""
  setupDialogOpen.value = false
}

function closeChangeDialog() {
  errorText.value = ""
  changeDialogOpen.value = false
}

function onLanRequiresPINChange(value: boolean) {
  void patchAuthSettings({ lanRequiresPin: value })
}

function onLockOnRestartChange(value: boolean) {
  void patchAuthSettings({ lockOnRestart: value })
}

async function refreshAuthStatus() {
  try {
    await authLockService.refreshStatus()
  } catch (error) {
    errorText.value = formatAuthError(error)
  }
}

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
      lanRequiresPin: status.value.lanRequiresPin,
      lockOnRestart: status.value.lockOnRestart,
    })
    setupDialogOpen.value = false
    resetSetupDrafts()
    successText.value = t("settings.securitySetupSaved")
  } catch (error) {
    errorText.value = formatAuthError(error)
  } finally {
    setupBusy.value = false
  }
}

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

async function onSessionTTLChange(value: unknown) {
  const ttl = Number(value)
  if (!Number.isFinite(ttl) || ttl <= 0 || ttl === status.value.sessionTtlMinutes) {
    return
  }
  await patchAuthSettings({ sessionTtlMinutes: ttl })
}

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
        <CardTitle class="min-w-0 text-lg tracking-tight">
          {{ t("settings.securityTitle") }}
        </CardTitle>
        <CardDescription
          class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
        >
          {{ t("settings.securityDesc") }}
        </CardDescription>
      </CardHeader>

      <CardContent class="flex flex-col gap-3 pt-0">
        <div
          data-security-block
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
        >
          <div class="flex flex-col gap-1">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securitySetupTitle") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ status.pinEnabled ? t("settings.securityEnabledHint") : t("settings.securitySetupHint") }}
            </p>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p
              v-if="status.pinEnabled"
              class="text-sm text-muted-foreground"
            >
              {{ t("settings.securityPinEnabled") }}
            </p>
            <p
              v-else
              class="text-sm text-muted-foreground"
            >
              {{ t("settings.securitySetupHint") }}
            </p>
            <Button
              v-if="!status.pinEnabled"
              data-setup-pin-trigger
              type="button"
              class="shrink-0 rounded-xl"
              :disabled="!authEnabled"
              @click="openSetupDialog"
            >
              {{ t("settings.securityEnablePin") }}
            </Button>
            <Button
              v-else
              data-change-pin-trigger
              type="button"
              variant="outline"
              class="shrink-0 rounded-xl"
              :disabled="!authEnabled"
              @click="openChangeDialog"
            >
              {{ t("settings.securityChangePin") }}
            </Button>
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

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.securityLockNowHint") }}
            </p>
            <Button
              variant="outline"
              size="sm"
              class="shrink-0 rounded-xl"
              :disabled="!status.pinEnabled || lockBusy"
              @click="lockNow"
            >
              <LockKeyhole class="size-4" />
              {{ t("settings.securityLockNow") }}
            </Button>
          </div>
        </div>

        <div
          data-security-block
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex min-w-0 flex-col gap-1">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securitySessionTitle") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.securitySessionHint") }}
            </p>
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
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securityLanRequiresPin") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.securityLanRequiresPinHint") }}
            </p>
          </div>
          <Switch
            :model-value="status.lanRequiresPin"
            :disabled="settingsBusy"
            :aria-label="t('settings.securityLanRequiresPin')"
            @update:model-value="onLanRequiresPINChange"
          />
        </div>

        <div
          data-security-block
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex min-w-0 flex-col gap-1">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securityLockOnRestart") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.securityLockOnRestartHint") }}
            </p>
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
          role="note"
          class="flex flex-col gap-4 rounded-2xl border border-border/40 border-l-[3px] border-l-muted-foreground/40 bg-surface-muted px-4 py-3"
        >
          <div class="flex flex-col gap-2">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securityTrustDeviceTitle") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.securityTrustDeviceHint") }}
            </p>
          </div>

          <div class="flex flex-col gap-2 border-t border-border/35 pt-4">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.securityLanPolicyTitle") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.securityLanPolicyHint") }}
            </p>
          </div>
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
