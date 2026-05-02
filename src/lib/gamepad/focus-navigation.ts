import type { GamepadDirection } from "@/lib/gamepad/standard-gamepad"

export interface FocusPoint {
  x: number
  y: number
}

export interface FocusRect {
  left: number
  top: number
  right: number
  bottom: number
  width: number
  height: number
}

export interface FocusNavigationItem<T = HTMLElement> {
  id?: string
  target: T
  rect: FocusRect
  disabled?: boolean
  hidden?: boolean
  ariaHidden?: boolean
}

export const DEFAULT_GAMEPAD_FOCUSABLE_SELECTOR = [
  "button:not(:disabled)",
  "a[href]",
  "input:not(:disabled)",
  "select:not(:disabled)",
  "textarea:not(:disabled)",
  "[role='button']",
  "[role='tab']",
  "[data-gamepad-focusable]",
].join(",")

function centerOf(rect: FocusRect): FocusPoint {
  return {
    x: rect.left + rect.width / 2,
    y: rect.top + rect.height / 2,
  }
}

function distanceBetween(a: FocusPoint, b: FocusPoint): number {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

export function isUsableFocusItem<T>(item: FocusNavigationItem<T>): boolean {
  return !item.disabled && !item.hidden && !item.ariaHidden && item.rect.width > 0 && item.rect.height > 0
}

export function findInitialFocusable<T>(
  items: readonly FocusNavigationItem<T>[],
  point: FocusPoint,
): FocusNavigationItem<T> | null {
  const candidates = items.filter(isUsableFocusItem)
  if (candidates.length === 0) return null

  return candidates
    .slice()
    .sort((left, right) => {
      const leftDistance = distanceBetween(centerOf(left.rect), point)
      const rightDistance = distanceBetween(centerOf(right.rect), point)
      return leftDistance - rightDistance
    })[0] ?? null
}

function directionScore(
  origin: FocusPoint,
  target: FocusPoint,
  direction: GamepadDirection,
): number | null {
  const dx = target.x - origin.x
  const dy = target.y - origin.y
  const primary =
    direction === "right" ? dx :
    direction === "left" ? -dx :
    direction === "down" ? dy :
    -dy

  if (primary <= 0) return null

  const cross = direction === "left" || direction === "right" ? Math.abs(dy) : Math.abs(dx)
  const angle = Math.atan2(cross, primary)
  const distance = Math.hypot(dx, dy)
  return angle * 100_000 + distance
}

export function findNextFocusable<T>(
  items: readonly FocusNavigationItem<T>[],
  current: FocusNavigationItem<T> | null | undefined,
  direction: GamepadDirection,
  fallbackPoint: FocusPoint = {
    x: typeof window === "undefined" ? 0 : window.innerWidth / 2,
    y: typeof window === "undefined" ? 0 : window.innerHeight / 2,
  },
): FocusNavigationItem<T> | null {
  if (!current || !isUsableFocusItem(current)) {
    return findInitialFocusable(items, fallbackPoint)
  }

  const origin = centerOf(current.rect)
  let best: { item: FocusNavigationItem<T>; score: number } | null = null

  for (const candidate of items) {
    if (candidate === current || candidate.target === current.target) continue
    if (!isUsableFocusItem(candidate)) continue
    const score = directionScore(origin, centerOf(candidate.rect), direction)
    if (score == null) continue
    if (!best || score < best.score) {
      best = { item: candidate, score }
    }
  }

  return best?.item ?? current
}

function isElementDisabled(el: HTMLElement): boolean {
  if (el.hasAttribute("disabled") || el.getAttribute("aria-disabled") === "true") return true
  if ("disabled" in el && typeof el.disabled === "boolean") {
    return el.disabled
  }
  return false
}

function isElementAriaHidden(el: HTMLElement): boolean {
  return Boolean(el.closest("[aria-hidden='true']"))
}

function isElementHidden(el: HTMLElement, rect: FocusRect): boolean {
  return el.hidden || rect.width <= 0 || rect.height <= 0
}

export function collectGamepadFocusableItems(
  root: ParentNode = document,
): FocusNavigationItem<HTMLElement>[] {
  return Array.from(root.querySelectorAll<HTMLElement>(DEFAULT_GAMEPAD_FOCUSABLE_SELECTOR)).map((el) => {
    const rect = el.getBoundingClientRect()
    return {
      id: el.id || el.dataset.gamepadFocusId,
      target: el,
      rect,
      disabled: isElementDisabled(el),
      hidden: isElementHidden(el, rect),
      ariaHidden: isElementAriaHidden(el),
    }
  }).filter(isUsableFocusItem)
}
