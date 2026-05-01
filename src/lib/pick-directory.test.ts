import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string, params?: Record<string, unknown>) =>
        params?.name ? `${key}:${String(params.name)}` : key,
    },
  },
}))

beforeEach(() => {
  vi.resetModules()
  delete (window as Window & { javLibrary?: unknown }).javLibrary
  delete (window as Window & { showDirectoryPicker?: unknown }).showDirectoryPicker
})

describe("pickLibraryDirectory", () => {
  it("returns a localized hint when Chromium only exposes the selected folder name", async () => {
    ;(window as Window & { showDirectoryPicker?: unknown }).showDirectoryPicker = vi.fn(
      async () => ({ name: "Movies" }) as FileSystemDirectoryHandle,
    )
    const { pickLibraryDirectory } = await import("./pick-directory")

    const outcome = await pickLibraryDirectory()

    expect(outcome).toEqual({
      status: "hint",
      suggestedTitle: "Movies",
      message: "pickDir.selected:Movies",
    })
  })
})
