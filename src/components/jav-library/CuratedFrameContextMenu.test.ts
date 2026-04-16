import { mount } from "@vue/test-utils"
import { nextTick } from "vue"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameContextMenu from "./CuratedFrameContextMenu.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("CuratedFrameContextMenu", () => {
  const frame = {
    id: "frame-1",
    movieId: "movie-1",
    title: "Sample Frame",
    code: "ABC-123",
    actors: ["Airi"],
    positionSec: 12.3,
    capturedAt: "2026-04-16T12:00:00Z",
    tags: ["closeup"],
  }

  it("emits export and delete actions from the right-click menu", async () => {
    const wrapper = mount(CuratedFrameContextMenu, {
      attachTo: document.body,
      props: {
        frame,
        x: 120,
        y: 240,
        useWebApi: true,
      },
    })

    await nextTick()

    const exportWebp = document.body.querySelector('[data-curated-frame-context-action="export-webp"]') as HTMLButtonElement | null
    const exportPng = document.body.querySelector('[data-curated-frame-context-action="export-png"]') as HTMLButtonElement | null
    const deleteButton = document.body.querySelector('[data-curated-frame-context-action="delete"]') as HTMLButtonElement | null

    expect(exportWebp).not.toBeNull()
    expect(exportPng).not.toBeNull()
    expect(deleteButton).not.toBeNull()

    exportWebp?.click()
    exportPng?.click()
    deleteButton?.click()

    expect(wrapper.emitted("exportWebp")).toHaveLength(1)
    expect(wrapper.emitted("exportPng")).toHaveLength(1)
    expect(wrapper.emitted("delete")).toHaveLength(1)

    wrapper.unmount()
  })

  it("disables export actions when Web API is unavailable", async () => {
    const wrapper = mount(CuratedFrameContextMenu, {
      attachTo: document.body,
      props: {
        frame,
        x: 0,
        y: 0,
        useWebApi: false,
      },
    })

    await nextTick()

    const exportWebp = document.body.querySelector('[data-curated-frame-context-action="export-webp"]') as HTMLButtonElement | null
    const exportPng = document.body.querySelector('[data-curated-frame-context-action="export-png"]') as HTMLButtonElement | null
    const deleteButton = document.body.querySelector('[data-curated-frame-context-action="delete"]') as HTMLButtonElement | null

    expect(exportWebp?.disabled).toBe(true)
    expect(exportPng?.disabled).toBe(true)
    expect(deleteButton?.disabled).toBe(false)

    wrapper.unmount()
  })
})
