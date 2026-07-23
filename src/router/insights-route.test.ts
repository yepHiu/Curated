import { describe, expect, it } from "vitest"
import router from "@/router"

describe("personal insights route", () => {
  it("registers the lazy-loaded insights page inside the application shell", () => {
    const route = router.getRoutes().find((candidate) => candidate.name === "insights")
    expect(route?.path).toBe("/insights")
    expect(typeof route?.components?.default).toBe("function")
  })
})
