import { computed, ref } from "vue"
import { api } from "@/api/endpoints"
import type {
  AuthStatusDTO,
  AuthSessionDTO,
  AuthSessionsDTO,
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
const trustedSessionsState = ref<AuthSessionDTO[]>([])
let refreshInFlight: Promise<AuthStatusDTO> | null = null

function setStatus(next: AuthStatusDTO): AuthStatusDTO {
  statusState.value = {
    ...defaultStatus,
    ...next,
    sessionTtlMinutes: Math.max(1, Number(next.sessionTtlMinutes ?? defaultStatus.sessionTtlMinutes)),
  }
  return statusState.value
}

function setTrustedSessions(next: AuthSessionsDTO): AuthSessionDTO[] {
  trustedSessionsState.value = Array.isArray(next.items) ? [...next.items] : []
  return trustedSessionsState.value
}

export function isAuthLockEnabled(): boolean {
  return import.meta.env.VITE_USE_WEB_API === "true"
}

export const authLockService = {
  status: computed(() => statusState.value),
  trustedSessions: computed(() => trustedSessionsState.value),

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
    const status = setStatus(await api.lockApp())
    trustedSessionsState.value = []
    return status
  },

  /** 更新非密钥安全设置；关闭 PIN 时清空本地受信任会话列表。 */
  async patchSettings(body: PatchAuthSettingsBody): Promise<AuthStatusDTO> {
    const status = setStatus(await api.patchAuthSettings(body))
    if (!status.pinEnabled) {
      trustedSessionsState.value = []
    }
    return status
  },

  async listTrustedSessions(): Promise<AuthSessionDTO[]> {
    return setTrustedSessions(await api.listTrustedAuthSessions())
  },

  async revokeTrustedSession(publicId: string): Promise<AuthSessionDTO[]> {
    return setTrustedSessions(await api.revokeTrustedAuthSession(publicId))
  },

  async revokeOtherTrustedSessions(): Promise<AuthSessionDTO[]> {
    return setTrustedSessions(await api.revokeOtherTrustedAuthSessions())
  },
}
