import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => {
      const messages: Record<string, string> = {
        "player.playbackSettings": "Playback Settings",
        "player.playbackSettingsAria": "Open playback settings",
        "player.playbackSpeed": "Playback Speed",
        "player.playbackMode": "Playback Mode",
        "player.playbackModeDirect": "Direct",
        "player.playbackModeHls": "HLS",
        "player.playbackModeDirectUnavailable": "Direct unavailable",
      }
      return messages[key] ?? key
    },
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Settings2: { name: "Settings2", template: "<span data-icon='settings' />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled", "ariaLabel", "ariaPressed", "variant", "size", "type"],
    emits: ["click"],
    template:
      "<button :disabled='disabled' :aria-label='ariaLabel' :aria-pressed='ariaPressed' :data-variant='variant' :data-size='size' :type='type || \"button\"' @click='$emit(\"click\", $event)'><slot /></button>",
  },
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div data-dropdown-menu><slot /></div>" },
  DropdownMenuTrigger: { name: "DropdownMenuTrigger", template: "<div data-dropdown-trigger><slot /></div>" },
  DropdownMenuContent: {
    name: "DropdownMenuContent",
    template: "<div data-dropdown-content :class='$attrs.class'><slot /></div>",
  },
  DropdownMenuGroup: {
    name: "DropdownMenuGroup",
    template: "<div data-dropdown-group><slot /></div>",
  },
  DropdownMenuLabel: {
    name: "DropdownMenuLabel",
    template: "<div data-dropdown-label :class='$attrs.class'><slot /></div>",
  },
  DropdownMenuSeparator: {
    name: "DropdownMenuSeparator",
    template: "<hr data-dropdown-separator :class='$attrs.class' />",
  },
  DropdownMenuRadioGroup: {
    name: "DropdownMenuRadioGroup",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<div :data-model-value='modelValue'><slot /></div>",
  },
  DropdownMenuRadioItem: {
    name: "DropdownMenuRadioItem",
    props: ["value", "disabled"],
    emits: ["select"],
    template:
      "<button :disabled='disabled' :data-radio-value='value' :class='$attrs.class' @click='$emit(\"select\", $event)'><slot /></button>",
  },
}))

async function mountComponent(props?: Record<string, unknown>) {
  const mod = await import("./PlayerPlaybackSettingsMenu.vue")
  return mount(mod.default, {
    props: {
      disabled: false,
      playbackRate: 1,
      playbackMode: "direct",
      canSwitchToDirect: true,
      switchingMode: false,
      ...props,
    },
  })
}

describe("PlayerPlaybackSettingsMenu", () => {
  it("renders the trigger and grouped playback settings options", async () => {
    const wrapper = await mountComponent()

    expect(wrapper.find('button[aria-label="Open playback settings"]').exists()).toBe(true)
    expect(wrapper.text()).toContain("Playback Settings")
    expect(wrapper.text()).toContain("Playback Speed")
    expect(wrapper.text()).toContain("Playback Mode")
    expect(wrapper.text()).toContain("0.75x")
    expect(wrapper.text()).toContain("1x")
    expect(wrapper.text()).toContain("1.25x")
    expect(wrapper.text()).toContain("1.5x")
    expect(wrapper.text()).toContain("2x")
    expect(wrapper.text()).toContain("Direct")
    expect(wrapper.text()).toContain("HLS")
  })

  it("emits playback-rate and playback-mode updates when an option is clicked", async () => {
    const wrapper = await mountComponent({
      playbackRate: 1,
      playbackMode: "hls",
    })

    const rateButton = wrapper.find('[data-radio-value="1.5"]')
    const directButton = wrapper.find('[data-radio-value="direct"]')

    await rateButton.trigger("click")
    await directButton.trigger("click")

    expect(wrapper.emitted("update:playbackRate")).toEqual([[1.5]])
    expect(wrapper.emitted("update:playbackMode")).toEqual([["direct"]])
  })

  it("disables the direct option when direct playback is unavailable", async () => {
    const wrapper = await mountComponent({
      canSwitchToDirect: false,
      playbackMode: "hls",
    })

    const directButton = wrapper.get('[data-radio-value="direct"]')

    expect(directButton.attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("Direct unavailable")
  })

  it("uses a high-contrast menu surface and non-overlapping checked-state styling", async () => {
    const wrapper = await mountComponent()

    const content = wrapper.get("[data-dropdown-content]")
    const hlsButton = wrapper.get('[data-radio-value="hls"]')

    expect(content.classes()).toContain("backdrop-blur-xl")
    expect(content.classes()).toContain("bg-neutral-950/95")
    expect(content.classes()).not.toContain("bg-white/8")
    expect(hlsButton.classes()).toContain("data-[state=checked]:bg-primary/18")
    expect(hlsButton.classes()).toContain("data-[state=checked]:text-primary")
    expect(hlsButton.classes()).toContain("data-[state=checked]:focus:bg-primary/22")
    expect(hlsButton.classes()).not.toContain("data-[state=checked]:ring-1")
  })
})
