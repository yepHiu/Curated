import { computed, onMounted, onUnmounted, ref, unref, type Ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useGamepad, type UseGamepadReturn } from "@/composables/use-gamepad"
import { useGamepadControlsPreference } from "@/lib/gamepad/gamepad-settings"
import {
  collectGamepadFocusableItems,
  findInitialFocusable,
  findNextFocusable,
  type FocusNavigationItem,
  type FocusPoint,
} from "@/lib/gamepad/focus-navigation"
import type { GamepadDirection } from "@/lib/gamepad/standard-gamepad"

type MaybeRef<T> = Ref<T> | T

export interface UseGamepadFocusNavigationOptions {
  enabled?: MaybeRef<boolean>
  root?: MaybeRef<ParentNode | null | undefined>
}

export interface UseGamepadFocusNavigationReturn {
  gamepad: UseGamepadReturn
  focusedElement: Ref<HTMLElement | null>
  clearFocus: () => void
}

const PRIMARY_ROUTE_NAMES = [
  "home",
  "library",
  "favorites",
  "actors",
  "tags",
  "curated-frames",
  "history",
  "settings",
] as const

function getBrowserWindow(): Window | null {
  return typeof window === "undefined" ? null : window
}

function getBrowserDocument(): Document | null {
  return typeof document === "undefined" ? null : document
}

function viewportCenter(): FocusPoint {
  const win = getBrowserWindow()
  return {
    x: win ? win.innerWidth / 2 : 0,
    y: win ? win.innerHeight / 2 : 0,
  }
}

function resolveRoot(root: MaybeRef<ParentNode | null | undefined> | undefined): ParentNode | null {
  return unref(root) ?? getBrowserDocument()
}

function findItemForElement(
  items: readonly FocusNavigationItem<HTMLElement>[],
  el: HTMLElement | null,
): FocusNavigationItem<HTMLElement> | null {
  if (!el) return null
  return items.find((item) => item.target === el) ?? null
}

function focusElement(el: HTMLElement) {
  try {
    el.focus({ preventScroll: true })
  } catch {
    try {
      el.focus()
    } catch {
      // Some custom focusable elements may not be focusable despite matching selectors.
    }
  }
  try {
    el.scrollIntoView({ block: "nearest", inline: "nearest" })
  } catch {
    // Non-visual test environments can omit scrollIntoView.
  }
}

function clickElement(el: HTMLElement) {
  el.click()
}

export function useGamepadFocusNavigation(
  options: UseGamepadFocusNavigationOptions = {},
): UseGamepadFocusNavigationReturn {
  const route = useRoute()
  const router = useRouter()
  const { gamepadControlsEnabled } = useGamepadControlsPreference()
  const focusedElement = ref<HTMLElement | null>(null)
  const enabled = computed(
    () =>
      gamepadControlsEnabled.value &&
      String(route.name ?? "") !== "player" &&
      (options.enabled == null || unref(options.enabled)),
  )

  function clearFocus() {
    focusedElement.value?.removeAttribute("data-controller-focused")
    focusedElement.value = null
  }

  function setFocus(el: HTMLElement) {
    if (focusedElement.value !== el) {
      clearFocus()
    }
    focusedElement.value = el
    el.setAttribute("data-controller-focused", "true")
    focusElement(el)
  }

  function currentItem(items: readonly FocusNavigationItem<HTMLElement>[]) {
    const explicit = findItemForElement(items, focusedElement.value)
    if (explicit) return explicit

    const active = getBrowserDocument()?.activeElement
    if (active instanceof HTMLElement) {
      return findItemForElement(items, active)
    }
    return null
  }

  function collectItems() {
    const root = resolveRoot(options.root)
    return root ? collectGamepadFocusableItems(root) : []
  }

  function moveFocus(direction: GamepadDirection) {
    if (!enabled.value) return
    const items = collectItems()
    const next = findNextFocusable(items, currentItem(items), direction, viewportCenter())
    if (!next) return
    setFocus(next.target)
    void gamepad.rumble({ duration: 18, weakMagnitude: 0.18, strongMagnitude: 0.05 })
  }

  function clickFocusedElement() {
    if (!enabled.value) return
    const items = collectItems()
    const current = currentItem(items)
    if (current) {
      clickElement(current.target)
      void gamepad.rumble({ duration: 35, weakMagnitude: 0.32, strongMagnitude: 0.12 })
      return
    }

    const initial = findInitialFocusable(items, viewportCenter())
    if (initial) {
      setFocus(initial.target)
    }
  }

  function navigatePrimaryRoute(delta: -1 | 1) {
    if (!enabled.value) return
    const currentName = String(route.name ?? "")
    const currentIndex = PRIMARY_ROUTE_NAMES.findIndex((name) => name === currentName)
    const baseIndex = currentIndex >= 0 ? currentIndex : 0
    const nextIndex =
      (baseIndex + delta + PRIMARY_ROUTE_NAMES.length) % PRIMARY_ROUTE_NAMES.length
    void router.push({ name: PRIMARY_ROUTE_NAMES[nextIndex] })
  }

  function goBack() {
    if (!enabled.value) return
    router.back()
  }

  const gamepad = useGamepad({ enabled })
  const cleanups = [
    gamepad.onDirectionPress("up", () => moveFocus("up")),
    gamepad.onDirectionPress("down", () => moveFocus("down")),
    gamepad.onDirectionPress("left", () => moveFocus("left")),
    gamepad.onDirectionPress("right", () => moveFocus("right")),
    gamepad.onButtonPress("cross", clickFocusedElement),
    gamepad.onButtonPress("circle", goBack),
    gamepad.onButtonPress("l1", () => navigatePrimaryRoute(-1)),
    gamepad.onButtonPress("r1", () => navigatePrimaryRoute(1)),
    gamepad.onButtonPress("options", () => {
      if (enabled.value) void router.push({ name: "settings" })
    }),
  ]

  function onPointerOrKeyboardInput() {
    clearFocus()
  }

  onMounted(() => {
    const win = getBrowserWindow()
    win?.addEventListener("pointerdown", onPointerOrKeyboardInput)
    win?.addEventListener("keydown", onPointerOrKeyboardInput)
  })

  onUnmounted(() => {
    const win = getBrowserWindow()
    win?.removeEventListener("pointerdown", onPointerOrKeyboardInput)
    win?.removeEventListener("keydown", onPointerOrKeyboardInput)
    for (const cleanup of cleanups) {
      cleanup()
    }
    clearFocus()
  })

  return {
    gamepad,
    focusedElement,
    clearFocus,
  }
}
