import { ref, type Ref } from "vue"

export const GAMEPAD_CONTROLS_STORAGE_KEY = "curated-gamepad-controls-v1"

export type GamepadPreferenceStorage =
  | Pick<Storage, "getItem" | "setItem">
  | Map<string, string>

function getBrowserStorage(): GamepadPreferenceStorage | null {
  if (typeof window === "undefined") return null
  try {
    return window.localStorage
  } catch {
    return null
  }
}

function readPreference(storage: GamepadPreferenceStorage, key: string): string | null {
  if ("getItem" in storage) {
    return storage.getItem(key)
  }
  return storage.get(key) ?? null
}

function writePreference(storage: GamepadPreferenceStorage, key: string, value: string) {
  if ("setItem" in storage) {
    storage.setItem(key, value)
    return
  }
  storage.set(key, value)
}

export function getStoredGamepadControlsEnabled(
  storage: GamepadPreferenceStorage | null = getBrowserStorage(),
): boolean {
  if (!storage) return true
  const raw = readPreference(storage, GAMEPAD_CONTROLS_STORAGE_KEY)
  if (raw === "false") return false
  if (raw === "true") return true
  return true
}

export function setStoredGamepadControlsEnabled(
  enabled: boolean,
  storage: GamepadPreferenceStorage | null = getBrowserStorage(),
) {
  if (!storage) return
  writePreference(storage, GAMEPAD_CONTROLS_STORAGE_KEY, enabled ? "true" : "false")
}

const gamepadControlsEnabled = ref(getStoredGamepadControlsEnabled())

export function useGamepadControlsPreference(): {
  gamepadControlsEnabled: Ref<boolean>
  setGamepadControlsEnabled: (enabled: boolean) => void
} {
  function setGamepadControlsEnabled(enabled: boolean) {
    gamepadControlsEnabled.value = enabled
    setStoredGamepadControlsEnabled(enabled)
  }

  return {
    gamepadControlsEnabled,
    setGamepadControlsEnabled,
  }
}
