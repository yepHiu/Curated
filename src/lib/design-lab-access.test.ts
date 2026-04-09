import { describe, expect, it } from "vitest"
import { resolveDesignLabAccess } from "@/lib/design-lab-access"

describe("resolveDesignLabAccess", () => {
  it("allows access in development mode", () => {
    expect(resolveDesignLabAccess(true)).toEqual({
      enabled: true,
      fallbackTarget: null,
    })
  })

  it("redirects to settings about outside development mode", () => {
    expect(resolveDesignLabAccess(false)).toEqual({
      enabled: false,
      fallbackTarget: {
        name: "settings",
        query: {
          section: "about",
        },
      },
    })
  })
})
