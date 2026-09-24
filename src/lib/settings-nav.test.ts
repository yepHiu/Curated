import { describe, expect, it } from "vitest"
import {
  SETTINGS_NAV_GROUPS,
  SETTINGS_NAV_ITEMS,
  isSettingsSectionSlug,
  resolveSettingsSectionSlug,
} from "./settings-nav"

describe("settings navigation", () => {
  it("includes the security section", () => {
    expect(SETTINGS_NAV_ITEMS).toContainEqual({
      slug: "security",
      labelKey: "settings.navSecurity",
    })
    expect(isSettingsSectionSlug("security")).toBe(true)
  })

  it("groups content libraries and preserves legacy deep links", () => {
    expect(SETTINGS_NAV_GROUPS.find((group) => group.labelKey === "settings.navGroupLibrary")?.items)
      .toEqual(expect.arrayContaining([
        { slug: "comics", labelKey: "settings.navComics", beta: true },
        { slug: "photos", labelKey: "settings.navPhotos", beta: true },
      ]))
    expect(SETTINGS_NAV_ITEMS.map((item) => item.slug)).not.toContain("experimental")
    expect(resolveSettingsSectionSlug("experimental")).toBe("comics")
    expect(resolveSettingsSectionSlug("logging")).toBe("maintenance")
    expect(resolveSettingsSectionSlug("libraryBehavior")).toBe("library")
  })
})
