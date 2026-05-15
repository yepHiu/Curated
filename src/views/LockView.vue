<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { LockKeyhole } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { authLockService } from "@/services/auth-lock-service"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const authStatus = computed(() => authLockService.status.value)

const pin = ref("")
const trustedForever = ref(false)
const submitting = ref(false)
const errorText = ref("")
const forgotPinHelpOpen = ref(false)
const pinInputRef = ref<HTMLInputElement | null>(null)

const visiblePINLength = computed(() => {
  const configuredLength = Number(authStatus.value.pinLength)
  if (Number.isInteger(configuredLength) && configuredLength >= 4 && configuredLength <= 8) {
    return configuredLength
  }
  return Math.min(8, Math.max(4, pin.value.length))
})

const pinCells = computed(() =>
  Array.from({ length: visiblePINLength.value }, (_, index) => pin.value[index] ?? ""),
)

function normalizePIN(value: string): string {
  return value.replace(/\D/g, "").slice(0, 8)
}

function focusPinInput() {
  void nextTick(() => {
    pinInputRef.value?.focus()
  })
}

onMounted(() => {
  focusPinInput()
})

function redirectTarget(): string {
  const raw = route.query.redirect
  const value = typeof raw === "string" ? raw : Array.isArray(raw) ? raw[0] : ""
  if (!value || !value.startsWith("/")) {
    return "/"
  }
  return value
}

async function submitUnlock() {
  errorText.value = ""
  const candidate = normalizePIN(pin.value)
  pin.value = candidate
  if (candidate.length < 4) {
    errorText.value = t("lock.pinTooShort")
    return
  }

  try {
    submitting.value = true
    await authLockService.unlock({
      pin: candidate,
      trustedForever: trustedForever.value,
    })
    await router.replace(redirectTarget())
  } catch (error) {
    if (error instanceof HttpClientError && error.apiError?.message) {
      errorText.value = error.apiError.message
    } else {
      errorText.value = t("lock.unlockFailed")
    }
    pin.value = ""
    focusPinInput()
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="flex min-h-dvh items-center justify-center bg-background px-4 py-10 text-foreground">
    <Card class="w-full max-w-md gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="flex flex-row items-center gap-x-2.5 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <LockKeyhole class="size-4" />
        </span>
        <CardTitle class="min-w-0 flex-1 text-lg tracking-tight">
          {{ t("lock.title") }}
        </CardTitle>
      </CardHeader>

      <CardContent class="pt-0">
        <form class="flex flex-col gap-4" @submit.prevent="submitUnlock">
          <label class="sr-only" for="curated-pin-input">{{ t("lock.pinLabel") }}</label>
          <input
            id="curated-pin-input"
            ref="pinInputRef"
            data-pin-input
            class="sr-only"
            :value="pin"
            inputmode="numeric"
            pattern="[0-9]*"
            autocomplete="current-password"
            autofocus
            :aria-label="t('lock.pinLabel')"
            @input="pin = normalizePIN(($event.target as HTMLInputElement).value)"
          >

          <div
            class="flex justify-center gap-2"
            aria-hidden="true"
            @click="focusPinInput"
          >
            <span
              v-for="(cell, index) in pinCells"
              :key="index"
              data-pin-cell
              class="flex size-10 items-center justify-center rounded-lg border border-border bg-muted/20 text-base font-semibold tabular-nums"
            >
              {{ cell ? "*" : "" }}
            </span>
          </div>

          <div
            class="mx-auto flex w-full max-w-xs items-center justify-between gap-4 rounded-lg border border-border/50 bg-muted/5 p-4 text-sm"
          >
            <span class="min-w-0 font-medium text-foreground">{{ t("lock.trustForever") }}</span>
            <Switch
              data-trust-forever
              :model-value="trustedForever"
              :aria-label="t('lock.trustForever')"
              @update:model-value="trustedForever = Boolean($event)"
            />
          </div>

          <p
            v-if="errorText"
            class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {{ errorText }}
          </p>

          <Button type="submit" class="mx-auto w-full max-w-xs" :disabled="submitting || pin.length < 4">
            {{ submitting ? t("lock.unlocking") : t("lock.unlock") }}
          </Button>

          <button
            type="button"
            data-forgot-pin
            class="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
            :aria-expanded="forgotPinHelpOpen"
            @click="forgotPinHelpOpen = !forgotPinHelpOpen"
          >
            {{ t("lock.forgotPin") }}
          </button>

          <p
            v-if="forgotPinHelpOpen"
            class="rounded-lg border border-border/50 bg-muted/5 px-3 py-2 text-xs leading-relaxed text-muted-foreground sm:text-sm"
          >
            {{ t("lock.forgotPinHint") }}
          </p>
        </form>
      </CardContent>
    </Card>
  </main>
</template>
