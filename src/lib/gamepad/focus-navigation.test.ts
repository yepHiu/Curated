import { describe, expect, it } from "vitest"
import {
  findInitialFocusable,
  findNextFocusable,
  type FocusNavigationItem,
} from "@/lib/gamepad/focus-navigation"

function item(
  id: string,
  rect: { left: number; top: number; width: number; height: number },
  options: Partial<FocusNavigationItem<string>> = {},
): FocusNavigationItem<string> {
  return {
    id,
    target: id,
    rect: {
      ...rect,
      right: rect.left + rect.width,
      bottom: rect.top + rect.height,
    },
    ...options,
  }
}

describe("focus navigation", () => {
  it("chooses the closest aligned target in the requested direction", () => {
    const current = item("current", { left: 100, top: 100, width: 50, height: 50 })
    const rightAligned = item("right", { left: 220, top: 105, width: 50, height: 50 })
    const rightFarDiagonal = item("diagonal", { left: 190, top: 260, width: 50, height: 50 })
    const left = item("left", { left: 20, top: 105, width: 50, height: 50 })

    const next = findNextFocusable([current, rightFarDiagonal, rightAligned, left], current, "right")

    expect(next?.target).toBe("right")
  })

  it("skips hidden disabled and zero-size candidates", () => {
    const current = item("current", { left: 100, top: 100, width: 50, height: 50 })
    const hidden = item("hidden", { left: 210, top: 100, width: 50, height: 50 }, { hidden: true })
    const disabled = item("disabled", { left: 230, top: 100, width: 50, height: 50 }, { disabled: true })
    const zeroSize = item("zero", { left: 250, top: 100, width: 0, height: 50 })
    const enabled = item("enabled", { left: 270, top: 100, width: 50, height: 50 })

    const next = findNextFocusable([current, hidden, disabled, zeroSize, enabled], current, "right")

    expect(next?.target).toBe("enabled")
  })

  it("selects the candidate nearest to the viewport center when current focus is missing", () => {
    const nearCenter = item("center", { left: 470, top: 360, width: 80, height: 40 })
    const farTopLeft = item("top-left", { left: 0, top: 0, width: 80, height: 40 })
    const farBottomRight = item("bottom-right", { left: 900, top: 700, width: 80, height: 40 })

    const next = findInitialFocusable([farTopLeft, nearCenter, farBottomRight], {
      x: 500,
      y: 380,
    })

    expect(next?.target).toBe("center")
  })
})
