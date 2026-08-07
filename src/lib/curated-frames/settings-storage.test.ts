import { beforeEach, describe, expect, it } from "vitest"
import {
  getCuratedCaptureKeyCode,
  getCuratedFrameSaveMode,
  getCuratedFrameExportMode,
  resetCuratedCaptureKeyCode,
  setCuratedCaptureKeyCode,
  setCuratedFrameSaveMode,
  setCuratedFrameExportMode,
} from "@/lib/curated-frames/settings-storage"

describe("curated frame settings storage", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("keeps the existing save-mode preference behavior", () => {
    expect(getCuratedFrameSaveMode()).toBe("app")
    setCuratedFrameSaveMode("directory")
    expect(getCuratedFrameSaveMode()).toBe("directory")
  })

  it("defaults and persists the curated frame export mode", () => {
    expect(getCuratedFrameExportMode()).toBe("raw")
    setCuratedFrameExportMode("watermarked")
    expect(getCuratedFrameExportMode()).toBe("watermarked")
    setCuratedFrameExportMode("raw")
    expect(getCuratedFrameExportMode()).toBe("raw")
  })

  it("uses C as the default curated capture shortcut", () => {
    expect(getCuratedCaptureKeyCode()).toBe("KeyC")
  })

  it("persists a supported curated capture shortcut", () => {
    setCuratedCaptureKeyCode("F8")
    expect(getCuratedCaptureKeyCode()).toBe("F8")
  })

  it("persists Page Up and Page Down curated capture shortcuts", () => {
    setCuratedCaptureKeyCode("PageUp")
    expect(getCuratedCaptureKeyCode()).toBe("PageUp")

    setCuratedCaptureKeyCode("PageDown")
    expect(getCuratedCaptureKeyCode()).toBe("PageDown")
  })

  it("falls back to the default when storage contains an invalid or reserved value", () => {
    localStorage.setItem("jav-curated-capture-key-code", "ArrowUp")
    expect(getCuratedCaptureKeyCode()).toBe("KeyC")

    localStorage.setItem("jav-curated-capture-key-code", "Minus")
    expect(getCuratedCaptureKeyCode()).toBe("KeyC")
  })

  it("resets the curated capture shortcut back to the default key", () => {
    setCuratedCaptureKeyCode("F8")
    resetCuratedCaptureKeyCode()
    expect(getCuratedCaptureKeyCode()).toBe("KeyC")
  })
})
