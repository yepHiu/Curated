import { computed, isRef, onMounted, onUnmounted, ref, unref, watch, type Ref } from "vue"
import {
  applyDeadzone,
  createRepeatGate,
  diffButtonEdges,
  directionFromAxes,
  supportsDualRumble,
} from "@/lib/gamepad/gamepad-input"
import {
  DEFAULT_GAMEPAD_DEADZONE,
  DEFAULT_GAMEPAD_REPEAT_INITIAL_DELAY_MS,
  DEFAULT_GAMEPAD_REPEAT_MS,
  STANDARD_GAMEPAD_AXIS,
  STANDARD_GAMEPAD_BUTTON,
  type GamepadDirection,
  type StandardGamepadButtonName,
} from "@/lib/gamepad/standard-gamepad"

type ButtonPressHandler = (payload: { button: number; gamepad: Gamepad; timestamp: number }) => void
type DirectionPressHandler = (payload: { direction: GamepadDirection; gamepad: Gamepad; timestamp: number }) => void
type HandlerCleanup = () => void

export interface GamepadRumblePattern {
  duration: number
  startDelay?: number
  strongMagnitude?: number
  weakMagnitude?: number
}

export interface UseGamepadOptions {
  enabled?: Ref<boolean> | boolean
  deadzone?: number
  repeatInitialDelayMs?: number
  repeatMs?: number
}

export interface UseGamepadReturn {
  supported: Ref<boolean>
  connected: Ref<boolean>
  gamepadId: Ref<string>
  lastInputAt: Ref<number>
  onButtonPress(button: StandardGamepadButtonName | number, handler: ButtonPressHandler): HandlerCleanup
  onDirectionPress(direction: GamepadDirection, handler: DirectionPressHandler): HandlerCleanup
  rumble(pattern: GamepadRumblePattern): Promise<void>
  stop(): void
}

function clampMagnitude(value: number | undefined): number {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(1, Number(value)))
}

function resolveButtonIndex(button: StandardGamepadButtonName | number): number {
  if (typeof button === "number") {
    return Math.max(0, Math.floor(button))
  }
  return STANDARD_GAMEPAD_BUTTON[button]
}

function getBrowserWindow(): Window | null {
  return typeof window === "undefined" ? null : window
}

function getBrowserNavigator(): Navigator | null {
  return typeof navigator === "undefined" ? null : navigator
}

function selectStandardGamepad(nav: Navigator | null): Gamepad | null {
  const getGamepads = nav?.getGamepads
  if (typeof getGamepads !== "function") return null
  const pads = getGamepads.call(nav)
  for (const pad of pads) {
    if (pad?.connected && pad.mapping === "standard") {
      return pad
    }
  }
  return null
}

function readPressedButtons(gamepad: Gamepad): boolean[] {
  return gamepad.buttons.map((button) => button.pressed || button.value >= 0.5)
}

function readDirection(gamepad: Gamepad, deadzone: number): GamepadDirection | null {
  const pressed = readPressedButtons(gamepad)
  if (pressed[STANDARD_GAMEPAD_BUTTON.dpadUp]) return "up"
  if (pressed[STANDARD_GAMEPAD_BUTTON.dpadDown]) return "down"
  if (pressed[STANDARD_GAMEPAD_BUTTON.dpadLeft]) return "left"
  if (pressed[STANDARD_GAMEPAD_BUTTON.dpadRight]) return "right"

  return directionFromAxes(
    applyDeadzone(gamepad.axes[STANDARD_GAMEPAD_AXIS.leftX] ?? 0, deadzone),
    applyDeadzone(gamepad.axes[STANDARD_GAMEPAD_AXIS.leftY] ?? 0, deadzone),
    deadzone,
  )
}

