import type { HealthDTO } from "@/api/types"

/** Product version and build identity are displayed separately. */
export function formatAboutBackendVersion(h: HealthDTO): string {
  return h.version
}

/** Health now reports a stable installer version string in both release and dev runtimes. */
export function formatAboutInstallerVersion(h: HealthDTO): string {
  return typeof h.installerVersion === "string" ? h.installerVersion.trim() : ""
}
