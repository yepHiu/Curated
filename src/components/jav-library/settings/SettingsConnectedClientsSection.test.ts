import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ConnectedClientDTO } from "@/api/types"
import SettingsConnectedClientsSection from "./SettingsConnectedClientsSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key} ${JSON.stringify(params)}` : key,
  }),
}))

const clients: ConnectedClientDTO[] = [
  {
    key: "local-chrome",
    ip: "127.0.0.1",
    browser: "Chrome",
    browserVersion: "132",
    os: "Windows",
    deviceType: "desktop",
    accessKind: "local",
    isLocalMachine: true,
    firstSeen: "2026-05-15T10:00:00Z",
    lastSeen: "2026-05-15T10:01:00Z",
    requestCount: 3,
  },
  {
    key: "iphone-safari",
    ip: "192.168.1.8",
    hostname: "iphone.lan",
    browser: "Safari",
    os: "iOS",
    deviceType: "mobile",
    accessKind: "remote",
    isLocalMachine: false,
    firstSeen: "2026-05-15T09:50:00Z",
    lastSeen: "2026-05-15T09:55:00Z",
    requestCount: 12,
  },
]

describe("SettingsConnectedClientsSection", () => {
  it("renders connected client visibility without exposing MAC address copy", async () => {
    const wrapper = mount(SettingsConnectedClientsSection, {
      props: {
        clients,
        total: 2,
        localCount: 1,
        remoteCount: 1,
        sampledAt: "2026-05-15T10:01:00Z",
      },
    })

    expect(wrapper.text()).toContain("settings.connectedClientsTitle")
    expect(wrapper.text()).toContain("Chrome")
    expect(wrapper.text()).toContain("127.0.0.1")
    expect(wrapper.text()).toContain("Safari")
    expect(wrapper.text()).toContain("iphone.lan")
    expect(wrapper.text()).toContain("settings.connectedClientsThisDevice")
    expect(wrapper.text()).toContain("settings.connectedClientsPrivacy")
    expect(wrapper.text()).not.toContain("MAC")

    await wrapper.get("button").trigger("click")

    expect(wrapper.emitted("refresh")).toHaveLength(1)
  })
})
