import { describe, expect, it } from "vitest"
import {
  applyDeadzone,
  createRepeatGate,
  diffButtonEdges,
  directionFromAxes,
  supportsDualRumble,
} from "@/lib/gamepad/gamepad-input"

describe("gamepad input utilities", () => {
  it("filters small axis drift inside the deadzone", () => {
    expect(applyDeadzone(0.1, 0.18)).toBe(0)
    expect(applyDeadzone(-0.12, 0.18)).toBe(0)
  })

  it("keeps axis values outside the deadzone", () => {
    expect(applyDeadzone(0.5, 0.18)).toBeCloseTo(0.5)
    expect(applyDeadzone(-0.5, 0.18)).toBeCloseTo(-0.5)
  })

  it("resolves the dominant joystick direction", () => {
    expect(directionFromAxes(0.72, 0.22, 0.18)).toBe("right")
    expect(directionFromAxes(-0.2, -0.66, 0.18)).toBe("up")
    expect(directionFromAxes(0.05, -0.08, 0.18)).toBeNull()
  })

  it("reports button press and release edges once per state transition", () => {
    const previous = [false, false, true]
    const current = [false, true, false]

    expect(diffButtonEdges(previous, current)).toEqual({
      pressed: [1],
      released: [2],
    })
    expect(diffButtonEdges(current, current)).toEqual({
      pressed: [],
      released: [],
    })
  })

  it("allows initial and repeated directional actions after the configured delays", () => {
    const gate = createRepeatGate({ initialDelayMs: 240, repeatMs: 90 })

    expect(gate.shouldFire("right", 1000)).toBe(true)
    expect(gate.shouldFire("right", 1100)).toBe(false)
    expect(gate.shouldFire("right", 1240)).toBe(true)
    expect(gate.shouldFire("right", 1280)).toBe(false)
    expect(gate.shouldFire("right", 1330)).toBe(true)
    expect(gate.shouldFire("left", 1340)).toBe(true)

    gate.reset("left")
    expect(gate.shouldFire("left", 1350)).toBe(true)
  })

  it("detects dual-rumble support without throwing for missing browser capabilities", () => {
    const unsupported = {}
    expect(supportsDualRumble(unsupported)).toBe(false)

    const supported = {
      vibrationActuator: {
        playEffect: () => Promise.resolve("complete" as GamepadHapticsResult),
      },
    }
    expect(supportsDualRumble(supported)).toBe(true)
  })
})
