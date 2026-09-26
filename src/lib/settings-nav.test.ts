import { describe, expect, it } from "vitest"
import {
  SETTINGS_NAV_GROUPS,
  isSettingsSectionAvailable,
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

it("keeps Desktop server switching available on remote connections", () => {
  expect(isSettingsSectionAvailable("network", false, true)).toBe(true)
  expect(isSettingsSectionAvailable("network", false, false)).toBe(false)
  for (const slug of ["metadata", "maintenance", "ai"] as const) {
    expect(isSettingsSectionAvailable(slug, true, false)).toBe(true)
    expect(isSettingsSectionAvailable(slug, false, true)).toBe(false)
  }
})
