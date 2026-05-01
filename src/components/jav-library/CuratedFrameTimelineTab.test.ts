import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameTimelineTab from "./CuratedFrameTimelineTab.vue"

vi.mock("./CuratedFrameGrid.vue", () => ({
  default: {
    name: "CuratedFrameGrid",
    props: [
      "items",
      "batchMode",
      "selectedIds",
      "nearDuplicateIds",
    ],
    emits: ["toggleSelection", "open", "contextmenu"],
    template:
      "<div data-grid><button data-toggle @click=\"$emit('toggleSelection', items[0].row.id)\">toggle</button><button data-open @click=\"$emit('open', items[0])\">open</button><button data-menu @click=\"$emit('contextmenu', $event, items[0])\">menu</button></div>",
  },
}))

const rowA = {
  id: "frame-a",
  movieId: "movie-a",
  title: "Scene A",
  code: "ABC-001",
  actors: ["Alice"],
  positionSec: 65,
  capturedAt: "2026-05-01T00:00:00.000Z",
  tags: [],
}

describe("CuratedFrameTimelineTab", () => {
  it("renders the frame grid with timeline state", () => {
    const items = [{ row: rowA, url: "blob:a" }]
    const wrapper = mount(CuratedFrameTimelineTab, {
      props: {
        items,
        batchMode: true,
        selectedIds: ["frame-a"],
        nearDuplicateIds: ["frame-b"],
      },
    })
    const grid = wrapper.findComponent({ name: "CuratedFrameGrid" })

    expect(grid.exists()).toBe(true)
    expect(grid.props("items")).toEqual(items)
    expect(grid.props("batchMode")).toBe(true)
    expect(grid.props("selectedIds")).toEqual(["frame-a"])
    expect(grid.props("nearDuplicateIds")).toEqual(["frame-b"])
  })

  it("forwards frame card events", async () => {
    const items = [{ row: rowA, url: "blob:a" }]
    const wrapper = mount(CuratedFrameTimelineTab, {
      props: {
        items,
        batchMode: false,
        selectedIds: [],
        nearDuplicateIds: [],
      },
    })

    await wrapper.get("[data-toggle]").trigger("click")
    await wrapper.get("[data-open]").trigger("click")
    await wrapper.get("[data-menu]").trigger("click")

    expect(wrapper.emitted("toggleSelection")).toEqual([["frame-a", undefined]])
    expect(wrapper.emitted("open")?.[0]?.[0]).toEqual(items[0])
    expect(wrapper.emitted("open")?.[0]?.[1]).toBeUndefined()
    expect(wrapper.emitted("contextmenu")?.[0]?.[1]).toEqual(items[0])
    expect(wrapper.emitted("contextmenu")?.[0]?.[2]).toBeUndefined()
  })
})
