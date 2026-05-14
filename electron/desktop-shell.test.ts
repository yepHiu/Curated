import path from "node:path"

import { describe, expect, it } from "vitest"

import {
  buildTrayMenuModel,
  pickDirectoryChannel,
  resolveAppIconPath,
  selectedDirectoryFromOpenDialogResult,
  shouldHideWindowOnClose,
  shouldStopBackendOnQuit,
  shouldUseApplicationMenu,
} from "./desktop-shell"

describe("Electron desktop shell integration", () => {
  it("uses the bundled Curated ico as the preferred app icon", () => {
    const appPath = "C:/repo"
    const preferredIcon = path.join(appPath, "backend", "internal", "assets", "curated.ico")

    const iconPath = resolveAppIconPath(appPath, (candidate) => candidate === preferredIcon)

    expect(iconPath).toBe(preferredIcon)
  })

  it("uses the packaged app ico when running from Electron resources", () => {
    const appPath = "C:/Program Files/Curated/resources/app"
    const packagedIcon = path.join(appPath, "curated.ico")

    const iconPath = resolveAppIconPath(appPath, (candidate) => candidate === packagedIcon)

    expect(iconPath).toBe(packagedIcon)
  })

  it("falls back to the public png icon when the Windows ico is unavailable", () => {
    const appPath = "C:/repo"
    const fallbackIcon = path.join(appPath, "public", "Curated-icon.png")

    const iconPath = resolveAppIconPath(appPath, (candidate) => candidate === fallbackIcon)

    expect(iconPath).toBe(fallbackIcon)
  })

  it("returns the first selected folder from the native directory dialog", () => {
    expect(
      selectedDirectoryFromOpenDialogResult({
        canceled: false,
        filePaths: ["D:/Media", "E:/Ignored"],
      }),
    ).toEqual({ path: "D:/Media" })
  })

  it("treats cancelled or empty native directory selections as cancelled", () => {
    expect(selectedDirectoryFromOpenDialogResult({ canceled: true, filePaths: ["D:/Media"] })).toBeNull()
    expect(selectedDirectoryFromOpenDialogResult({ canceled: false, filePaths: [] })).toBeNull()
    expect(selectedDirectoryFromOpenDialogResult({ canceled: false, filePaths: ["   "] })).toBeNull()
  })

  it("uses a namespaced IPC channel for directory picking", () => {
    expect(pickDirectoryChannel).toBe("curated:pick-directory")
  })

  it("removes the native application menu on Windows and Linux desktop shells", () => {
    expect(shouldUseApplicationMenu("win32")).toBe(false)
    expect(shouldUseApplicationMenu("linux")).toBe(false)
  })

  it("keeps the native application menu on macOS", () => {
    expect(shouldUseApplicationMenu("darwin")).toBe(true)
  })

  it("builds a polished tray menu model with app actions and normalized URLs", () => {
    expect(buildTrayMenuModel({ baseUrl: "http://127.0.0.1:8081/", attachedToExistingBackend: false })).toEqual([
      { id: "app-title", type: "normal", label: "Curated", enabled: false },
      { id: "backend-status", type: "normal", label: "Desktop service running", enabled: false },
      { type: "separator" },
      { id: "open-curated", type: "normal", label: "Open Curated" },
      {
        id: "open-browser",
        type: "normal",
        label: "Open Web App in Browser",
        url: "http://127.0.0.1:8081",
      },
      {
        id: "open-settings",
        type: "normal",
        label: "Open Settings",
        url: "http://127.0.0.1:8081/#/settings",
      },
      { type: "separator" },
      { id: "quit", type: "normal", label: "Quit Curated" },
    ])
  })

  it("labels tray status differently when Electron is attached to an existing backend", () => {
    const model = buildTrayMenuModel({
      baseUrl: "http://127.0.0.1:8081",
      attachedToExistingBackend: true,
    })

    expect(model[1]).toEqual({
      id: "backend-status",
      type: "normal",
      label: "Connected to existing backend",
      enabled: false,
    })
  })

  it("hides the window on close unless the app is intentionally quitting", () => {
    expect(shouldHideWindowOnClose({ isQuitting: false })).toBe(true)
    expect(shouldHideWindowOnClose({ isQuitting: true })).toBe(false)
  })

  it("only stops the backend on quit when Electron owns the backend process", () => {
    expect(shouldStopBackendOnQuit({ attachedToExistingBackend: false })).toBe(true)
    expect(shouldStopBackendOnQuit({ attachedToExistingBackend: true })).toBe(false)
  })
})
