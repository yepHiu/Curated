import { describe, expect, it } from "vitest"
import router from "@/router"

describe("design lab removal", () => {
  it("does not register a design lab route", () => {
    expect(router.getRoutes().some((route) => route.name === "design-lab")).toBe(false)
  })
})
