import { mount } from "@vue/test-utils"
import { defineComponent, nextTick } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import { useGamepad } from "@/composables/use-gamepad"

function makeButton(pressed = false): GamepadButton {
  return {
    pressed,
    touched: pressed,
    value: pressed ? 1 : 0,
  } as GamepadButton
}

function makeGamepad(options: {
  id?: string
  buttons?: GamepadButton[]
  axes?: number[]
  vibrationActuator?: Partial<GamepadHapticActuator>
} = {}): Gamepad {
  return {
    axes: options.axes ?? [0, 0, 0, 0],
    buttons: options.buttons ?? Array.from({ length: 18 }, () => makeButton(false)),
    connected: true,
    id: options.id ?? "DualSense Wireless Controller",
    index: 0,
    mapping: "standard",
    timestamp: 1,
    vibrationActuator: options.vibrationActuator,
  } as Gamepad
}

describe("useGamepad", () => {
  let gamepads: Array<Gamepad | null> = []
  let rafCallbacks: FrameRequestCallback[] = []

  function installBrowserGamepadMocks() {
    Object.defineProperty(navigator, "getGamepads", {
      configurable: true,
      value: vi.fn(() => gamepads),
    })
    Object.defineProperty(window, "requestAnimationFrame", {
      configurable: true,
      value: vi.fn((callback: FrameRequestCallback) => {
        rafCallbacks.push(callback)
        return rafCallbacks.length
      }),
    })
    Object.defineProperty(window, "cancelAnimationFrame", {
      configurable: true,
      value: vi.fn(),
    })
  }

  async function runNextFrame(nowMs: number) {
    const callback = rafCallbacks.shift()
    if (!callback) {
      throw new Error("expected a queued animation frame")
    }
    callback(nowMs)
    await nextTick()
  }

  afterEach(() => {
    vi.restoreAllMocks()
    gamepads = []
    rafCallbacks = []
  })

  it("tracks the first connected standard gamepad", async () => {
    installBrowserGamepadMocks()
    gamepads = [makeGamepad({ id: "DualSense Edge" })]

    const Harness = defineComponent({
      setup() {
        return useGamepad()
      },
      template: "<p>{{ connected }} {{ gamepadId }}</p>",
    })

    const wrapper = mount(Harness)
    await runNextFrame(1000)

    expect(wrapper.text()).toContain("true DualSense Edge")
  })

  it("fires button press handlers once per press edge", async () => {
    installBrowserGamepadMocks()
    const buttons = Array.from({ length: 18 }, () => makeButton(false))
    gamepads = [makeGamepad({ buttons })]
    const onCross = vi.fn()

    const Harness = defineComponent({
      setup() {
        const gamepad = useGamepad()
        gamepad.onButtonPress("cross", onCross)
        return () => null
      },
    })

    mount(Harness)
    buttons[0] = makeButton(true)
    await runNextFrame(1000)
    await runNextFrame(1016)

    buttons[0] = makeButton(false)
    await runNextFrame(1032)
    buttons[0] = makeButton(true)
    await runNextFrame(1048)

    expect(onCross).toHaveBeenCalledTimes(2)
  })

  it("repeats held directions after the configured delay", async () => {
    installBrowserGamepadMocks()
    const buttons = Array.from({ length: 18 }, () => makeButton(false))
    buttons[15] = makeButton(true)
    gamepads = [makeGamepad({ buttons })]
    const onRight = vi.fn()

    const Harness = defineComponent({
      setup() {
        const gamepad = useGamepad({ repeatInitialDelayMs: 240, repeatMs: 90 })
        gamepad.onDirectionPress("right", onRight)
        return () => null
      },
    })

    mount(Harness)
    await runNextFrame(1000)
    await runNextFrame(1100)
    await runNextFrame(1240)
    await runNextFrame(1280)
    await runNextFrame(1330)

    expect(onRight).toHaveBeenCalledTimes(3)
  })

  it("silently skips rumble when the connected gamepad has no haptic actuator", async () => {
    installBrowserGamepadMocks()
    gamepads = [makeGamepad()]
    let rumbleResult: Promise<void> | null = null

    const Harness = defineComponent({
      setup() {
        const gamepad = useGamepad()
        rumbleResult = gamepad.rumble({ duration: 20 })
        return () => null
      },
    })

    mount(Harness)

    await expect(rumbleResult).resolves.toBeUndefined()
  })
})
