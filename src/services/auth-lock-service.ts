import { computed, ref } from "vue"
import { api } from "@/api/endpoints"
import type {
  AuthStatusDTO,
  ChangePinBody,
  PatchAuthSettingsBody,
  SetupPinBody,
  UnlockPinBody,
} from "@/api/types"

const defaultStatus: AuthStatusDTO = {
  pinEnabled: false,
  unlocked: true,
  setupRequired: true,
  pinLength: 0,
  trustedForever: false,
  sessionTtlMinutes: 60,
  lanRequiresPin: true,
  lockOnRestart: true,
}

const statusState = ref<AuthStatusDTO>({ ...defaultStatus })
let refreshInFlight: Promise<AuthStatusDTO> | null = null

function setStatus(next: AuthStatusDTO): AuthStatusDTO {
  statusState.value = {
    ...defaultStatus,
    ...next,
    sessionTtlMinutes: Math.max(1, Number(next.sessionTtlMinutes ?? defaultStatus.sessionTtlMinutes)),
  }
  return statusState.value
}

export function isAuthLockEnabled(): boolean {
  return import.meta.env.VITE_USE_WEB_API === "true"
}

export const authLockService = {
  status: computed(() => statusState.value),

  async refreshStatus(): Promise<AuthStatusDTO> {
    if (refreshInFlight) return refreshInFlight
    refreshInFlight = api.authStatus()
      .then(setStatus)
      .finally(() => {
        refreshInFlight = null
      })
    return refreshInFlight
  },

  async setupPin(body: SetupPinBody): Promise<AuthStatusDTO> {
    return setStatus(await api.setupPin(body))
  },

  async unlock(body: UnlockPinBody): Promise<AuthStatusDTO> {
    return setStatus(await api.unlockPin(body))
  },

  async changePin(body: ChangePinBody): Promise<AuthStatusDTO> {
    return setStatus(await api.changePin(body))
  },

  async lock(): Promise<AuthStatusDTO> {
    return setStatus(await api.lockApp())
  },

  async patchSettings(body: PatchAuthSettingsBody): Promise<AuthStatusDTO> {
    return setStatus(await api.patchAuthSettings(body))
  },
}
