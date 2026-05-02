import { onUnmounted, unref, type Ref } from "vue"
import { useGamepad } from "@/composables/use-gamepad"
import type { UseGamepadReturn } from "@/composables/use-gamepad"

type MaybeRef<T> = Ref<T> | T
type MaybeAsyncAction = () => void | Promise<void>

export interface PlayerGamepadActions {
  togglePlayPause: MaybeAsyncAction
  seekDelta?: (deltaSec: number) => void
  adjustVolume?: (deltaPercent: number) => void
  toggleMute?: MaybeAsyncAction
  toggleChrome?: MaybeAsyncAction
  toggleDetailedStats?: MaybeAsyncAction
  runCuratedCapture?: MaybeAsyncAction
  exitFullscreen?: MaybeAsyncAction
  exitPlayer?: MaybeAsyncAction
}

export interface UsePlayerGamepadControlsOptions {
  enabled?: MaybeRef<boolean>
  isFullscreen?: MaybeRef<boolean> | (() => boolean)
  seekBackwardStepSec: MaybeRef<number>
  seekForwardStepSec: MaybeRef<number>
  actions: PlayerGamepadActions
}

export interface UsePlayerGamepadControlsReturn {
  gamepad: UseGamepadReturn
}

function readMaybeRefNumber(value: MaybeRef<number>): number {
  const next = Number(unref(value))
  return Number.isFinite(next) ? next : 0
}

function readFullscreen(value: UsePlayerGamepadControlsOptions["isFullscreen"]): boolean {
  if (typeof value === "function") {
    return value()
  }
  return Boolean(value == null ? false : unref(value))
}

function runAction(action: MaybeAsyncAction | undefined) {
  if (!action) return
  void action()
}

export function usePlayerGamepadControls(
  options: UsePlayerGamepadControlsOptions,
): UsePlayerGamepadControlsReturn {
  const gamepad = useGamepad({ enabled: options.enabled })
  const cleanups = [
    gamepad.onButtonPress("cross", () => {
      runAction(options.actions.togglePlayPause)
      void gamepad.rumble({ duration: 35, weakMagnitude: 0.35, strongMagnitude: 0.15 })
    }),
    gamepad.onButtonPress("circle", () => {
      if (readFullscreen(options.isFullscreen)) {
        runAction(options.actions.exitFullscreen)
      } else {
        runAction(options.actions.exitPlayer)
      }
      void gamepad.rumble({ duration: 25, weakMagnitude: 0.25, strongMagnitude: 0.1 })
    }),
    gamepad.onButtonPress("square", () => {
      runAction(options.actions.runCuratedCapture)
      void gamepad.rumble({ duration: 35, weakMagnitude: 0.35, strongMagnitude: 0.15 })
    }),
    gamepad.onButtonPress("triangle", () => {
      runAction(options.actions.toggleDetailedStats)
    }),
    gamepad.onButtonPress("options", () => {
      runAction(options.actions.toggleChrome)
    }),
    gamepad.onButtonPress("psOrHome", () => {
      runAction(options.actions.toggleMute)
    }),
    gamepad.onDirectionPress("left", () => {
      options.actions.seekDelta?.(-readMaybeRefNumber(options.seekBackwardStepSec))
    }),
    gamepad.onDirectionPress("right", () => {
      options.actions.seekDelta?.(readMaybeRefNumber(options.seekForwardStepSec))
    }),
    gamepad.onDirectionPress("up", () => {
      options.actions.adjustVolume?.(5)
    }),
    gamepad.onDirectionPress("down", () => {
      options.actions.adjustVolume?.(-5)
    }),
  ]

  onUnmounted(() => {
    for (const cleanup of cleanups) {
      cleanup()
    }
  })

  return { gamepad }
}
