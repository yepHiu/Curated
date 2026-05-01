import { describe, expect, it } from "vitest"
import detailPageSource from "./DetailPage.vue?raw"
import libraryBatchActionBarSource from "./LibraryBatchActionBar.vue?raw"
import scanProgressDockSource from "./ScanProgressDock.vue?raw"

describe("jav-library render hints", () => {
  it("keeps scan progress stat labels, detail placeholders, and batch confirm titles static", () => {
    for (const key of ["scan.processed", "scan.newItems", "scan.updated", "scan.skipped"]) {
      expect(scanProgressDockSource).toContain(`<span v-once>{{ t("${key}") }}</span>`)
    }

    expect(detailPageSource).toContain(`v-once
          v-for="index in 3"`)
    expect(libraryBatchActionBarSource.match(/<DialogTitle v-once>/g)).toHaveLength(4)
  })
})
