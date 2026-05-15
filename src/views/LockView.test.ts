import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import LockView from "./LockView.vue"
import { authLockService } from "@/services/auth-lock-service"

const replace = vi.fn()
const authMock = vi.hoisted(() => ({
  status: {
    value: {
      pinEnabled: true,
      unlocked: false,
      setupRequired: false,
      pinLength: 4,
      trustedForever: false,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    },
  },
  unlock: vi.fn(),
}))

vi.mock("vue-router", () => ({
  useRoute: () => ({
    query: {
      redirect: "/library?actor=Mina",
    },
  }),
  useRouter: () => ({
    replace,
  }),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/services/auth-lock-service", () => ({
  authLockService: authMock,
}))

vi.mock("lucide-vue-next", () => ({
  LockKeyhole: { name: "LockKeyhole", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["type", "disabled"],
    template: "<button :type=\"type\" :disabled=\"disabled\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section data-card><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h1><slot /></h1>" },
}))

vi.mock("@/components/ui/switch", () => ({
  Switch: {
    name: "Switch",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: `
      <button
        type="button"
        data-trust-forever
        :aria-checked="modelValue"
        @click="$emit('update:modelValue', !modelValue)"
      />
    `,
  },
}))

describe("LockView", () => {
  beforeEach(() => {
    replace.mockReset()
    authMock.unlock.mockReset()
    authMock.status.value = {
      pinEnabled: true,
      unlocked: false,
      setupRequired: false,
      pinLength: 4,
      trustedForever: false,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    }
  })

  it("renders configured PIN cells without keypad or background image", () => {
    const wrapper = mount(LockView)

    expect(wrapper.findAll("[data-pin-cell]")).toHaveLength(4)
    expect(wrapper.find("[data-pin-keypad]").exists()).toBe(false)
    expect(wrapper.find("img").exists()).toBe(false)
    expect(wrapper.html()).not.toContain("background-image")
    expect(wrapper.html()).not.toContain("url(")
  })

  it("shows local-only recovery guidance when asked", async () => {
    const wrapper = mount(LockView)

    await wrapper.get("[data-forgot-pin]").trigger("click")

    expect(wrapper.text()).toContain("lock.forgotPinHint")
  })

  it("submits keyboard-entered PIN with permanent device trust", async () => {
    vi.mocked(authLockService.unlock).mockResolvedValueOnce({
      pinEnabled: true,
      unlocked: true,
      setupRequired: false,
      pinLength: 4,
      trustedForever: true,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    })
    const wrapper = mount(LockView)

    await wrapper.get("[data-pin-input]").setValue("123456")
    await wrapper.get("[data-trust-forever]").trigger("click")
    await wrapper.get("form").trigger("submit.prevent")

    expect(authLockService.unlock).toHaveBeenCalledWith({
      pin: "123456",
      trustedForever: true,
    })
    expect(replace).toHaveBeenCalledWith("/library?actor=Mina")
  })
})
