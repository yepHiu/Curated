import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsSecuritySection from "./SettingsSecuritySection.vue"

const authMock = vi.hoisted(() => ({
  status: {
    value: {
      pinEnabled: false,
      unlocked: true,
      setupRequired: true,
      pinLength: 0,
      trustedForever: false,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    },
  },
  refreshStatus: vi.fn(),
  setupPin: vi.fn(),
  changePin: vi.fn(),
  lock: vi.fn(),
  patchSettings: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/services/auth-lock-service", () => ({
  authLockService: authMock,
  isAuthLockEnabled: () => true,
}))

vi.mock("lucide-vue-next", () => ({
  LockKeyhole: { name: "LockKeyhole", template: "<span />" },
  ShieldCheck: { name: "ShieldCheck", template: "<span />" },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section data-card :class=\"$attrs.class\"><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div data-card-content :class=\"$attrs.class\"><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<p data-card-description :class=\"$attrs.class\"><slot /></p>" },
  CardHeader: { name: "CardHeader", template: "<header data-card-header :class=\"$attrs.class\"><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h3 data-card-title :class=\"$attrs.class\"><slot /></h3>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["type", "disabled", "variant", "size"],
    template: "<button v-bind=\"$attrs\" :type=\"type\" :disabled=\"disabled\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: {
    name: "Dialog",
    props: ["open"],
    template: "<div v-if=\"open\" data-dialog><slot /></div>",
  },
  DialogContent: {
    name: "DialogContent",
    template: "<section data-dialog-content><slot /></section>",
  },
  DialogDescription: {
    name: "DialogDescription",
    template: "<p data-dialog-description><slot /></p>",
  },
  DialogFooter: {
    name: "DialogFooter",
    template: "<footer data-dialog-footer><slot /></footer>",
  },
  DialogHeader: {
    name: "DialogHeader",
    template: "<header data-dialog-header><slot /></header>",
  },
  DialogTitle: {
    name: "DialogTitle",
    template: "<h2 data-dialog-title><slot /></h2>",
  },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue", "type", "placeholder"],
    emits: ["update:modelValue"],
    template:
      "<input v-bind=\"$attrs\" :type=\"type\" :placeholder=\"placeholder\" :value=\"modelValue\" @input=\"$emit('update:modelValue', $event.target.value)\">",
  },
}))

vi.mock("@/components/ui/switch", () => ({
  Switch: {
    name: "Switch",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<button class=\"switch-stub\" :disabled=\"disabled\" @click=\"$emit('update:modelValue', !modelValue)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/select", () => {
  const Select = {
    name: "Select",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<div data-select :data-model-value=\"modelValue\"><slot /></div>",
  }
  return {
    Select,
    SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
    SelectGroup: { name: "SelectGroup", template: "<div><slot /></div>" },
    SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
    SelectTrigger: { name: "SelectTrigger", template: "<button><slot /></button>" },
    SelectValue: { name: "SelectValue", template: "<span />" },
  }
})

describe("SettingsSecuritySection", () => {
  beforeEach(() => {
    authMock.status.value = {
      pinEnabled: false,
      unlocked: true,
      setupRequired: true,
      pinLength: 0,
      trustedForever: false,
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    }
    authMock.refreshStatus.mockReset()
    authMock.setupPin.mockReset()
    authMock.changePin.mockReset()
    authMock.lock.mockReset()
    authMock.patchSettings.mockReset()
  })

  it("uses the settings card and nested block layout contract", () => {
    const wrapper = mount(SettingsSecuritySection)

    expect(wrapper.get("[data-card]").classes()).toContain("gap-2")
    expect(wrapper.get("[data-card-header]").classes()).toEqual(
      expect.arrayContaining(["grid", "grid-cols-[auto_minmax(0,1fr)]", "pb-0"]),
    )
    expect(wrapper.get("[data-card-title]").classes()).toEqual(
      expect.arrayContaining(["min-w-0", "text-lg", "tracking-tight"]),
    )
    expect(wrapper.get("[data-card-content]").classes()).toEqual(
      expect.arrayContaining(["flex", "flex-col", "gap-3", "pt-0"]),
    )
    expect(wrapper.findAll("[data-security-block]")).toHaveLength(5)
  })

  it("renders PIN setup entry, session TTL, LAN PIN, and lock-now controls", () => {
    const wrapper = mount(SettingsSecuritySection)

    expect(wrapper.text()).toContain("settings.securityTitle")
    expect(wrapper.text()).toContain("settings.securitySetupTitle")
    expect(wrapper.text()).toContain("settings.securityEnablePin")
    expect(wrapper.text()).toContain("settings.securitySessionTitle")
    expect(wrapper.text()).toContain("settings.securityLanRequiresPin")
    expect(wrapper.text()).toContain("settings.securityLockNow")
    expect(wrapper.text()).toContain("settings.securityTrustDeviceTitle")
    expect(wrapper.text()).toContain("settings.securityLanPolicyTitle")
    expect(wrapper.find("[data-setup-pin-trigger]").exists()).toBe(true)
    expect(wrapper.findAll("input[type='password']")).toHaveLength(0)
  })

  it("opens a dialog before submitting a new PIN setup", async () => {
    authMock.setupPin.mockResolvedValueOnce({
      ...authMock.status.value,
      pinEnabled: true,
      setupRequired: false,
      pinLength: 4,
    })
    const wrapper = mount(SettingsSecuritySection)

    expect(wrapper.find("[data-setup-pin-form]").exists()).toBe(false)
    expect(wrapper.findAll("input[type='password']")).toHaveLength(0)

    await wrapper.get("[data-setup-pin-trigger]").trigger("click")

    expect(wrapper.find("[data-setup-pin-form]").exists()).toBe(true)
    await wrapper.get("[data-setup-pin-input]").setValue("1234")
    await wrapper.get("[data-confirm-pin-input]").setValue("1234")
    await wrapper.get("[data-setup-pin-form]").trigger("submit.prevent")

    expect(authMock.setupPin).toHaveBeenCalledWith({
      pin: "1234",
      confirmPin: "1234",
      sessionTtlMinutes: 60,
      lanRequiresPin: true,
      lockOnRestart: true,
    })
    expect(wrapper.text()).toContain("settings.securitySetupSaved")
  })

  it("opens a dialog before submitting current and new PIN values", async () => {
    authMock.status.value = {
      ...authMock.status.value,
      pinEnabled: true,
      pinLength: 4,
    }
    authMock.changePin.mockResolvedValueOnce({
      ...authMock.status.value,
      pinLength: 5,
    })
    const wrapper = mount(SettingsSecuritySection)

    expect(wrapper.find("[data-change-pin-form]").exists()).toBe(false)
    expect(wrapper.find("[data-change-pin-trigger]").exists()).toBe(true)
    expect(wrapper.findAll("input[type='password']")).toHaveLength(0)

    await wrapper.get("[data-change-pin-trigger]").trigger("click")

    expect(wrapper.find("[data-change-pin-form]").exists()).toBe(true)
    await wrapper.get("[data-current-pin-input]").setValue("1234")
    await wrapper.get("[data-new-pin-input]").setValue("98765")
    await wrapper.get("[data-confirm-new-pin-input]").setValue("98765")
    await wrapper.get("[data-change-pin-form]").trigger("submit.prevent")

    expect(authMock.changePin).toHaveBeenCalledWith({
      currentPin: "1234",
      newPin: "98765",
      confirmPin: "98765",
    })
    expect(wrapper.text()).toContain("settings.securityPinChanged")
  })
})
