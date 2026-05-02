import type { GamepadDirection } from "@/lib/gamepad/standard-gamepad"
import { DEFAULT_GAMEPAD_DEADZONE } from "@/lib/gamepad/standard-gamepad"

export interface ButtonEdgeDiff {
  pressed: number[]
  released: number[]
}

export interface RepeatGateOptions {
  initialDelayMs: number
  repeatMs: number
}

interface RepeatGateState {
  direction: GamepadDirection
  firstFiredAt: number
  lastFiredAt: number
}

export interface RepeatGate {
  shouldFire(direction: GamepadDirection | null, nowMs: number): boolean
  reset(direction?: GamepadDirection): void
}

export type DualRumbleGamepad = Gamepad & {
  vibrationActuator: {
    playEffect: (
      type: "dual-rumble",
      params?: GamepadEffectParameters,
    ) => Promise<GamepadHapticsResult>
  }
}

export function applyDeadzone(
  value: number,
  threshold = DEFAULT_GAMEPAD_DEADZONE,
): number {
  if (!Number.isFinite(value)) return 0
  const safeThreshold = Math.max(0, Math.min(0.99, threshold))
  return Math.abs(value) < safeThreshold ? 0 : value
}

export function directionFromAxes(
  x: number,
  y: number,
  threshold = DEFAULT_GAMEPAD_DEADZONE,
): GamepadDirection | null {
  const safeX = applyDeadzone(x, threshold)
  const safeY = applyDeadzone(y, threshold)
  if (safeX === 0 && safeY === 0) return null
  if (Math.abs(safeX) >= Math.abs(safeY)) {
    return safeX > 0 ? "right" : "left"
  }
  return safeY > 0 ? "down" : "up"
}

export function diffButtonEdges(
  previous: readonly boolean[],
  current: readonly boolean[],
): ButtonEdgeDiff {
  const maxLength = Math.max(previous.length, current.length)
  const pressed: number[] = []
  const released: number[] = []

  for (let index = 0; index < maxLength; index++) {
    const wasPressed = previous[index] === true
    const isPressed = current[index] === true
    if (!wasPressed && isPressed) {
      pressed.push(index)
    } else if (wasPressed && !isPressed) {
      released.push(index)
    }
  }

  return { pressed, released }
}

export function createRepeatGate(options: RepeatGateOptions): RepeatGate {
  let state: RepeatGateState | null = null
  const initialDelayMs = Math.max(0, options.initialDelayMs)
  const repeatMs = Math.max(1, options.repeatMs)

  return {
    shouldFire(direction, nowMs) {
      if (!direction) {
        state = null
        return false
      }

      if (!state || state.direction !== direction) {
        state = {
          direction,
          firstFiredAt: nowMs,
          lastFiredAt: nowMs,
        }
        return true
      }

      const elapsedSinceFirst = nowMs - state.firstFiredAt
      const elapsedSinceLast = nowMs - state.lastFiredAt
      if (elapsedSinceFirst >= initialDelayMs && elapsedSinceLast >= repeatMs) {
        state = {
          ...state,
          lastFiredAt: nowMs,
        }
        return true
      }

      return false
    },
    reset(direction) {
      if (!direction || state?.direction === direction) {
        state = null
      }
    },
  }
}

export function supportsDualRumble(gamepad: unknown): gamepad is DualRumbleGamepad {
  const actuator = (gamepad as Partial<DualRumbleGamepad> | null | undefined)?.vibrationActuator
  return typeof actuator?.playEffect === "function"
}
