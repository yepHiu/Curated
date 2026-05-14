import { onScopeDispose, ref, watch, type Ref } from "vue"
import type {
  ConnectedClientDTO,
  ConnectedClientsDTO,
} from "@/api/types"

type ConnectedClientsService = {
  listConnectedClients(): Promise<ConnectedClientsDTO>
}

export function useConnectedClients(
  service: ConnectedClientsService,
  active: Ref<boolean>,
  options: { pollMs?: number } = {},
) {
  const pollMs = options.pollMs ?? 60_000
  const clients = ref<ConnectedClientDTO[]>([])
  const total = ref(0)
  const localCount = ref(0)
  const remoteCount = ref(0)
  const sampledAt = ref("")
  const loading = ref(false)
  const error = ref("")
  let timer: ReturnType<typeof setInterval> | null = null
  let refreshSeq = 0

  function stopPolling() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  function startPolling() {
    if (timer !== null || pollMs <= 0) {
      return
    }
    timer = setInterval(() => {
      void refresh()
    }, pollMs)
  }

  async function refresh() {
    const seq = ++refreshSeq
    loading.value = true
    error.value = ""
    try {
      const dto = await service.listConnectedClients()
      if (seq !== refreshSeq) {
        return
      }
      clients.value = [...dto.clients]
      total.value = dto.total
      localCount.value = dto.localCount
      remoteCount.value = dto.remoteCount
      sampledAt.value = dto.sampledAt
    } catch (err) {
      if (seq !== refreshSeq) {
        return
      }
      error.value = err instanceof Error && err.message.trim()
        ? err.message
        : "Failed to load connected clients"
    } finally {
      if (seq === refreshSeq) {
        loading.value = false
      }
    }
  }

  watch(
    active,
    (isActive) => {
      if (!isActive) {
        stopPolling()
        return
      }
      void refresh()
      startPolling()
    },
    { immediate: true },
  )

  onScopeDispose(stopPolling)

  return {
    clients,
    total,
    localCount,
    remoteCount,
    sampledAt,
    loading,
    error,
    refresh,
    stop: stopPolling,
  }
}
