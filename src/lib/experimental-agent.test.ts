import { beforeEach, describe, expect, it } from "vitest"
import { useExperimentalAgent } from "./experimental-agent"

const STORAGE_KEY = "curated-agent-experimental-v1"

describe("useExperimentalAgent", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("defaults to disabled when storage is empty", () => {
    const { enabled } = useExperimentalAgent()
    expect(enabled.value).toBe(false)
  })

  it("persists the toggle to localStorage", () => {
    const { enabled, setEnabled } = useExperimentalAgent()
    setEnabled(true)
    expect(enabled.value).toBe(true)
    expect(localStorage.getItem(STORAGE_KEY)).toBe("true")

    setEnabled(false)
    expect(enabled.value).toBe(false)
    expect(localStorage.getItem(STORAGE_KEY)).toBe("false")
  })

  it("shares one state across composable calls", () => {
    const first = useExperimentalAgent()
    const second = useExperimentalAgent()
    first.setEnabled(true)
    expect(second.enabled.value).toBe(true)
  })
})
