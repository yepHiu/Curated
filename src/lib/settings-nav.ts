/** 设置页内导航 slug；与路由 query `section` 一致。 */
export type SettingsSectionSlug =
  | "overview"
  | "general"
  | "library"
  | "metadata"
  | "network"
  | "curated"
  | "playback"
  | "maintenance"
  | "about"

export function settingsSectionDomId(slug: SettingsSectionSlug): string {
  return `settings-section-${slug}`
}

export const SETTINGS_NAV_ITEMS: { slug: SettingsSectionSlug; labelKey: string }[] = [
  { slug: "overview", labelKey: "settings.navOverview" },
  { slug: "general", labelKey: "settings.navGeneral" },
  { slug: "library", labelKey: "settings.navLibrary" },
  { slug: "metadata", labelKey: "settings.navMetadata" },
  { slug: "network", labelKey: "settings.navNetwork" },
  { slug: "curated", labelKey: "settings.navCurated" },
  { slug: "playback", labelKey: "settings.navPlayback" },
  { slug: "maintenance", labelKey: "settings.navMaintenance" },
  { slug: "about", labelKey: "settings.navAbout" },
]

export function isSettingsSectionSlug(s: string): s is SettingsSectionSlug {
  return SETTINGS_NAV_ITEMS.some((item) => item.slug === s)
}
