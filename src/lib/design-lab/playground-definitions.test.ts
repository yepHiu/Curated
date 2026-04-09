import { describe, expect, it } from "vitest"
import {
  createDefaultDesignLabPlaygroundState,
  renderPlaygroundOutput,
} from "@/lib/design-lab/playground-definitions"

describe("renderPlaygroundOutput", () => {
  it("renders a button snippet with width and loading state", () => {
    const state = createDefaultDesignLabPlaygroundState()
    state.componentId = "button"
    state.button.loading = true
    state.button.fullWidth = true

    const output = renderPlaygroundOutput(state)

    expect(output.vueSnippet).toContain("<Button")
    expect(output.vueSnippet).toContain("disabled")
    expect(output.vueSnippet).toContain("w-full")
    expect(output.tokenNotes.length).toBeGreaterThan(0)
  })

  it("renders a card snippet with semantic utility classes", () => {
    const state = createDefaultDesignLabPlaygroundState()
    state.componentId = "card"
    state.card.shadow = "lg"

    const output = renderPlaygroundOutput(state)

    expect(output.vueSnippet).toContain("<Card")
    expect(output.vueSnippet).toContain("shadow-[var(--shadow-lg)]")
    expect(output.tokenNotes).toContain("Use surface and border tokens to keep cards aligned with app chrome.")
  })
})
