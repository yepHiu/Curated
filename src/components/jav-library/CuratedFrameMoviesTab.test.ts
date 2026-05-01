import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameMoviesTab from "./CuratedFrameMoviesTab.vue"

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

const rowB = {
  ...rowA,
  id: "frame-b",
  movieId: "",
  title: "Scene B",
}

const movieGroups = [
  {
    movieKey: "movie-a",
    heading: "ABC-001",
    sub: "Scene A",
    items: [{ row: rowA, url: "blob:a" }],
  },
  {
    movieKey: "__curated_no_movie__",
    heading: "No movie",
    sub: "",
    items: [{ row: rowB, url: "blob:b" }],
  },
] as const

describe("CuratedFrameMoviesTab", () => {
  it("renders movie groups and passes state to each grid", () => {
    const wrapper = mount(CuratedFrameMoviesTab, {
      props: {
        movieGroups,
        batchMode: false,
        selectedIds: ["frame-a"],
        nearDuplicateIds: ["frame-b"],
        selectGroupAriaLabel: "Select movie group",
      },
    })
    const grids = wrapper.findAllComponents({ name: "CuratedFrameGrid" })

    expect(wrapper.text()).toContain("ABC-001")
    expect(wrapper.text()).toContain("Scene A")
    expect(wrapper.text()).toContain("No movie")
    expect(wrapper.find("input[type='checkbox']").exists()).toBe(false)
    expect(grids).toHaveLength(2)
    expect(grids[0]!.props("items")).toEqual(movieGroups[0]!.items)
    expect(grids[0]!.props("selectedIds")).toEqual(["frame-a"])
    expect(grids[0]!.props("nearDuplicateIds")).toEqual(["frame-b"])
  })

  it("forwards group checkbox and frame card events", async () => {
    const wrapper = mount(CuratedFrameMoviesTab, {
      props: {
        movieGroups,
        batchMode: true,
        selectedIds: ["frame-a"],
        nearDuplicateIds: [],
        selectGroupAriaLabel: "Select movie group",
      },
    })
    const checkboxes = wrapper.findAll("input[type='checkbox']")
    const checkboxElements = checkboxes.map((checkbox) => checkbox.element as HTMLInputElement)

    expect(checkboxElements[0]!.checked).toBe(true)
    expect(checkboxes[0]!.attributes("aria-label")).toBe("Select movie group")
    expect(checkboxElements[1]!.checked).toBe(false)

    await checkboxes[1]!.setValue(true)
    await wrapper.findAll("[data-toggle]")[0]!.trigger("click")
    await wrapper.findAll("[data-open]")[0]!.trigger("click")
    await wrapper.findAll("[data-menu]")[0]!.trigger("click")

    expect(wrapper.emitted("groupSelectionChange")?.[0]).toEqual([
      "__curated_no_movie__",
      movieGroups[1]!.items,
      true,
    ])
    expect(wrapper.emitted("toggleSelection")).toEqual([["frame-a", undefined]])
    expect(wrapper.emitted("open")?.[0]?.[0]).toEqual(movieGroups[0]!.items[0])
    expect(wrapper.emitted("open")?.[0]?.[1]).toBeUndefined()
    expect(wrapper.emitted("contextmenu")?.[0]?.[1]).toEqual(movieGroups[0]!.items[0])
    expect(wrapper.emitted("contextmenu")?.[0]?.[2]).toBeUndefined()
  })
})
