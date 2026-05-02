export const STANDARD_GAMEPAD_BUTTON = {
  cross: 0,
  circle: 1,
  square: 2,
  triangle: 3,
  l1: 4,
  r1: 5,
  l2: 6,
  r2: 7,
  share: 8,
  options: 9,
  l3: 10,
  r3: 11,
  dpadUp: 12,
  dpadDown: 13,
  dpadLeft: 14,
  dpadRight: 15,
  psOrHome: 16,
  touchpad: 17,
} as const

export const STANDARD_GAMEPAD_AXIS = {
  leftX: 0,
  leftY: 1,
  rightX: 2,
  rightY: 3,
} as const

export type StandardGamepadButtonName = keyof typeof STANDARD_GAMEPAD_BUTTON
export type StandardGamepadAxisName = keyof typeof STANDARD_GAMEPAD_AXIS
export type GamepadDirection = "up" | "down" | "left" | "right"

export const DEFAULT_GAMEPAD_DEADZONE = 0.18
export const DEFAULT_GAMEPAD_REPEAT_INITIAL_DELAY_MS = 240
export const DEFAULT_GAMEPAD_REPEAT_MS = 90
