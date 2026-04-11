import { describe, expect, it } from "vitest"
import { statusBadgeClass } from "@/lib/ui/status-tone"
import { badgeVariants } from "./index"

describe("badge variants", () => {
  it("uses semantic status tone classes for status variants", () => {
    expect(badgeVariants({ variant: "success" })).toContain(statusBadgeClass("success"))
    expect(badgeVariants({ variant: "warning" })).toContain(statusBadgeClass("warning"))
    expect(badgeVariants({ variant: "danger" })).toContain(statusBadgeClass("danger"))
    expect(badgeVariants({ variant: "info" })).toContain(statusBadgeClass("info"))
  })
})
