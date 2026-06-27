import type { TaskDTO } from "@/api/types"
import { resolveApiBaseUrl } from "@/api/http-client"

type BackendEventsEnv = Pick<ImportMetaEnv, "VITE_API_BASE_URL" | "VITE_USE_WEB_API"> & {
  DEV?: boolean
}

export interface BackendEventSubscription {
  close(): void
}

export interface SubscribeBackendEventsOptions {
  onTaskUpdated?: (task: TaskDTO) => void
  onOpen?: () => void
  onError?: (event: Event) => void
}

interface BackendTaskEventPayload {
  type?: string
  task?: TaskDTO
}

export function resolveBackendEventsUrl(
  env: BackendEventsEnv = import.meta.env,
  origin = window.location.origin,
): string {
  const apiBaseUrl = resolveApiBaseUrl(env, origin)
  return new URL(`${apiBaseUrl.replace(/\/$/, "")}/events`, origin).href
}

export function subscribeBackendEvents(
  options: SubscribeBackendEventsOptions,
): BackendEventSubscription {
  if (typeof EventSource === "undefined") {
    return { close() {} }
  }

  const source = new EventSource(resolveBackendEventsUrl(), {
    withCredentials: true,
  })

  source.onopen = () => {
    options.onOpen?.()
  }
  source.onerror = (event) => {
    options.onError?.(event)
  }
  source.addEventListener("task.updated", (event) => {
    const task = parseTaskUpdatedEvent((event as MessageEvent<string>).data)
    if (task) {
      options.onTaskUpdated?.(task)
    }
  })

  return {
    close() {
      source.close()
    },
  }
}

function parseTaskUpdatedEvent(data: string): TaskDTO | null {
  try {
    const payload = JSON.parse(data) as BackendTaskEventPayload
    if (payload.type !== "task.updated" || !payload.task?.taskId) {
      return null
    }
    return payload.task
  } catch {
    return null
  }
}
