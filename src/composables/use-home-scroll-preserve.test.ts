import { beforeEach, describe, expect, it } from "vitest"

import {
  armHomeDetailReturnRestore,
  consumeHomeDetailReturnRestore,
  readHomeScrollSnapshot,
  resetHomeScrollRestoreState,
  saveHomeScrollSnapshot,
} from "./use-home-scroll-preserve"

describe("use-home-scroll-preserve", () => {
  beforeEach(() => {
    resetHomeScrollRestoreState()
  })

  it("stores a snapshot and returns it only once after the detail-return restore is armed", () => {
    saveHomeScrollSnapshot(428)
    armHomeDetailReturnRestore()

    expect(readHomeScrollSnapshot()).toBe(428)
    expect(consumeHomeDetailReturnRestore()).toBe(428)
    expect(consumeHomeDetailReturnRestore()).toBeNull()
  })

  it("does not restore when the home detail-return flag was never armed", () => {
    saveHomeScrollSnapshot(512)

    expect(consumeHomeDetailReturnRestore()).toBeNull()
    expect(readHomeScrollSnapshot()).toBe(512)
  })
})
