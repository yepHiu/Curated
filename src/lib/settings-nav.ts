/** 设置页内导航 slug；与路由 query `section` 一致。 */
export type SettingsSectionSlug =
  | "overview"
  | "general"
  | "security"
  | "library"
  | "metadata"
  | "network"
  | "curated"
  | "playback"
  | "maintenance"
  | "comics"
  | "photos"
  | "ai"
  | "about"

export function settingsSectionDomId(slug: SettingsSectionSlug): string {
  return `settings-section-${slug}`
}

type SettingsNavItem = { slug: SettingsSectionSlug; labelKey: string; beta?: boolean }

export const SETTINGS_OVERVIEW_NAV_ITEM: SettingsNavItem = {
  slug: "overview",
  labelKey: "settings.navOverview",
}

export const SETTINGS_NAV_GROUPS: { labelKey: string; items: SettingsNavItem[] }[] = [
  {
    labelKey: "settings.navGroupExperience",
    items: [
      { slug: "general", labelKey: "settings.navGeneral" },
      { slug: "playback", labelKey: "settings.navPlayback" },
      { slug: "curated", labelKey: "settings.navCurated" },
      { slug: "ai", labelKey: "settings.navAI" },
    ],
  },
  {
    labelKey: "settings.navGroupLibrary",
    items: [
      { slug: "library", labelKey: "settings.navLibrary" },
      { slug: "metadata", labelKey: "settings.navMetadata" },
      { slug: "comics", labelKey: "settings.navComics", beta: true },
      { slug: "photos", labelKey: "settings.navPhotos", beta: true },
    ],
  },
  {
    labelKey: "settings.navGroupAccess",
    items: [
      { slug: "network", labelKey: "settings.navNetwork" },
      { slug: "security", labelKey: "settings.navSecurity" },
    ],
  },
  {
    labelKey: "settings.navGroupSystem",
    items: [
      { slug: "maintenance", labelKey: "settings.navMaintenance" },
      { slug: "about", labelKey: "settings.navAbout" },
    ],
  },
]

export const SETTINGS_NAV_ITEMS: SettingsNavItem[] = [
  SETTINGS_OVERVIEW_NAV_ITEM,
  ...SETTINGS_NAV_GROUPS.flatMap((group) => group.items),
]

export function resolveSettingsSectionSlug(value: string): SettingsSectionSlug | null {
  if (value === "experimental") return "comics"
  if (value === "libraryBehavior") return "library"
  if (value === "logging") return "maintenance"
  return isSettingsSectionSlug(value) ? value : null
}

export function isSettingsSectionSlug(s: string): s is SettingsSectionSlug {
  return SETTINGS_NAV_ITEMS.some((item) => item.slug === s)
}

/** Remote clients keep personal preferences and read-only library information. */
export function isSettingsSectionAvailable(slug: SettingsSectionSlug, serverLocal: boolean, desktop: boolean): boolean {
  if (serverLocal) return true
  if (slug === "network") return desktop
  return !(["metadata", "maintenance", "ai"] as SettingsSectionSlug[]).includes(slug)
}
