import { afterEach, describe, expect, it, vi } from "vitest"
import type { TaskDTO } from "@/api/types"
import { resolveBackendEventsUrl, subscribeBackendEvents } from "./backend-events"

class FakeEventSource {
  static instances: FakeEventSource[] = []

  readonly listeners = new Map<string, Set<(event: MessageEvent<string>) => void>>()
  onopen: ((event: Event) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  closed = false
  readonly url: string
  readonly init?: EventSourceInit

  constructor(url: string, init?: EventSourceInit) {
    this.url = url
    this.init = init
    FakeEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    const next = this.listeners.get(type) ?? new Set()
    next.add(listener as (event: MessageEvent<string>) => void)
    this.listeners.set(type, next)
  }

  close() {
    this.closed = true
  }

  emit(type: string, data: string) {
    for (const listener of this.listeners.get(type) ?? []) {
      listener({ data } as MessageEvent<string>)
    }
  }
}

function makeTask(overrides: Partial<TaskDTO> = {}): TaskDTO {
  return {
    taskId: "task-1",
    type: "scan.library",
    status: "running",
    createdAt: "2026-06-28T00:00:00Z",
    progress: 25,
    message: "Scanning",
    ...overrides,
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
  FakeEventSource.instances = []
})

describe("backend events", () => {
  it("resolves the events URL through the same API base rules as regular HTTP calls", () => {
    expect(
      resolveBackendEventsUrl(
        { DEV: true, VITE_USE_WEB_API: "true", VITE_API_BASE_URL: "" },
        "http://127.0.0.1:5173",
      ),
    ).toBe("http://127.0.0.1:8080/api/events")

    expect(
      resolveBackendEventsUrl(
        { DEV: false, VITE_USE_WEB_API: "true", VITE_API_BASE_URL: "" },
        "http://app.local:8081",
      ),
    ).toBe("http://app.local:8081/api/events")

    expect(
      resolveBackendEventsUrl(
        { DEV: false, VITE_USE_WEB_API: "true", VITE_API_BASE_URL: "http://server.test/api" },
        "http://app.local:8081",
      ),
    ).toBe("http://server.test/api/events")
  })

  it("subscribes with credentials and parses task.updated events", () => {
    vi.stubGlobal("EventSource", FakeEventSource)
    const onTaskUpdated = vi.fn()

    const subscription = subscribeBackendEvents({ onTaskUpdated })

    const instance = FakeEventSource.instances[0]
    expect(instance.url).toMatch(/\/api\/events$/)
    expect(instance.init).toEqual({ withCredentials: true })

    const task = makeTask({ status: "completed", progress: 100 })
    instance.emit("task.updated", JSON.stringify({ type: "task.updated", task }))

    expect(onTaskUpdated).toHaveBeenCalledWith(task)

    subscription.close()
    expect(instance.closed).toBe(true)
  })
})
