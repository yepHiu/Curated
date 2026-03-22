import { ref, type Ref } from "vue"

export type AppToastVariant = "default" | "success" | "destructive"

const toastOpen: Ref<boolean> = ref(false)
const toastMessage: Ref<string> = ref("")
const toastVariant: Ref<AppToastVariant> = ref("default")

let hideTimer: ReturnType<typeof setTimeout> | null = null

export function useAppToastHost() {
  return {
    toastOpen,
    toastMessage,
    toastVariant,
  }
}

export function pushAppToast(
  message: string,
  options?: { variant?: AppToastVariant; durationMs?: number },
): void {
  const variant = options?.variant ?? "default"
  const durationMs = options?.durationMs ?? 4500
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
  toastMessage.value = message
  toastVariant.value = variant
  toastOpen.value = true
  hideTimer = setTimeout(() => {
    toastOpen.value = false
    hideTimer = null
  }, durationMs)
}
