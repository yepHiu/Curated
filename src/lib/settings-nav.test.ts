import { describe, expect, it } from "vitest"
import { SETTINGS_NAV_ITEMS, isSettingsSectionSlug } from "./settings-nav"

describe("settings navigation", () => {
  it("includes the security section", () => {
    expect(SETTINGS_NAV_ITEMS).toContainEqual({
      slug: "security",
      labelKey: "settings.navSecurity",
    })
    expect(isSettingsSectionSlug("security")).toBe(true)
  })
})