export function useGamepad(options: UseGamepadOptions = {}): UseGamepadReturn {
  const win = getBrowserWindow()
  const nav = getBrowserNavigator()
  const enabled = computed(() => options.enabled == null || unref(options.enabled))
  const supported = ref(Boolean(nav && typeof nav.getGamepads === "function"))
  const connected = ref(false)
  const gamepadId = ref("")
  const lastInputAt = ref(0)
  const selectedGamepad = ref<Gamepad | null>(null)
  const deadzone = options.deadzone ?? DEFAULT_GAMEPAD_DEADZONE
  const buttonHandlers = new Map<number, Set<ButtonPressHandler>>()
  const directionHandlers = new Map<GamepadDirection, Set<DirectionPressHandler>>()
  const repeatGate = createRepeatGate({
    initialDelayMs: options.repeatInitialDelayMs ?? DEFAULT_GAMEPAD_REPEAT_INITIAL_DELAY_MS,
    repeatMs: options.repeatMs ?? DEFAULT_GAMEPAD_REPEAT_MS,
  })

  let frameId: number | null = null
  let previousButtons: boolean[] = []

  function clearConnectionState() {
    connected.value = false
    gamepadId.value = ""
    selectedGamepad.value = null
    previousButtons = []
    repeatGate.reset()
  }

  function queueFrame() {
    if (!win || frameId != null || !enabled.value || !supported.value) return
    frameId = win.requestAnimationFrame(poll)
  }

  function fireButtonHandlers(gamepad: Gamepad, pressed: readonly number[], timestamp: number) {
    for (const button of pressed) {
      const handlers = buttonHandlers.get(button)
      if (!handlers?.size) continue
      lastInputAt.value = timestamp
      for (const handler of handlers) {
        handler({ button, gamepad, timestamp })
      }
    }
  }

  function fireDirectionHandler(gamepad: Gamepad, direction: GamepadDirection | null, timestamp: number) {
    if (!repeatGate.shouldFire(direction, timestamp) || !direction) return
    const handlers = directionHandlers.get(direction)
    if (!handlers?.size) return
    lastInputAt.value = timestamp
    for (const handler of handlers) {
      handler({ direction, gamepad, timestamp })
    }
  }

  function poll(timestamp: number) {
    frameId = null
    if (!enabled.value) {
      clearConnectionState()
      return
    }

    supported.value = Boolean(nav && typeof nav.getGamepads === "function")
    const gamepad = selectStandardGamepad(nav)
    if (!gamepad) {
      clearConnectionState()
      queueFrame()
      return
    }

    connected.value = true
    gamepadId.value = gamepad.id
    selectedGamepad.value = gamepad

    const currentButtons = readPressedButtons(gamepad)
    const edges = diffButtonEdges(previousButtons, currentButtons)
    fireButtonHandlers(gamepad, edges.pressed, timestamp)
    fireDirectionHandler(gamepad, readDirection(gamepad, deadzone), timestamp)
    previousButtons = currentButtons

    queueFrame()
  }

  function stop() {
    if (frameId != null && win) {
      win.cancelAnimationFrame(frameId)
      frameId = null
    }
    clearConnectionState()
  }

  function onButtonPress(
    button: StandardGamepadButtonName | number,
    handler: ButtonPressHandler,
  ): HandlerCleanup {
    const index = resolveButtonIndex(button)
    const handlers = buttonHandlers.get(index) ?? new Set<ButtonPressHandler>()
    handlers.add(handler)
    buttonHandlers.set(index, handlers)
    return () => {
      handlers.delete(handler)
      if (handlers.size === 0) {
        buttonHandlers.delete(index)
      }
    }
  }

  function onDirectionPress(
    direction: GamepadDirection,
    handler: DirectionPressHandler,
  ): HandlerCleanup {
    const handlers = directionHandlers.get(direction) ?? new Set<DirectionPressHandler>()
    handlers.add(handler)
    directionHandlers.set(direction, handlers)
    return () => {
      handlers.delete(handler)
      if (handlers.size === 0) {
        directionHandlers.delete(direction)
      }
    }
  }

  async function rumble(pattern: GamepadRumblePattern): Promise<void> {
    const gamepad = selectedGamepad.value
    if (!supportsDualRumble(gamepad)) return
    try {
      await gamepad.vibrationActuator.playEffect("dual-rumble", {
        duration: Math.max(0, pattern.duration),
        startDelay: Math.max(0, pattern.startDelay ?? 0),
        strongMagnitude: clampMagnitude(pattern.strongMagnitude),
        weakMagnitude: clampMagnitude(pattern.weakMagnitude),
      })
    } catch {
      // Haptics are optional across browsers; input should never fail because vibration did.
    }
  }

  function onConnectionChange() {
    queueFrame()
  }

  onMounted(() => {
    win?.addEventListener("gamepadconnected", onConnectionChange)
    win?.addEventListener("gamepaddisconnected", onConnectionChange)
    queueFrame()
  })

  onUnmounted(() => {
    win?.removeEventListener("gamepadconnected", onConnectionChange)
    win?.removeEventListener("gamepaddisconnected", onConnectionChange)
    stop()
  })

  if (isRef(options.enabled)) {
    watch(options.enabled, (nextEnabled) => {
      if (nextEnabled) {
        queueFrame()
      } else {
        stop()
      }
    })
  }

  return {
    supported,
    connected,
    gamepadId,
    lastInputAt,
    onButtonPress,
    onDirectionPress,
    rumble,
    stop,
  }
}
