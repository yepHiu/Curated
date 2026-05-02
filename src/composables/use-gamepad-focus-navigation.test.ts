import { mount } from "@vue/test-utils"
import { defineComponent, nextTick, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { GamepadDirection, StandardGamepadButtonName } from "@/lib/gamepad/standard-gamepad"
import { useGamepadFocusNavigation } from "@/composables/use-gamepad-focus-navigation"

const mocks = vi.hoisted(() => ({
  useGamepad: vi.fn(),
  buttonHandlers: new Map<string | number, () => void>(),
  directionHandlers: new Map<GamepadDirection, () => void>(),
  route: {
    name: "library",
  },
  routerBack: vi.fn(),
  routerPush: vi.fn(),
  gamepadControlsEnabled: { value: true, __v_isRef: true },
}))

vi.mock("@/composables/use-gamepad", () => ({
  useGamepad: mocks.useGamepad,
}))

vi.mock("@/lib/gamepad/gamepad-settings", () => ({
  useGamepadControlsPreference: () => ({
    gamepadControlsEnabled: mocks.gamepadControlsEnabled,
    setGamepadControlsEnabled: vi.fn(),
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({
    back: mocks.routerBack,
    push: mocks.routerPush,
  }),
}))

function pressButton(button: StandardGamepadButtonName) {
  const handler = mocks.buttonHandlers.get(button)
  if (!handler) throw new Error(`missing button handler for ${button}`)
  handler()
}

function pressDirection(direction: GamepadDirection) {
  const handler = mocks.directionHandlers.get(direction)
  if (!handler) throw new Error(`missing direction handler for ${direction}`)
  handler()
}

function mockRect(
  el: Element,
  rect: { left: number; top: number; width: number; height: number },
) {
  Object.defineProperty(el, "getBoundingClientRect", {
    configurable: true,
    value: () => ({
      ...rect,
      right: rect.left + rect.width,
      bottom: rect.top + rect.height,
      x: rect.left,
      y: rect.top,
      toJSON: () => ({}),
    }),
  })
}

function mountHarness(onRightClick = vi.fn()) {
  const Harness = defineComponent({
    setup() {
      useGamepadFocusNavigation()
      return { onRightClick }
    },
    template: `
      <div>
        <button id="left">Left</button>
        <button id="right" @click="onRightClick">Right</button>
      </div>
    `,
  })

  const wrapper = mount(Harness, { attachTo: document.body })
  mockRect(wrapper.get("#left").element, { left: 450, top: 380, width: 40, height: 40 })
  mockRect(wrapper.get("#right").element, { left: 540, top: 380, width: 40, height: 40 })
  return wrapper
}

describe("useGamepadFocusNavigation", () => {
  beforeEach(() => {
    document.body.innerHTML = ""
    mocks.buttonHandlers.clear()
    mocks.directionHandlers.clear()
    mocks.routerBack.mockReset()
    mocks.routerPush.mockReset()
    mocks.gamepadControlsEnabled.value = true
    mocks.route.name = "library"
    mocks.useGamepad.mockReset()
    mocks.useGamepad.mockReturnValue({
      connected: ref(true),
      gamepadId: ref("DualSense"),
      lastInputAt: ref(0),
      supported: ref(true),
      onButtonPress: vi.fn((button: string | number, handler: () => void) => {
        mocks.buttonHandlers.set(button, handler)
        return vi.fn()
      }),
      onDirectionPress: vi.fn((direction: GamepadDirection, handler: () => void) => {
        mocks.directionHandlers.set(direction, handler)
        return vi.fn()
      }),
      rumble: vi.fn(),
      stop: vi.fn(),
    })
  })

  it("moves controller focus through visible DOM controls and clicks the focused element", async () => {
    const onRightClick = vi.fn()
    const wrapper = mountHarness(onRightClick)

    pressDirection("right")
    await nextTick()
    expect(wrapper.get("#left").attributes("data-controller-focused")).toBe("true")

    pressDirection("right")
    await nextTick()
    expect(wrapper.get("#right").attributes("data-controller-focused")).toBe("true")

    pressButton("cross")

    expect(onRightClick).toHaveBeenCalledTimes(1)
  })

  it("clears controller focus when pointer input takes over", async () => {
    const wrapper = mountHarness()

    pressDirection("right")
    await nextTick()
    expect(wrapper.get("#left").attributes("data-controller-focused")).toBe("true")

    window.dispatchEvent(new Event("pointerdown"))
    await nextTick()

    expect(wrapper.get("#left").attributes("data-controller-focused")).toBeUndefined()
  })

  it("leaves directional focus to specialized library grid navigation", async () => {
    const Harness = defineComponent({
      setup() {
        useGamepadFocusNavigation()
        return {}
      },
      template: `
        <div data-gamepad-grid-navigation="library">
          <button id="left">Left</button>
          <button id="right">Right</button>
        </div>
      `,
    })
    const wrapper = mount(Harness, { attachTo: document.body })
    mockRect(wrapper.get("#left").element, { left: 450, top: 380, width: 40, height: 40 })
    mockRect(wrapper.get("#right").element, { left: 540, top: 380, width: 40, height: 40 })

    pressDirection("right")
    await nextTick()

    expect(wrapper.get("#left").attributes("data-controller-focused")).toBeUndefined()
    expect(wrapper.get("#right").attributes("data-controller-focused")).toBeUndefined()
  })

  it("leaves Cross to specialized library grid navigation", () => {
    const onClick = vi.fn()
    const Harness = defineComponent({
      setup() {
        useGamepadFocusNavigation()
        return { onClick }
      },
      template: `
        <div data-gamepad-grid-navigation="library">
          <button id="left" data-controller-focused="true" @click="onClick">Left</button>
        </div>
      `,
    })
    mount(Harness, { attachTo: document.body })

    pressButton("cross")

    expect(onClick).not.toHaveBeenCalled()
  })

  it("disables global navigation on the player route", () => {
    mocks.route.name = "player"

    mountHarness()

    expect(mocks.useGamepad).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: expect.objectContaining({ value: false }),
      }),
    )
  })

  it("maps navigation buttons to browser back and primary route switching", () => {
    mountHarness()

    pressButton("circle")
    pressButton("r1")

    expect(mocks.routerBack).toHaveBeenCalledTimes(1)
    expect(mocks.routerPush).toHaveBeenCalledWith({ name: "favorites" })
  })
})
