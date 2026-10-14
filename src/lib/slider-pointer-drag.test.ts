import { describe, expect, it } from "vitest"
import { createSliderPointerDrag, scaleSliderPointer } from "./slider-pointer-drag"

const track = { left: 0, width: 200, thumbWidth: 16, min: 0, max: 100, step: 1 }

describe("scaleSliderPointer", () => {
  it("maps a track click through the thumb center to the full range", () => {
    expect(scaleSliderPointer(8, 8, track)).toBe(0)
    expect(scaleSliderPointer(100, 8, track)).toBe(50)
    expect(scaleSliderPointer(192, 8, track)).toBe(100)
  })

  it("keeps a thumb grab offset so the value does not jump under the pointer", () => {
    expect(scaleSliderPointer(48, 12, track)).toBe(scaleSliderPointer(44, 8, track))
  })

  it("clamps outside the track and snaps to the step", () => {
    expect(scaleSliderPointer(-40, 8, track)).toBe(0)
    expect(scaleSliderPointer(400, 8, track)).toBe(100)
    expect(scaleSliderPointer(20, 8, { ...track, min: 0, max: 1, step: 0.1 })).toBe(0.1)
  })
})

describe("createSliderPointerDrag", () => {
  it("follows later moves for the same pointer and ignores a second pointer", () => {
    const drag = createSliderPointerDrag()
    expect(drag.begin(4, 100, 8, track)).toBe(50)
    expect(drag.update(9, 192, 0, 200)).toBeUndefined()
    expect(drag.update(4, 192, 0, 200)).toBe(100)
    expect(drag.finish(9, 0, 0, 200)).toBeUndefined()
    expect(drag.finish(4, 8, 0, 200)).toBe(0)
    expect(drag.active()).toBe(false)
  })

  it("cancels without a value when the pointer position is unknown", () => {
    const drag = createSliderPointerDrag()
    drag.begin(1, 100, 8, track)
    expect(drag.finish(1)).toBeUndefined()
    expect(drag.currentPointerId()).toBeUndefined()
  })
})
