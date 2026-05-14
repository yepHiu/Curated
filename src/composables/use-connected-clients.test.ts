import { flushPromises } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import type { ConnectedClientsDTO } from "@/api/types"
import { useConnectedClients } from "./use-connected-clients"

function connectedClientsDto(overrides: Partial<ConnectedClientsDTO> = {}): ConnectedClientsDTO {
  return {
    clients: [
      {
        key: "local-chrome",
        ip: "127.0.0.1",
        browser: "Chrome",
        os: "Windows",
        deviceType: "desktop",
        accessKind: "local",
        isLocalMachine: true,
        firstSeen: "2026-05-15T10:00:00Z",
        lastSeen: "2026-05-15T10:01:00Z",
        requestCount: 3,
      },
    ],
    total: 1,
    localCount: 1,
    remoteCount: 0,
    sampledAt: "2026-05-15T10:01:00Z",
    ...overrides,
  }
}

describe("useConnectedClients", () => {
  it("refreshes immediately while active and polls until inactive", async () => {
    vi.useFakeTimers()
    const active = ref(false)
    const listConnectedClients = vi.fn()
      .mockResolvedValueOnce(connectedClientsDto())
      .mockResolvedValueOnce(connectedClientsDto({ sampledAt: "2026-05-15T10:02:00Z" }))
    const state = useConnectedClients(
      { listConnectedClients },
      active,
      { pollMs: 60_000 },
    )

    expect(listConnectedClients).not.toHaveBeenCalled()

    active.value = true
    await flushPromises()

    expect(listConnectedClients).toHaveBeenCalledTimes(1)
    expect(state.clients.value).toHaveLength(1)
    expect(state.sampledAt.value).toBe("2026-05-15T10:01:00Z")

    await vi.advanceTimersByTimeAsync(60_000)
    await flushPromises()

    expect(listConnectedClients).toHaveBeenCalledTimes(2)
    expect(state.sampledAt.value).toBe("2026-05-15T10:02:00Z")

    active.value = false
    await vi.advanceTimersByTimeAsync(60_000)
    await flushPromises()

    expect(listConnectedClients).toHaveBeenCalledTimes(2)
    state.stop()
    vi.useRealTimers()
  })

  it("keeps the previous successful clients when a refresh fails", async () => {
    const active = ref(true)
    const listConnectedClients = vi.fn()
      .mockResolvedValueOnce(connectedClientsDto())
      .mockRejectedValueOnce(new Error("network down"))
    const state = useConnectedClients(
      { listConnectedClients },
      active,
      { pollMs: 60_000 },
    )

    await flushPromises()
    await state.refresh()

    expect(state.clients.value).toHaveLength(1)
    expect(state.error.value).toBe("network down")
  })
})
