import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsPlaybackSection from "./SettingsPlaybackSection.vue"

const mocks = vi.hoisted(() => ({
  setGamepadControlsEnabled: vi.fn(),
  patchPlayerSettings: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  PlayCircle: { name: "PlayCircle", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled"],
    emits: ["click"],
    template: "<button :disabled=\"disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<div><slot /></div>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<input :value=\"modelValue\" @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
}))

vi.mock("@/components/ui/select", () => {
  const Select = {
    name: "Select",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template: "<div class=\"select-stub\"><slot /></div>",
  }
  return {
    Select,
    SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
    SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
    SelectTrigger: { name: "SelectTrigger", template: "<div><slot /></div>" },
    SelectValue: { name: "SelectValue", props: ["placeholder"], template: "<span>{{ placeholder }}</span>" },
  }
})

vi.mock("@/components/ui/switch", () => ({
  Switch: {
    name: "Switch",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<button v-bind=\"$attrs\" class=\"switch-stub\" :disabled=\"disabled\" @click=\"$emit('update:modelValue', !modelValue)\"><slot /></button>",
  },
}))

vi.mock("@/composables/use-settings-scroll-preserve", () => ({
  useSettingsScrollPreserve: () => ({
    withPreservedScroll: async (fn: () => Promise<void>) => {
      await fn()
    },
  }),
}))

vi.mock("@/lib/gamepad/gamepad-settings", () => ({
  useGamepadControlsPreference: () => ({
    gamepadControlsEnabled: { value: true },
    setGamepadControlsEnabled: mocks.setGamepadControlsEnabled,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    playerSettings: {
      value: {
        hardwareDecode: true,
        hardwareEncoder: "auto",
        nativePlayerPreset: "custom",
        nativePlayerEnabled: false,
        nativePlayerCommand: "",
        streamPushEnabled: true,
        forceStreamPush: false,
        ffmpegCommand: "ffmpeg",
        preferNativePlayer: false,
        seekForwardStepSec: 10,
        seekBackwardStepSec: 10,
      },
    },
    patchPlayerSettings: mocks.patchPlayerSettings,
  }),
}))

describe("SettingsPlaybackSection gamepad controls", () => {
  it("renders and persists the gamepad controls toggle", async () => {
    const wrapper = mount(SettingsPlaybackSection, {
      props: { autoSaveReady: false },
    })

    expect(wrapper.text()).toContain("settings.gamepadControlsTitle")
    expect(wrapper.text()).toContain("settings.gamepadControlsHint")

    await wrapper.get("[data-gamepad-controls-toggle]").trigger("click")

    expect(mocks.setGamepadControlsEnabled).toHaveBeenCalledWith(false)
  })
})
