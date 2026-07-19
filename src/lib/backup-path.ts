const BACKUP_EXTENSION = ".curated-backup"

export function buildBackupFilename(now = new Date()): string {
  const timestamp = now
    .toISOString()
    .replace(/[-:]/g, "")
    .replace(/\.\d{3}Z$/, "Z")
    .replace("T", "-")
  return `curated-${timestamp}${BACKUP_EXTENSION}`
}

export function ensureBackupExtension(path: string): string {
  const trimmed = path.trim()
  if (!trimmed || trimmed.toLowerCase().endsWith(BACKUP_EXTENSION)) return trimmed
  return `${trimmed}${BACKUP_EXTENSION}`
}

export function joinBackupDestination(directory: string, filename: string): string {
  const trimmedDirectory = directory.trim()
  if (!trimmedDirectory) return filename
  if (trimmedDirectory === "/") return `/${filename}`
  if (/^[A-Za-z]:[\\/]$/.test(trimmedDirectory)) return `${trimmedDirectory}${filename}`

  const directoryWithoutTrailingSeparator = trimmedDirectory.replace(/[\\/]+$/, "")
  const separator = directoryWithoutTrailingSeparator.includes("\\") && !directoryWithoutTrailingSeparator.includes("/") ? "\\" : "/"
  return `${directoryWithoutTrailingSeparator}${separator}${filename}`
}
