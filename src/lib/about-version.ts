import type { HealthDTO } from "@/api/types"

/** Keep About app-version display aligned with the backend's version.Display() contract. */
export function formatAboutBackendVersion(h: HealthDTO): string {
  const ch = typeof h.channel === "string" ? h.channel.trim() : ""
  if (ch) return `${h.version}-${ch}`
  return h.version
}

/** Packaged installer/release version is optional and only exists for official package builds. */
export function formatAboutInstallerVersion(h: HealthDTO): string {
  return typeof h.installerVersion === "string" ? h.installerVersion.trim() : ""
}
