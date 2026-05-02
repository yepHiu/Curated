import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { GamepadDirection, StandardGamepadButtonName } from "@/lib/gamepad/standard-gamepad"
import { usePlayerGamepadControls } from "@/composables/use-player-gamepad-controls"

const mocks = vi.hoisted(() => ({
  useGamepad: vi.fn(),
  rumble: vi.fn(),
  buttonHandlers: new Map<string | number, () => void>(),
  directionHandlers: new Map<GamepadDirection, () => void>(),
}))

vi.mock("@/composables/use-gamepad", () => ({
  useGamepad: mocks.useGamepad,
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

describe("usePlayerGamepadControls", () => {
  beforeEach(() => {
    mocks.buttonHandlers.clear()
    mocks.directionHandlers.clear()
    mocks.rumble.mockReset()
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
      rumble: mocks.rumble,
      stop: vi.fn(),
    })
  })

  it("maps primary player buttons to existing player actions", () => {
    const actions = {
      togglePlayPause: vi.fn(),
      exitFullscreen: vi.fn(),
      exitPlayer: vi.fn(),
      runCuratedCapture: vi.fn(),
      toggleDetailedStats: vi.fn(),
      toggleChrome: vi.fn(),
      toggleMute: vi.fn(),
    }

    usePlayerGamepadControls({
      enabled: true,
      isFullscreen: ref(false),
      seekBackwardStepSec: ref(10),
      seekForwardStepSec: ref(30),
      actions,
    })

    pressButton("cross")
    pressButton("circle")
    pressButton("square")
    pressButton("triangle")
    pressButton("options")
    pressButton("psOrHome")

    expect(actions.togglePlayPause).toHaveBeenCalledTimes(1)
    expect(actions.exitPlayer).toHaveBeenCalledTimes(1)
    expect(actions.exitFullscreen).not.toHaveBeenCalled()
    expect(actions.runCuratedCapture).toHaveBeenCalledTimes(1)
    expect(actions.toggleDetailedStats).toHaveBeenCalledTimes(1)
    expect(actions.toggleChrome).toHaveBeenCalledTimes(1)
    expect(actions.toggleMute).toHaveBeenCalledTimes(1)
  })

  it("uses Circle to exit fullscreen before leaving the player", () => {
    const actions = {
      togglePlayPause: vi.fn(),
      exitFullscreen: vi.fn(),
      exitPlayer: vi.fn(),
    }

    usePlayerGamepadControls({
      enabled: true,
      isFullscreen: ref(true),
      seekBackwardStepSec: ref(10),
      seekForwardStepSec: ref(30),
      actions,
    })

    pressButton("circle")

    expect(actions.exitFullscreen).toHaveBeenCalledTimes(1)
    expect(actions.exitPlayer).not.toHaveBeenCalled()
  })

  it("maps directions to seek and volume actions using the configured seek steps", () => {
    const actions = {
      togglePlayPause: vi.fn(),
      seekDelta: vi.fn(),
      adjustVolume: vi.fn(),
    }

    usePlayerGamepadControls({
      enabled: true,
      seekBackwardStepSec: ref(12),
      seekForwardStepSec: ref(18),
      actions,
    })

    pressDirection("left")
    pressDirection("right")
    pressDirection("up")
    pressDirection("down")

    expect(actions.seekDelta).toHaveBeenNthCalledWith(1, -12)
    expect(actions.seekDelta).toHaveBeenNthCalledWith(2, 18)
    expect(actions.adjustVolume).toHaveBeenNthCalledWith(1, 5)
    expect(actions.adjustVolume).toHaveBeenNthCalledWith(2, -5)
  })

  it("maps L1 and R1 to larger seek jumps", () => {
    const actions = {
      togglePlayPause: vi.fn(),
      seekDelta: vi.fn(),
    }

    usePlayerGamepadControls({
      enabled: true,
      seekBackwardStepSec: ref(12),
      seekForwardStepSec: ref(18),
      actions,
    })

    pressButton("l1")
    pressButton("r1")

    expect(actions.seekDelta).toHaveBeenNthCalledWith(1, -36)
    expect(actions.seekDelta).toHaveBeenNthCalledWith(2, 54)
  })

  it("passes the enabled flag through to the gamepad input layer", () => {
    const enabled = ref(false)

    usePlayerGamepadControls({
      enabled,
      seekBackwardStepSec: ref(10),
      seekForwardStepSec: ref(30),
      actions: {
        togglePlayPause: vi.fn(),
      },
    })

    expect(mocks.useGamepad).toHaveBeenCalledWith(expect.objectContaining({ enabled }))
  })
})
