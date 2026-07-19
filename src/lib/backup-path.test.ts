import { describe, expect, it } from "vitest"
import {
  buildBackupFilename,
  ensureBackupExtension,
  joinBackupDestination,
} from "./backup-path"

describe("backup path helpers", () => {
  it("builds stable UTC filenames", () => {
    expect(buildBackupFilename(new Date("2026-07-20T02:03:04.567Z"))).toBe(
      "curated-20260720-020304Z.curated-backup",
    )
  })

  it("joins Windows and Unix directories without duplicate separators", () => {
    expect(joinBackupDestination("D:\\Backups\\", "curated.curated-backup")).toBe(
      "D:\\Backups\\curated.curated-backup",
    )
    expect(joinBackupDestination("/srv/backups/", "curated.curated-backup")).toBe(
      "/srv/backups/curated.curated-backup",
    )
    expect(joinBackupDestination("/", "curated.curated-backup")).toBe(
      "/curated.curated-backup",
    )
    expect(joinBackupDestination("D:\\", "curated.curated-backup")).toBe(
      "D:\\curated.curated-backup",
    )
  })

  it("adds the package extension exactly once", () => {
    expect(ensureBackupExtension("D:\\Backups\\curated")).toBe(
      "D:\\Backups\\curated.curated-backup",
    )
    expect(ensureBackupExtension("D:\\Backups\\CURATED.CURATED-BACKUP")).toBe(
      "D:\\Backups\\CURATED.CURATED-BACKUP",
    )
  })
})
