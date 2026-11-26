import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import { ref } from "vue"

import AgentChatOutcome from "./AgentChatOutcome.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

describe("agent outcome failure reasons", () => {
  it("localizes provider failures instead of showing the raw server sentence", () => {
    const wrapper = mount(AgentChatOutcome, {
      props: { outcome: { status: "failed", reason: "sk-live-secret", reasonCode: "provider_authentication", retryable: true } },
    })
    expect(wrapper.text()).toContain("agentWindow.outcomeReasons.providerAuthentication")
    expect(wrapper.text()).not.toContain("sk-live-secret")
    expect(wrapper.text()).toContain("agentWindow.outcomeRetryable")
  })

  it("keeps an empty model answer distinct from a cancelled request", () => {
    const wrapper = mount(AgentChatOutcome, {
      props: { outcome: { status: "failed", reasonCode: "provider_empty", retryable: true } },
    })
    expect(wrapper.text()).toContain("agentWindow.outcome.failed")
    expect(wrapper.text()).toContain("agentWindow.outcomeReasons.providerEmpty")
  })
})
